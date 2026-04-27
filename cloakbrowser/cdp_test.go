package cloakbrowser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCDPEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19227)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	endpoint := client.GetCDPEndpoint()
	expected := "ws://localhost:19227/devtools/browser"
	if endpoint != expected {
		t.Errorf("GetCDPEndpoint() = %s, want %s", endpoint, expected)
	}
}

func TestGetWebSocketURL(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19228)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	url := client.GetWebSocketURL("tab-abc123")
	expected := "ws://localhost:19228/devtools/page/tab-abc123"
	if url != expected {
		t.Errorf("GetWebSocketURL() = %s, want %s", url, expected)
	}
}

func TestCDPTargetStruct(t *testing.T) {
	target := &CDPTarget{
		ID:        "test-id",
		Type:      "page",
		Title:     "Test Title",
		URL:       "https://example.com",
		WebSocket: "ws://localhost:9222/devtools/page/test-id",
	}

	if target.ID != "test-id" {
		t.Errorf("CDPTarget.ID = %s, want test-id", target.ID)
	}
	if target.Type != "page" {
		t.Errorf("CDPTarget.Type = %s, want page", target.Type)
	}
}
