#!/bin/bash
# Emerald Engine: Cleanup Protocol
echo "[CLEANUP] Sanitizing workspace..."
rm -rf logs/* 2>/dev/null
rm -rf temp/* 2>/dev/null
rm -f core.* 2>/dev/null
find . -name "*.tmp" -type f -delete 2>/dev/null
find . -name ".DS_Store" -type f -delete 2>/dev/null
echo "[CLEANUP] System sanitized."
