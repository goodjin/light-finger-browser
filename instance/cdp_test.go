package instance

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCDPMessageFormat(t *testing.T) {
	msg := CDPMessage{
		ID:     1,
		Method: "Page.navigate",
		Params: map[string]interface{}{"url": "https://example.com"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal CDP message: %v", err)
	}

	var decoded CDPMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CDP message: %v", err)
	}

	if decoded.ID != 1 {
		t.Errorf("expected ID 1, got %d", decoded.ID)
	}

	if decoded.Method != "Page.navigate" {
		t.Errorf("expected method Page.navigate, got %s", decoded.Method)
	}

	url, ok := decoded.Params["url"].(string)
	if !ok || url != "https://example.com" {
		t.Errorf("expected url param https://example.com, got %v", decoded.Params["url"])
	}
}

func TestCDPResponseFormat(t *testing.T) {
	resp := CDPResponse{
		ID:     1,
		Result: json.RawMessage(`{"success": true}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal CDP response: %v", err)
	}

	var decoded CDPResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CDP response: %v", err)
	}

	if decoded.ID != 1 {
		t.Errorf("expected ID 1, got %d", decoded.ID)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decoded.Result, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["success"] != true {
		t.Errorf("expected success true, got %v", result["success"])
	}
}

func TestCDPErrorFormat(t *testing.T) {
	resp := CDPResponse{
		ID: 1,
		Error: &CDPError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal CDP response: %v", err)
	}

	var decoded CDPResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CDP response: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("expected error to be present")
	}

	if decoded.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", decoded.Error.Code)
	}

	if decoded.Error.Message != "Invalid Request" {
		t.Errorf("expected error message 'Invalid Request', got %s", decoded.Error.Message)
	}
}

func TestCDPClient_Screenshot_EmptyResponse(t *testing.T) {
	// Test that Screenshot handles empty result
	client := &CDPClient{}

	// Call Screenshot with a context that will timeout immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	_, err := client.Screenshot(ctx)
	if err == nil {
		t.Error("expected error when connection is nil")
	}
}

func TestCDPClient_Evaluate_EmptyResponse(t *testing.T) {
	client := &CDPClient{}

	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	_, err := client.Evaluate(ctx, "document.title")
	if err == nil {
		t.Error("expected error when connection is nil")
	}
}

func TestCDPClient_Click(t *testing.T) {
	client := &CDPClient{}

	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	err := client.Click(ctx, "#button")
	if err == nil {
		t.Error("expected error when connection is nil")
	}
}

func TestCDPClient_Type(t *testing.T) {
	client := &CDPClient{}

	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	err := client.Type(ctx, "#input", "hello")
	if err == nil {
		t.Error("expected error when connection is nil")
	}
}

func TestNewCDPClient(t *testing.T) {
	conn := &Conn{}
	client := NewCDPClient(conn)

	if client == nil {
		t.Error("expected non-nil client")
	}

	if client.conn != conn {
		t.Error("expected conn to be set")
	}

	if client.msgID != 0 {
		t.Errorf("expected msgID to be 0, got %d", client.msgID)
	}
}

func TestConnectCDP_InvalidEndpoint(t *testing.T) {
	ctx := context.Background()

	_, err := ConnectCDP(ctx, "ws://invalid-host:9999/nonexistent")
	if err == nil {
		t.Error("expected error when connecting to invalid endpoint")
	}
}

func TestCDPClient_Close(t *testing.T) {
	client := &CDPClient{}

	// Should not panic when conn is nil
	err := client.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCDPClient_MultipleExecute(t *testing.T) {
	client := &CDPClient{}

	ctx := context.Background()

	// Execute multiple times to test message ID incrementing
	client.mu.Lock()
	id1 := client.msgID
	client.mu.Unlock()

	// Try to execute (will fail since conn is nil)
	_, _ = client.execute(ctx, "Test.method", nil)

	client.mu.Lock()
	id2 := client.msgID
	client.mu.Unlock()

	if id2 <= id1 {
		t.Errorf("expected msgID to increment, got %d then %d", id1, id2)
	}
}

func TestWebSocketConn(t *testing.T) {
	conn := &Conn{}

	// Test Close
	err := conn.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}

	// Test IsClosed
	if !conn.IsClosed() {
		t.Error("expected connection to be closed")
	}
}

func TestFormatAddr(t *testing.T) {
	addr := FormatAddr("localhost", 9222)
	if addr != "localhost:9222" {
		t.Errorf("expected localhost:9222, got %s", addr)
	}
}