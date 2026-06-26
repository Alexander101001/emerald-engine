#!/usr/bin/env python3
import json
import time
import sys
import os
from pathlib import Path

BASE_DIR = Path("/tmp/opencode/emerald-engine")
STATE_DIR = BASE_DIR / "workspace_core" / "state"
SUCCESS_FILE = STATE_DIR / "success_patterns.json"
FAILURE_FILE = STATE_DIR / "failure_log.json"
os.makedirs(STATE_DIR, exist_ok=True)


def record(repo: str, action: str, status: str, details: dict = None):
    entry = {
        "timestamp": time.time(),
        "repo": repo,
        "action": action,
        "status": status,
        "details": details or {},
    }
    target = SUCCESS_FILE if status == "success" else FAILURE_FILE
    records = []
    if target.exists() and target.stat().st_size > 0:
        try:
            with open(target) as f:
                records = json.load(f)
        except (json.JSONDecodeError, ValueError):
            records = []
    records.append(entry)
    with open(target, "w") as f:
        json.dump(records, f, indent=2)
    return entry


def summarize():
    patterns = {"success": 0, "failure": 0, "total": 0, "by_repo": {}}
    for fname, key in [(SUCCESS_FILE, "success"), (FAILURE_FILE, "failure")]:
        if fname.exists() and fname.stat().st_size > 0:
            try:
                with open(fname) as f:
                    records = json.load(f)
                patterns[key] = len(records)
                patterns["total"] += len(records)
                for r in records:
                    repo = r.get("repo", "unknown")
                    if repo not in patterns["by_repo"]:
                        patterns["by_repo"][repo] = {"success": 0, "failure": 0}
                    patterns["by_repo"][repo][key] += 1
            except Exception:
                pass
    return patterns


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "summarize"
    if cmd == "summarize":
        print(json.dumps(summarize(), indent=2))
    elif cmd == "record":
        if len(sys.argv) >= 4:
            entry = record(sys.argv[2], sys.argv[3], sys.argv[4] if len(sys.argv) > 4 else "success")
            print(json.dumps(entry))
