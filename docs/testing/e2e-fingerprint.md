# Fingerprint Server E2E Test

## Prerequisites

1. Chrome browser running with remote debugging enabled on port 9222
2. Python fingerprint server running on port 18080
3. Go test tool at `/tmp/test_cdp_connect.go`

## Test 1: Python Fingerprint Server

### Start the server
```bash
python3 cmd/fingerprint-server-py/server.py &
```

### Test server endpoints
```bash
# Health check
curl http://localhost:18080/health

# Get fingerprint page
curl http://localhost:18080/

# Submit fingerprint
curl -X POST http://localhost:18080/api/fingerprint \
  -H "Content-Type: application/json" \
  -d '{"canvas":"hash123","webgl":"hash456"}'
```

## Test 2: CDP Connection

### Start Chrome with debugging
```bash
"/path/to/Chromium.app/Contents/MacOS/Chromium" \
  --remote-debugging-port=9222 \
  --user-data-dir=/tmp/chrome-test
```

### Test CDP connection
```bash
# Build the test tool
go build -o /tmp/test_cdp_connect /tmp/test_cdp_connect.go

# Test with /json endpoint (auto-discovers WebSocket URL)
go run /tmp/test_cdp_connect.go http://localhost:9222/json

# Expected output:
# Testing endpoint: http://localhost:9222/json
# Found WebSocket URL: ws://localhost:9222/devtools/page/...
# Connected successfully!
# Test passed!
```

## Test 3: Navigate Instance Browser

### Test script
```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 1. Get WebSocket URL from /json
	resp, _ := http.Get("http://localhost:9222/json")
	defer resp.Body.Close()

	// Parse JSON and extract webSocketDebuggerUrl...

	// 2. Connect via WebSocket
	wsURL := "ws://localhost:9222/devtools/page/..."
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, _ := dialer.DialContext(context.Background(), wsURL, nil)
	defer conn.Close()

	// 3. Navigate to fingerprint test page
	msg := map[string]interface{}{
		"id":     1,
		"method": "Page.navigate",
		"params": map[string]interface{}{
			"url": "http://localhost:18080/",
		},
	}
	conn.WriteJSON(msg)

	// 4. Wait for page load
	var respData map[string]interface{}
	conn.ReadJSON(&respData)

	fmt.Println("Navigation successful!")
}
```

## Test 4: Full Integration (App → Fingerprint Server)

### Manual test steps

1. **Start app**
   ```bash
   open build/bin/fingerbrower.app
   ```

2. **Start fingerprint server** (if not running)
   ```bash
   python3 cmd/fingerprint-server-py/server.py &
   ```

3. **Create instance** via UI or API

4. **Test Fingerprint button** - should:
   - Connect to instance's CDP endpoint
   - Navigate to `http://localhost:18080/`
   - Browser shows fingerprint test page

### Debug commands

```bash
# Check if fingerprint server is running
curl http://localhost:18080/health

# Check Chrome debugging port
curl http://localhost:9222/json

# View app logs
tail -f ~/Library/Logs/fingerbrower/fingerbrower.log
```

## Test 5: CDP Query (Current Instance State)

### Query instance via CDP
```go
// Get all tabs/targets
msg := map[string]interface{}{
    "id":     1,
    "method": "Target.getTargets",
}
conn.WriteJSON(msg)
var targets map[string]interface{}
conn.ReadJSON(&targets)
fmt.Printf("Targets: %v\n", targets)
```

## Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| `websocket: bad handshake` | Wrong WebSocket URL | Use URL from `/json`, not just port |
| `connection refused` | Chrome not running | Start Chrome with `--remote-debugging-port=9222` |
| `instance not running` | Instance status not "running" | Check instance status in database |
| `failed to query CDP targets` | Port mismatch | Verify instance.CDPEndpoint matches Chrome port |
