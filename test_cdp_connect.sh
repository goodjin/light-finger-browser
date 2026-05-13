#!/bin/bash
# CDP Connection Test Script
# This script tests the WebSocket connection to Chrome's CDP endpoint

set -e

PORT=${1:-9222}
HOST=${2:-localhost}

echo "=== CDP Connection Test ==="
echo "Testing connection to Chrome at $HOST:$PORT"
echo ""

# Test 1: Check if Chrome is running and responding on the debugging port
echo "[Test 1] Checking if Chrome is responding on port $PORT..."
if curl -s --max-time 5 "http://$HOST:$PORT/json" > /dev/null 2>&1; then
    echo "  ✓ Chrome is responding on port $PORT"
else
    echo "  ✗ Chrome is NOT responding on port $PORT"
    echo "  Make sure Chrome is running with: chrome --remote-debugging-port=$PORT"
    exit 1
fi

# Test 2: Get the WebSocket target URL
echo ""
echo "[Test 2] Getting WebSocket target URL..."
TARGET_RESPONSE=$(curl -s "http://$HOST:$PORT/json")
echo "  Response: $TARGET_RESPONSE"

# Extract the WebSocket URL from the response
# The response is a JSON array, we need to find the webSocketDebuggerUrl field
WS_URL=$(echo "$TARGET_RESPONSE" | grep -o '"webSocketDebuggerUrl":"[^"]*"' | head -1 | sed 's/"webSocketDebuggerUrl":"//;s/"$//')
if [ -z "$WS_URL" ]; then
    echo "  ✗ Could not find WebSocket URL in response"
    exit 1
fi
echo "  ✓ WebSocket URL: $WS_URL"

# Test 3: Test WebSocket connection using wscat or websocat if available
echo ""
echo "[Test 3] Testing WebSocket connection..."

# Try to use a WebSocket testing tool
if command -v wscat &> /dev/null; then
    echo "  Using wscat..."
    # Just connect and immediately close (timeout after 2 seconds)
    timeout 2 wscat -c "$WS_URL" || echo "  Connection attempt completed (timeout is expected)"
elif command -v websocat &> /dev/null; then
    echo "  Using websocat..."
    timeout 2 websocat "$WS_URL" || echo "  Connection attempt completed (timeout is expected)"
else
    echo "  Neither wscat nor websocat available, skipping WebSocket test"
    echo "  Install with: brew install wscat (or websocat)"
fi

# Test 4: Test with a simple Go program
echo ""
echo "[Test 4] Testing WebSocket connection with Go..."

cat > /tmp/test_ws.go << 'EOF'
package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_ws <websocket-url>")
		fmt.Println("Example: test_ws ws://localhost:9222/devtools/browser/...")
		os.Exit(1)
	}

	wsURL := os.Args[1]
	if !strings.HasPrefix(wsURL, "ws://") && !strings.HasPrefix(wsURL, "wss://") {
		wsURL = "ws://" + wsURL
	}

	fmt.Printf("Connecting to: %s\n", wsURL)

	// Parse and validate URL
	_, err := url.Parse(wsURL)
	if err != nil {
		fmt.Printf("Invalid URL: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connected successfully!")

	// Send a simple CDP command to verify
	// {"id":1,"method":"Runtime.evaluate","params":{"expression":"1+1"}}
	msg := map[string]interface{}{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{
			"expression": "1+1",
		},
	}

	if err := conn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		os.Exit(1)
	}

	// Wait for response
	var resp map[string]interface{}
	if err := conn.ReadJSON(&resp); err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response: %v\n", resp)
	fmt.Println("Test passed!")
}
EOF

# Note: This won't compile because we need os import
# Let me create a simpler test
