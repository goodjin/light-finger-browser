package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("CDP Connection Test")
		fmt.Println("Usage: test_cdp_connect <endpoint>")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  test_cdp_connect localhost:9222")
		fmt.Println("  test_cdp_connect ws://localhost:9222")
		fmt.Println("  test_cdp_connect http://localhost:9222/json")
		os.Exit(1)
	}

	endpoint := os.Args[1]
	fmt.Printf("Testing endpoint: %s\n", endpoint)

	// If it's a JSON endpoint, first fetch the WebSocket URL
	if strings.Contains(endpoint, "/json") {
		fmt.Println("Fetching WebSocket URL from /json...")
		resp, err := http.Get(endpoint)
		if err != nil {
			fmt.Printf("Failed to fetch /json: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		// Read response
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		body := string(buf[:n])
		fmt.Printf("Response: %s\n", body)

		// Extract webSocketDebuggerUrl - simple parsing
		wsURL := extractWebSocketURL(body)
		if wsURL == "" {
			fmt.Println("No WebSocket URL found in response")
			os.Exit(1)
		}
		fmt.Printf("Found WebSocket URL: %s\n", wsURL)
		endpoint = wsURL
	}

	// Normalize to WebSocket URL
	wsURL := endpoint
	if !strings.HasPrefix(wsURL, "ws://") && !strings.HasPrefix(wsURL, "wss://") {
		if strings.HasPrefix(wsURL, "http://") {
			wsURL = "ws://" + wsURL[7:]
		} else if strings.HasPrefix(wsURL, "https://") {
			wsURL = "wss://" + wsURL[8:]
		} else {
			wsURL = "ws://" + wsURL
		}
	}

	// Parse URL
	parsed, err := url.Parse(wsURL)
	if err != nil {
		fmt.Printf("Invalid URL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed URL: scheme=%s host=%s path=%s\n", parsed.Scheme, parsed.Host, parsed.Path)

	// Connect
	fmt.Println("\nAttempting WebSocket connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		if resp != nil {
			fmt.Printf("Response status: %s\n", resp.Status)
		}
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connected successfully!")

	// Send a simple CDP command
	fmt.Println("\nSending CDP command: Runtime.evaluate(1+1)")
	msg := map[string]interface{}{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]interface{}{
			"expression": "1+1",
		},
	}

	if err := conn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send: %v\n", err)
		os.Exit(1)
	}

	// Read response
	fmt.Println("Waiting for response...")
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var respData map[string]interface{}
	if err := conn.ReadJSON(&respData); err != nil {
		fmt.Printf("Failed to read: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Response: %v\n", respData)
	fmt.Println("\n✓ Test passed!")
}

func extractWebSocketURL(body string) string {
	// Look for the exact pattern "webSocketDebuggerUrl":"
	searchPattern := `"webSocketDebuggerUrl": "`
	start := strings.Index(body, searchPattern)
	if start == -1 {
		// Try with different quote style
		searchPattern = `"webSocketDebuggerUrl":"`
		start = strings.Index(body, searchPattern)
		if start == -1 {
			return ""
		}
		start += len(searchPattern)
	} else {
		start += len(searchPattern)
	}

	// Find the end quote
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		return ""
	}

	return body[start : start+end]
}
