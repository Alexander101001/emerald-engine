#!/bin/bash
# Start Emerald Engine with built-in watchdog
EMERALD_KEY=$(cat /tmp/emerald.key)
PYTHONPATH="/data/data/com.termux/files/usr/lib/python3.13/site-packages:$PYTHONPATH"
cd /tmp/opencode/emerald-engine

export EMERALD_MASTER_SECURE_KEY=$EMERALD_KEY
export PYTHONPATH

python3 app.py &
APP_PID=$!
echo "Emerald Engine started (PID: $APP_PID)"

# Simple watchdog loop
while true; do
    sleep 30
    if ! kill -0 $APP_PID 2>/dev/null; then
        echo "$(date): App crashed, restarting..." >> /var/log/emerald-watchdog.log
        python3 app.py &
        APP_PID=$!
        echo "Restarted with PID: $APP_PID" >> /var/log/emerald-watchdog.log
    fi
done
