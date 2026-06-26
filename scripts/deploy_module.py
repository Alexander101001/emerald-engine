#!/usr/bin/env python3
import os
import sys
import json
import shutil
import subprocess
import time
import logging
from pathlib import Path
from typing import Optional

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger("DeployModule")

BASE_DIR = Path("/tmp/opencode/emerald-engine")
EXTERNAL_TOOLS = BASE_DIR / "external_tools"
WORKSPACE_CORE = BASE_DIR / "workspace_core"
STATE_DIR = WORKSPACE_CORE / "state"
LOGS_DIR = STATE_DIR / "logs"
SUCCESS_FILE = STATE_DIR / "success_patterns.json"
NETWORK_NAME = "emerald_network"
COMPOSE_FRAGMENTS = BASE_DIR / "compose_fragments"

os.makedirs(EXTERNAL_TOOLS, exist_ok=True)
os.makedirs(WORKSPACE_CORE / "data", exist_ok=True)
os.makedirs(STATE_DIR, exist_ok=True)
os.makedirs(LOGS_DIR, exist_ok=True)
os.makedirs(COMPOSE_FRAGMENTS, exist_ok=True)

with open(BASE_DIR / "scripts" / "module_registry.json") as f:
    REGISTRY = json.load(f)

with open(SUCCESS_FILE, "a") as f:
    pass


def _log(message: str, level: str = "info"):
    ts = time.strftime("%Y-%m-%d %H:%M:%S")
    line = f"[{ts}] [{level.upper()}] {message}"
    log_path = LOGS_DIR / "deploy.log"
    with open(log_path, "a") as f:
        f.write(line + "\n")
    getattr(logger, level, logger.info)(message)


def _record_success(repo: str, action: str, details: dict):
    entry = {
        "timestamp": time.time(),
        "repo": repo,
        "action": action,
        "details": details,
    }
    try:
        patterns = []
        if SUCCESS_FILE.exists() and SUCCESS_FILE.stat().st_size > 0:
            with open(SUCCESS_FILE) as f:
                patterns = json.load(f)
        patterns.append(entry)
        with open(SUCCESS_FILE, "w") as f:
            json.dump(patterns, f, indent=2)
    except Exception as e:
        _log(f"Failed to record success pattern: {e}", "error")


def _record_failure(repo: str, action: str, error: str):
    entry = {
        "timestamp": time.time(),
        "repo": repo,
        "action": action,
        "error": error,
    }
    try:
        failures = []
        fail_path = STATE_DIR / "failure_log.json"
        if fail_path.exists() and fail_path.stat().st_size > 0:
            with open(fail_path) as f:
                failures = json.load(f)
        failures.append(entry)
        with open(fail_path, "w") as f:
            json.dump(failures, f, indent=2)
    except Exception as e:
        _log(f"Failed to record failure: {e}", "error")


def _ensure_network():
    result = subprocess.run(
        ["docker", "network", "ls", "--filter", f"name={NETWORK_NAME}", "--format", "{{.Name}}"],
        capture_output=True, text=True
    )
    if NETWORK_NAME not in result.stdout:
        _log(f"Creating Docker network: {NETWORK_NAME}")
        subprocess.run(
            ["docker", "network", "create", "--driver", "bridge", "--subnet", "172.29.0.0/16", NETWORK_NAME],
            check=True
        )
        _record_success("network", "created", {"name": NETWORK_NAME})
    else:
        _log(f"Network {NETWORK_NAME} already exists")


