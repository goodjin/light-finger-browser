package proxy

import (
	"testing"
)

// TestProxyStatusPtr tests ProxyStatusPtr helper function
func TestProxyStatusPtr(t *testing.T) {
	status := ProxyStatusAvailable
	ptr := ProxyStatusPtr(status)

	if ptr == nil {
		t.Fatal("ProxyStatusPtr returned nil")
	}

	if *ptr != ProxyStatusAvailable {
		t.Errorf("Expected 'available', got '%s'", *ptr)
	}
}

// TestProxyTypePtr tests ProxyTypePtr helper function
func TestProxyTypePtr(t *testing.T) {
	ptype := ProxyTypeResidential
	ptr := ProxyTypePtr(ptype)

	if ptr == nil {
		t.Fatal("ProxyTypePtr returned nil")
	}

	if *ptr != ProxyTypeResidential {
		t.Errorf("Expected 'residential', got '%s'", *ptr)
	}
}

// TestProxyTypeConstants tests proxy type constants
func TestProxyTypeConstants(t *testing.T) {
	if ProxyTypeResidential != "residential" {
		t.Errorf("Expected 'residential', got '%s'", ProxyTypeResidential)
	}

	if ProxyTypeDatacenter != "datacenter" {
		t.Errorf("Expected 'datacenter', got '%s'", ProxyTypeDatacenter)
	}
}

// TestProxyStatusConstants tests proxy status constants
func TestProxyStatusConstants(t *testing.T) {
	if ProxyStatusAvailable != "available" {
		t.Errorf("Expected 'available', got '%s'", ProxyStatusAvailable)
	}

	if ProxyStatusInUse != "in_use" {
		t.Errorf("Expected 'in_use', got '%s'", ProxyStatusInUse)
	}

	if ProxyStatusChecking != "checking" {
		t.Errorf("Expected 'checking', got '%s'", ProxyStatusChecking)
	}

	if ProxyStatusDead != "dead" {
		t.Errorf("Expected 'dead', got '%s'", ProxyStatusDead)
	}
}