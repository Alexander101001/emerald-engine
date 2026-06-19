#!/bin/bash
BRAIN_PID=$(pgrep -f "brain.js")
UI_PID=$(pgrep -f "server.js")

if [ -z "$BRAIN_PID" ]; then
  echo "[WATCHDOG] Brain dead — restarting"
  nohup node /app/src/agi/brain.js > /tmp/emerald-brain.log 2>&1 &
fi

if [ -z "$UI_PID" ]; then
  echo "[WATCHDOG] UI dead — restarting"
  nohup node /app/src/ui/server.js > /tmp/emerald-ui.log 2>&1 &
fi
