package bootstrap

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestIsValidDBName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid simple", "pkl_staging", true},
		{"Valid development", "pkl_development", true},
		{"Valid alphanumeric", "pkl_test123", true},
		{"Invalid with spaces", "pkl test", false},
		{"Invalid with dashes", "pkl-staging", false},
		{"Invalid with quotes", "pkl'; DROP DATABASE;", false},
		{"Invalid with semicolon", "pkl_db;--", false},
		{"Invalid empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidDBName(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidDBName(%q) = %v, expected %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWaitForTCP_Success(t *testing.T) {
	// Start a local dummy TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer listener.Close()

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse listener addr: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = WaitForTCP(ctx, "127.0.0.1", portStr, 3*time.Second)
	if err != nil {
		t.Errorf("expected WaitForTCP to succeed, got %v", err)
	}
}

func TestWaitForTCP_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Pick an improbable port that won't connect
	err := WaitForTCP(ctx, "127.0.0.1", "59999", 500*time.Millisecond)
	if err == nil {
		t.Error("expected WaitForTCP to fail due to timeout, got nil")
	}
}