def _clone_repo(repo_full: str) -> Optional[Path]:
    owner, repo_name = repo_full.split("/")
    target = EXTERNAL_TOOLS / repo_name
    if target.exists():
        _log(f"Already cloned: {repo_full} -> {target}")
        return target
    url = f"https://github.com/{repo_full}.git"
    _log(f"Cloning {url} -> {target}")
    result = subprocess.run(
        ["git", "clone", "--depth", "1", url, str(target)],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        _log(f"Clone failed for {repo_full}: {result.stderr.strip()}", "error")
        _record_failure(repo_full, "clone", result.stderr.strip())
        return None
    _record_success(repo_full, "clone", {"path": str(target)})
    _log(f"Cloned successfully: {repo_full}")
    return target


def _generate_compose_entry(service: dict) -> str:
    repo = service["repo"]
    owner, repo_name = repo.split("/")
    container_name = f"emerald-{repo_name.lower().replace('_', '-')}"
    lines = [f"  {container_name}:"]
    if "image" in service and service["image"]:
        lines.append(f'    image: {service["image"]}')
    elif service.get("build"):
        lines.append(f"    build: {EXTERNAL_TOOLS / repo_name}")
    else:
        image_name = f"ghcr.io/{repo.lower()}:latest" if "/" not in repo else repo
        lines.append(f"    image: {image_name}")
    lines.append(f"    container_name: {container_name}")
    lines.append(f'    restart: {"unless-stopped" if service.get("internal") else "always"}')
    lines.append("    networks:")
    lines.append(f"      - {NETWORK_NAME}")
    port = service.get("port")
    if port:
        lines.append("    ports:")
        lines.append(f'      - "{port}:{port}"')
    if not service.get("no_expose"):
        resource_limits = service.get("ram_mb", 256)
        cpu_limit = service.get("cpu", 0.25)
        lines.append("    deploy:")
        lines.append("      resources:")
        lines.append("        limits:")
        lines.append(f"          memory: {resource_limits}M")
        lines.append(f'          cpus: "{cpu_limit}"')
    lines.append("    logging:")
    lines.append('      driver: "json-file"')
    lines.append("      options:")
    lines.append('        max-size: "10m"')
    lines.append('        max-file: "3"')
    lines.append("")
    return "\n".join(lines)


def _generate_docker_compose(tier_key: str, services: list) -> str:
    lines = ['version: "3.9"', "", "services:"]
    for svc in services:
        if svc.get("skip"):
            continue
        lines.append("")
        lines.append(_generate_compose_entry(svc))
    lines.append("networks:")
    lines.append(f"  {NETWORK_NAME}:")
    lines.append("    external: true")
    lines.append("")
    return "\n".join(lines)


def deploy_tier(tier_key: str):
    tier = REGISTRY["tiers"].get(tier_key)
    if not tier:
        _log(f"Unknown tier: {tier_key}", "error")
        return
    _log(f"=== Deploying {tier['label']} ({tier_key}) ===")
    services = tier["services"]
    for svc in services:
        if svc.get("skip"):
            _log(f"Skipping {svc['repo']}: {svc.get('reason', 'not suitable')}", "warn")
            continue
        repo_path = _clone_repo(svc["repo"])
        if not repo_path:
            continue
        if svc.get("pip"):
            req_file = repo_path / "requirements.txt"
            if req_file.exists():
                _log(f"Installing pip deps for {svc['repo']}")
                subprocess.run(
                    ["pip3", "install", "--quiet", "-r", str(req_file)],
                    capture_output=True, text=True
                )
    compose_yaml = _generate_docker_compose(tier_key, services)
    fragment_path = COMPOSE_FRAGMENTS / f"docker-compose.{tier_key}.yml"
    fragment_path.write_text(compose_yaml)
    _log(f"Wrote compose fragment: {fragment_path}")
    _log(f"To deploy: docker compose -f {fragment_path} up -d")
    _record_success(tier_key, "compose_generated", {"path": str(fragment_path), "services": len(services)})


def deploy_all():
    _ensure_network()
    for tier_key in ["core_infrastructure", "logic_layer", "interface_layer"]:
        deploy_tier(tier_key)
        _log(f"Completed {tier_key}, ready for next module")


def status_report():
    report = {
        "network": NETWORK_NAME,
        "external_tools": [str(p) for p in EXTERNAL_TOOLS.iterdir() if p.is_dir()],
        "compose_fragments": [str(p) for p in COMPOSE_FRAGMENTS.glob("*.yml")],
        "state_dir": str(STATE_DIR),
        "success_patterns_count": 0,
        "failure_count": 0,
    }
    if SUCCESS_FILE.exists() and SUCCESS_FILE.stat().st_size > 0:
        with open(SUCCESS_FILE) as f:
            try:
                report["success_patterns_count"] = len(json.load(f))
            except Exception:
                pass
    fail_path = STATE_DIR / "failure_log.json"
    if fail_path.exists() and fail_path.stat().st_size > 0:
        with open(fail_path) as f:
            try:
                report["failure_count"] = len(json.load(f))
            except Exception:
                pass
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "all"
    if cmd == "all":
        deploy_all()
    elif cmd == "core":
        _ensure_network()
        deploy_tier("core_infrastructure")
    elif cmd == "logic":
        _ensure_network()
        deploy_tier("logic_layer")
    elif cmd == "interface":
        _ensure_network()
        deploy_tier("interface_layer")
    elif cmd == "status":
        status_report()
    else:
        _ensure_network()
        deploy_tier(cmd)
