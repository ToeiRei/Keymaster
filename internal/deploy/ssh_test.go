// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"errors"
	"testing"
)

func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.ConnectionTimeout != DefaultConnectionTimeout {
		t.Errorf("Expected ConnectionTimeout %v, got %v", DefaultConnectionTimeout, config.ConnectionTimeout)
	}

	if config.CommandTimeout != DefaultCommandTimeout {
		t.Errorf("Expected CommandTimeout %v, got %v", DefaultCommandTimeout, config.CommandTimeout)
	}

	if config.SFTPTimeout != DefaultSFTPTimeout {
		t.Errorf("Expected SFTPTimeout %v, got %v", DefaultSFTPTimeout, config.SFTPTimeout)
	}
}

func TestIsConnectionTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("connection timeout"), true},
		{"deadline exceeded", errors.New("deadline exceeded"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"other error", errors.New("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionTimeoutError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsConnectionRefusedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"no route to host", errors.New("no route to host"), true},
		{"other error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionRefusedError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionRefusedError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"authentication failed", errors.New("authentication failed"), true},
		{"permission denied", errors.New("permission denied"), true},
		{"public key error", errors.New("public key authentication failed"), true},
		{"other error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthenticationError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsHostKeyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"host key mismatch", errors.New("HOST KEY MISMATCH"), true},
		{"unknown host key", errors.New("unknown host key"), true},
		{"host key verification failed", errors.New("host key verification failed"), true},
		{"other error", errors.New("timeout"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHostKeyError(tt.err)
			if result != tt.expected {
				t.Errorf("IsHostKeyError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestClassifyConnectionError(t *testing.T) {
	host := "test-host"

	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{"nil error", nil, ""},
		{"timeout error", errors.New("timeout"), "connection to test-host timed out"},
		{"connection refused", errors.New("connection refused"), "connection to test-host refused"},
		{"authentication failed", errors.New("authentication failed"), "authentication failed for test-host"},
		{"host key error", errors.New("HOST KEY MISMATCH"), "host key verification failed for test-host"},
		{"generic error", errors.New("some other error"), "failed to connect to test-host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyConnectionError(host, tt.err)
			if tt.err == nil {
				if result != nil {
					t.Errorf("Expected nil for nil input, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			if !contains(result.Error(), tt.expectedMsg) {
				t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedMsg, result.Error())
			}
		})
	}
}

// Helper function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
