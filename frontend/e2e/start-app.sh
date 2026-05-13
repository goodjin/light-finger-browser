#!/bin/bash
set -e

APP_DIR="/Users/jin/github/light-finger-browser"

echo "[E2E] Starting application for testing..."

# Kill any existing processes on relevant ports
lsof -ti :5173 | xargs kill -9 2>/dev/null || true
lsof -ti :34115 | xargs kill -9 2>/dev/null || true
lsof -ti :9222 | xargs kill -9 2>/dev/null || true
pkill -9 -f "wails dev" 2>/dev/null || true
sleep 2

# Start the full application (wails dev)
echo "[E2E] Starting wails dev..."
cd "$APP_DIR"
wails dev > /tmp/wails-dev.log 2>&1 &
WAILS_PID=$!

echo "[E2E] Wails dev started (PID: $WAILS_PID)"

# Wait for the frontend to be ready (up to 10 minutes)
echo "[E2E] Waiting for frontend to be ready..."
MAX_WAIT=600
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -sf http://localhost:5173 > /dev/null 2>&1; then
        echo "[E2E] Frontend is ready after ${WAITED}s"
        exit 0
    fi
    sleep 5
    WAITED=$((WAITED + 5))
    echo "[E2E] Still waiting... (${WAITED}s / ${MAX_WAIT}s)"
done

echo "[E2E] Timeout waiting for frontend"
cat /tmp/wails-dev.log | tail -50
exit 1
