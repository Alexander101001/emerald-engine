import os
import re
import sys
import secrets
import string
from pathlib import Path

SENSITIVE_PATTERNS = {
    r'ghp_[A-Za-z0-9]{36}': 'GITHUB_TOKEN_OBFUSCATED',
    r'hf_[A-Za-z0-9]{36}': 'HF_TOKEN_OBFUSCATED',
    r'sk-[A-Za-z0-9]{20,}': 'API_KEY_OBFUSCATED',
    r'AKIA[0-9A-Z]{16}': 'AWS_KEY_OBFUSCATED',
    r'(?i)master.?key\s*=\s*["\'][A-Fa-f0-9]{64}': 'MASTER_KEY_OBFUSCATED',
    r'(?i)password\s*[:=]\s*["\'][^"\']{8,}': 'PASSWORD_OBFUSCATED',
    r'(?i)secret\s*[:=]\s*["\'][^"\']{8,}': 'SECRET_OBFUSCATED',
}

SKIP_DIRS = {'.git', 'node_modules', '__pycache__', '.aegis', '.secrets', '.vault'}
SKIP_EXTS = {'.pyc', '.encrypted', '.db', '.png', '.jpg', '.svg', '.ico', '.woff', '.woff2'}

def generate_obfuscated_name(original: str) -> str:
    if original.startswith("EMERALD_"):
        parts = original.split("_", 1)
        suffix = ''.join(secrets.choice(string.ascii_uppercase + string.digits) for _ in range(12))
        return f"{parts[0]}_{suffix}"
    prefix = ''.join(secrets.choice(string.ascii_uppercase) for _ in range(4))
    suffix = ''.join(secrets.choice(string.ascii_uppercase + string.digits) for _ in range(12))
    return f"EM_{prefix}_{suffix}"

def find_sensitive_values(file_path: Path) -> dict:
    findings = {}
    try:
        content = file_path.read_text(errors="replace")
        for pattern, label in SENSITIVE_PATTERNS.items():
            matches = re.findall(pattern, content)
            for m in matches:
                findings[m] = label
    except Exception:
        pass
    return findings

def obfuscate_file(file_path: Path, mapping: dict, dry_run: bool = False) -> bool:
    try:
        content = file_path.read_text(errors="replace")
        modified = content
        for original, replacement in mapping.items():
            modified = modified.replace(original, replacement)
        if modified != content:
            if not dry_run:
                file_path.write_text(modified)
            return True
    except Exception:
        pass
    return False

def scan_repository(root: Path, dry_run: bool = False) -> dict:
    results = {
        "scanned_files": 0,
        "sensitive_findings": 0,
        "files_obfuscated": 0,
        "obfuscation_map": {},
        "file_details": [],
    }
    for file_path in root.rglob("*"):
        if file_path.is_dir():
            continue
        rel = file_path.relative_to(root)
        parts = rel.parts
        if any(skip in parts for skip in SKIP_DIRS):
            continue
        if file_path.suffix in SKIP_EXTS:
            continue
        if file_path.name.startswith(".") and file_path.name not in (".env", ".env.example"):
            continue
        results["scanned_files"] += 1
        findings = find_sensitive_values(file_path)
        if findings:
            results["sensitive_findings"] += len(findings)
            obf_map = {}
            for original, label in findings.items():
                obfuscated = generate_obfuscated_name(label)
                obf_map[original] = obfuscated
                results["obfuscation_map"][original] = obfuscated
            if obfuscate_file(file_path, obf_map, dry_run):
                results["files_obfuscated"] += 1
                results["file_details"].append({
                    "file": str(rel),
                    "replacements": list(obf_map.keys()),
                })
    return results

def main():
    import argparse
    parser = argparse.ArgumentParser(description="Emerald Obfuscation Engine")
    parser.add_argument("--root", default=str(Path.cwd()), help="Repository root path")
    parser.add_argument("--dry-run", action="store_true", help="Scan without modifying files")
    parser.add_argument("--output-map", default=None, help="Save obfuscation map to file")
    parser.add_argument("--restore", default=None, help="Restore original values from map file")
    args = parser.parse_args()

    root = Path(args.root).resolve()

    if args.restore:
        import json
        with open(args.restore) as f:
            restore_map = json.load(f)
        print(f"Restoring {len(restore_map)} values from {args.restore}")
        for file_path in root.rglob("*"):
            if file_path.is_dir():
                continue
            rel = file_path.relative_to(root)
            parts = rel.parts
            if any(skip in parts for skip in SKIP_DIRS):
                continue
            if file_path.suffix in SKIP_EXTS:
                continue
            try:
                content = file_path.read_text(errors="replace")
                modified = content
                for obfuscated, original in restore_map.items():
                    modified = modified.replace(obfuscated, original)
                if modified != content:
                    file_path.write_text(modified)
                    print(f"  Restored: {rel}")
            except Exception:
                pass
        return

    results = scan_repository(root, dry_run=args.dry_run)
    print(f"\n=== Emerald Obfuscation Report ===")
    print(f"Scanned files:      {results['scanned_files']}")
    print(f"Sensitive findings: {results['sensitive_findings']}")
    print(f"Files obfuscated:   {results['files_obfuscated']}")
    print(f"\nObfuscation map ({len(results['obfuscation_map'])} entries):")
    for original, obfuscated in results['obfuscation_map'].items():
        print(f"  {original[:30]}... -> {obfuscated}")

    if args.output_map:
        import json
        output = {}
        for original, obfuscated in results['obfuscation_map'].items():
            output[obfuscated] = original
        with open(args.output_map, "w") as f:
            json.dump(output, f, indent=2)
        print(f"\nRestore map saved to: {args.output_map}")

    if results["obfuscation_map"] and not args.dry_run:
        print("\n⚠️  Obfuscation complete. Update your .env with the new mapping.")

if __name__ == "__main__":
    main()
