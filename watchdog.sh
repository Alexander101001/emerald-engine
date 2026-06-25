#!/bin/bash
# Emerald Engine Watchdog — restart service if down
EMERALD_KEY=$(cat /tmp/emerald.key)
PYTHONPATH="/data/data/com.termux/files/usr/lib/python3.13/site-packages:$PYTHONPATH"
cd /tmp/opencode/emerald-engine

if ! curl -sf http://localhost:7860/health > /dev/null 2>&1; then
    kill $(lsof -t -i:7860) 2>/dev/null
    sleep 2
    EMERALD_MASTER_SECURE_KEY=$EMERALD_KEY PYTHONPATH=$PYTHONPATH nohup python3 app.py >> /var/log/emerald-engine.log 2>&1 &
    echo "$(date): Emerald Engine restarted" >> /var/log/emerald-watchdog.log
fi
