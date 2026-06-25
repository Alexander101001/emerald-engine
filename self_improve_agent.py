import asyncio
import json
import logging
import os
import subprocess
import sys
from pathlib import Path

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

MASTER_KEY = os.getenv("EMERALD_MASTER_SECURE_KEY", "")
REPO_PATH = Path("/app") if os.path.exists("/app") else Path(".")
TELEMETRY_FILE = Path("/tmp/telemetry.json")


async def call_llm(prompt: str, model: str = "qwen2.5:0.5b") -> str:
    try:
        proc = await asyncio.create_subprocess_exec(
            "ollama", "run", model,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate(input=prompt.encode(), timeout=30)
        return stdout.decode().strip()
    except Exception as e:
        logging.warning(f"Ollama call failed: {e}")
        return ""


async def fetch_commit_history(count: int = 10) -> str:
    try:
        proc = await asyncio.create_subprocess_exec(
            "git", "log", f"-{count}", "--oneline",
            cwd=REPO_PATH,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await proc.communicate(timeout=10)
        return stdout.decode()
    except Exception as e:
        return f"git log error: {e}"


async def read_recent_files(extensions=(".py", ".js", ".go"), max_lines: int = 2000) -> str:
    """Read up to max_lines of code from recent files."""
    result = []
    total = 0
    for ext in extensions:
        if total >= max_lines:
            break
        for f in sorted(Path(REPO_PATH).rglob(f"*{ext}")):
            if ".git" in str(f) or ".aegis" in str(f) or "__pycache__" in str(f):
                continue
            try:
                content = f.read_text()
                lines = content.split("\n")
                header = f"\n--- {f.relative_to(REPO_PATH)} ({len(lines)} lines) ---\n"
                result.append(header)
                total += len(lines) + 1
                result.append(content)
            except Exception:
                continue
    return "".join(result[:max_lines])


def parse_suggestion(llm_output: str):
    """Parse LLM output into structured suggestions."""
    output = llm_output.strip()
    suggestions = []
    current = {"file": "", "issue": "", "fix": "", "priority": "medium"}

    for line in output.split("\n"):
        line = line.strip()
        ll = line.lower()
        if ll.startswith("file:") or line.startswith("## File"):
            if current["file"] or current["issue"]:
                suggestions.append(current)
                current = {"file": "", "issue": "", "fix": "", "priority": "medium"}
            current["file"] = line.split(":", 1)[-1].strip() if ":" in line else line.replace("##", "").strip()
        elif ll.startswith("issue:") or ll.startswith("problem:"):
            current["issue"] = line.split(":", 1)[-1].strip()
        elif ll.startswith("fix:") or ll.startswith("suggestion:"):
            current["fix"] = line.split(":", 1)[-1].strip()
        elif ll.startswith("priority:"):
            current["priority"] = line.split(":", 1)[-1].strip().lower()

    if current["file"] or current["issue"]:
        suggestions.append(current)

    # Fallback: treat whole output as single suggestion
    if not suggestions:
        suggestions.append({
            "file": "unknown",
            "issue": output[:200],
            "fix": output,
            "priority": "medium",
        })

    return suggestions


async def create_pr_branch(suggestions: list[dict]) -> str:
    """Create a branch with auto-fixes and return the branch name."""
    branch = f"auto-improve/{int(__import__('time').time())}"
    try:
        proc = await asyncio.create_subprocess_exec(
            "git", "checkout", "-b", branch,
            cwd=REPO_PATH,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        await proc.communicate(timeout=10)
    except Exception as e:
        logging.error(f"Branch creation failed: {e}")
        return ""

    changed = False
    for s in suggestions:
        if s["file"] and s["fix"] and s["fix"] != s.get("issue", ""):
            file_path = REPO_PATH / s["file"]
            if file_path.exists():
                try:
                    original = file_path.read_text()
                    new_content = original.replace(s["issue"][:50], s["fix"][:50])
                    if new_content != original:
                        file_path.write_text(new_content)
                        changed = True
                        logging.info(f"Patched {s['file']}")
                except Exception as e:
                    logging.warning(f"Could not patch {s['file']}: {e}")

    if not changed:
        try:
            proc = await asyncio.create_subprocess_exec(
                "git", "checkout", "main",
                cwd=REPO_PATH,
                stdout=asyncio.subprocess.PIPE,
            )
            await proc.communicate(timeout=10)
            proc = await asyncio.create_subprocess_exec(
                "git", "branch", "-D", branch,
                cwd=REPO_PATH,
            )
            await proc.communicate(timeout=10)
        except Exception:
            pass
        return ""

    try:
        proc = await asyncio.create_subprocess_exec(
            "git", "add", ".",
            cwd=REPO_PATH,
        )
        await proc.communicate(timeout=10)
        msg = f"auto: self-improvement patch from telemetry analysis"
        proc = await asyncio.create_subprocess_exec(
            "git", "commit", "-m", msg,
            cwd=REPO_PATH,
            env={**os.environ, "GIT_AUTHOR_NAME": "Emerald Self-Improve",
                 "GIT_AUTHOR_EMAIL": "self-improve@emerald.app",
                 "GIT_COMMITTER_NAME": "Emerald Self-Improve",
                 "GIT_COMMITTER_EMAIL": "self-improve@emerald.app"},
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        await proc.communicate(timeout=10)
    except Exception as e:
        logging.error(f"Commit failed: {e}")
        return ""

    # Push branch
    try:
        proc = await asyncio.create_subprocess_exec(
            "git", "push", "origin", branch,
            cwd=REPO_PATH,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        await proc.communicate(timeout=30)
    except Exception as e:
        logging.error(f"Push failed: {e}")
        return ""

    return branch


async def run_self_improve_cycle():
    logging.info("=== Self-Improvement Cycle Start ===")

    telemetry = {}
    if TELEMETRY_FILE.exists():
        telemetry = json.loads(TELEMETRY_FILE.read_text())

    commits = await fetch_commit_history(15)
    codebase = await read_recent_files(max_lines=3000)

    prompt = f"""You are a senior DevOps engineer optimizing a SaaS platform.

TELEMETRY:
{json.dumps(telemetry, indent=2)}

RECENT COMMITS:
{commits}

Analyze the telemetry and codebase for performance bottlenecks, 
resource leaks, or improvements. For each issue found, output:

File: <path>
Issue: <description>
Fix: <code fix or suggestion>
Priority: <high/medium/low>

Focus on:
- CPU/memory optimization
- Async non-blocking patterns
- Reducing disk I/O
- Removing dead code
- Adding error handling
- Streaming vs batch tradeoffs

Codebase snapshot:
{codebase[:5000]}
"""

    llm_output = await call_llm(prompt)
    if not llm_output:
        logging.warning("LLM returned empty, skipping cycle")
        return

    suggestions = parse_suggestion(llm_output)
    logging.info(f"Parsed {len(suggestions)} suggestions")

    branch = await create_pr_branch(suggestions)
    if branch:
        logging.info(f"Created PR branch: {branch}")
    else:
        logging.info("No changes to commit — system healthy")

    logging.info("=== Self-Improvement Cycle Complete ===")


async def self_improve_loop(interval: int = 3600):
    while True:
        try:
            await run_self_improve_cycle()
        except Exception as e:
            logging.error(f"Self-improve cycle failed: {e}")
        logging.info(f"Sleeping {interval}s until next cycle...")
        await asyncio.sleep(interval)


if __name__ == "__main__":
    asyncio.run(run_self_improve_cycle())
