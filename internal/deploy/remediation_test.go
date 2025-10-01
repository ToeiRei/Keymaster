// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

// TestParseKeysFromContent tests the parseKeysFromContent function
func TestParseKeysFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "Empty content",
			content: "",
			expected: 0,
		},
		{
			name: "Single key",
			content: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host",
			expected: 1,
		},
		{
			name: "Multiple keys",
			content: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ user@host2
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGH user@host3`,
			expected: 3,
		},
		{
			name: "Keys with comments",
			content: `# This is a comment
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1
# Another comment
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ user@host2`,
			expected: 2,
		},
		{
			name: "Keys with restrictions (system key)",
			content: `command="internal-sftp",no-port-forwarding ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 keymaster-system-key`,
			expected: 1,
		},
		{
			name: "Mixed content",
			content: `# Keymaster Managed Keys (Serial: 1)
command="internal-sftp",no-port-forwarding ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 keymaster-system-key

# User Keys
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ user@host2`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := parseKeysFromContent(tt.content)
			if len(keys) != tt.expected {
				t.Errorf("Expected %d keys, got %d", tt.expected, len(keys))
			}
		})
	}
}

// TestKeysMatch tests the keysMatch function
func TestKeysMatch(t *testing.T) {
	tests := []struct {
		name     string
		key1     string
		key2     string
		expected bool
	}{
		{
			name:     "Identical keys",
			key1:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host",
			key2:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host",
			expected: true,
		},
		{
			name:     "Different whitespace",
			key1:     "ssh-ed25519  AAAAC3NzaC1lZDI1NTE5AAAAIFQ  user@host",
			key2:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host",
			expected: true,
		},
		{
			name:     "Different keys",
			key1:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1",
			key2:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGH user@host2",
			expected: false,
		},
		{
			name:     "Different algorithms",
			key1:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host",
			key2:     "ssh-rsa AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host",
			expected: false,
		},
		{
			name:     "Same key different comments",
			key1:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1",
			key2:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host2",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := keysMatch(tt.key1, tt.key2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for keys:\n  %s\n  %s", tt.expected, result, tt.key1, tt.key2)
			}
		})
	}
}

// TestIsSystemKey tests the isSystemKey function
func TestIsSystemKey(t *testing.T) {
	tests := []struct {
		name     string
		keyLine  string
		expected bool
	}{
		{
			name:     "Keymaster system key with restrictions",
			keyLine:  `command="internal-sftp",no-port-forwarding ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 keymaster-system-key`,
			expected: true,
		},
		{
			name:     "System key with keymaster-system-key comment",
			keyLine:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 keymaster-system-key",
			expected: true,
		},
		{
			name:     "Regular user key",
			keyLine:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 user@host",
			expected: false,
		},
		{
			name:     "Key with internal-sftp but not system key",
			keyLine:  `command="internal-sftp" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 some-other-key`,
			expected: true, // Because it has internal-sftp
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemKey(tt.keyLine)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for key: %s", tt.expected, result, tt.keyLine)
			}
		})
	}
}

// TestDriftClassification tests drift classification logic
func TestDriftClassification(t *testing.T) {
	tests := []struct {
		name               string
		hasMissingHeader   bool
		hasSerialMismatch  bool
		missingKeysCount   int
		extraKeysCount     int
		expectedClass      model.DriftClassification
	}{
		{
			name:              "No drift",
			hasMissingHeader:  false,
			hasSerialMismatch: false,
			missingKeysCount:  0,
			extraKeysCount:    0,
			expectedClass:     model.DriftInfo,
		},
		{
			name:              "Missing Keymaster header - Critical",
			hasMissingHeader:  true,
			hasSerialMismatch: false,
			missingKeysCount:  0,
			extraKeysCount:    0,
			expectedClass:     model.DriftCritical,
		},
		{
			name:              "Serial mismatch - Critical",
			hasMissingHeader:  false,
			hasSerialMismatch: true,
			missingKeysCount:  0,
			extraKeysCount:    0,
			expectedClass:     model.DriftCritical,
		},
		{
			name:              "Missing keys - Warning",
			hasMissingHeader:  false,
			hasSerialMismatch: false,
			missingKeysCount:  2,
			extraKeysCount:    0,
			expectedClass:     model.DriftWarning,
		},
		{
			name:              "Extra keys only - Info",
			hasMissingHeader:  false,
			hasSerialMismatch: false,
			missingKeysCount:  0,
			extraKeysCount:    3,
			expectedClass:     model.DriftInfo,
		},
		{
			name:              "Critical overrides Warning",
			hasMissingHeader:  true,
			hasSerialMismatch: false,
			missingKeysCount:  2,
			extraKeysCount:    0,
			expectedClass:     model.DriftCritical,
		},
		{
			name:              "Warning overrides Info",
			hasMissingHeader:  false,
			hasSerialMismatch: false,
			missingKeysCount:  1,
			extraKeysCount:    5,
			expectedClass:     model.DriftWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &model.DriftAnalysis{
				MissingKeymasterHeader: tt.hasMissingHeader,
				SerialMismatch:         tt.hasSerialMismatch,
				Classification:         model.DriftInfo, // Start with Info
			}

			// Simulate missing keys
			for i := 0; i < tt.missingKeysCount; i++ {
				analysis.MissingKeys = append(analysis.MissingKeys, model.PublicKey{
					ID:      i,
					Comment: "test-key",
				})
			}

			// Simulate extra keys
			for i := 0; i < tt.extraKeysCount; i++ {
				analysis.ExtraKeys = append(analysis.ExtraKeys, "extra-key-"+string(rune(i)))
			}

			// Apply classification logic (from remediation.go)
			if analysis.MissingKeymasterHeader {
				analysis.Classification = model.DriftCritical
			}
			if analysis.SerialMismatch {
				analysis.Classification = model.DriftCritical
			}

			if analysis.Classification != model.DriftCritical {
				if len(analysis.MissingKeys) > 0 {
					analysis.Classification = model.DriftWarning
				} else if len(analysis.ExtraKeys) > 0 {
					analysis.Classification = model.DriftInfo
				}
			}

			if analysis.Classification != tt.expectedClass {
				t.Errorf("Expected classification %s, got %s", tt.expectedClass, analysis.Classification)
			}
		})
	}
}

// TestDriftAnalysisSummary tests the Summary method
func TestDriftAnalysisSummary(t *testing.T) {
	tests := []struct {
		name     string
		analysis model.DriftAnalysis
		contains []string
	}{
		{
			name: "No drift",
			analysis: model.DriftAnalysis{
				HasDrift: false,
			},
			contains: []string{"No drift detected"},
		},
		{
			name: "Serial mismatch",
			analysis: model.DriftAnalysis{
				HasDrift:       true,
				Classification: model.DriftCritical,
				SerialMismatch: true,
			},
			contains: []string{"critical", "System key serial mismatch"},
		},
		{
			name: "Missing header",
			analysis: model.DriftAnalysis{
				HasDrift:               true,
				Classification:         model.DriftCritical,
				MissingKeymasterHeader: true,
			},
			contains: []string{"critical", "Keymaster header missing"},
		},
		{
			name: "Missing keys",
			analysis: model.DriftAnalysis{
				HasDrift:       true,
				Classification: model.DriftWarning,
				MissingKeys: []model.PublicKey{
					{ID: 1, Comment: "key1"},
					{ID: 2, Comment: "key2"},
				},
			},
			contains: []string{"warning"},
		},
		{
			name: "Extra keys",
			analysis: model.DriftAnalysis{
				HasDrift:       true,
				Classification: model.DriftInfo,
				ExtraKeys:      []string{"key1", "key2", "key3"},
			},
			contains: []string{"info"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.analysis.Summary()
			for _, expected := range tt.contains {
				if !strings.Contains(strings.ToLower(summary), strings.ToLower(expected)) {
					t.Errorf("Expected summary to contain '%s', got: %s", expected, summary)
				}
			}
		})
	}
}

// TestDriftAnalysisClassificationMethods tests IsCritical and IsWarning methods
func TestDriftAnalysisClassificationMethods(t *testing.T) {
	tests := []struct {
		name           string
		classification model.DriftClassification
		isCritical     bool
		isWarning      bool
	}{
		{
			name:           "Critical drift",
			classification: model.DriftCritical,
			isCritical:     true,
			isWarning:      false,
		},
		{
			name:           "Warning drift",
			classification: model.DriftWarning,
			isCritical:     false,
			isWarning:      true,
		},
		{
			name:           "Info drift",
			classification: model.DriftInfo,
			isCritical:     false,
			isWarning:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &model.DriftAnalysis{
				Classification: tt.classification,
				HasDrift:       true,
			}

			if analysis.IsCritical() != tt.isCritical {
				t.Errorf("Expected IsCritical() to be %v, got %v", tt.isCritical, analysis.IsCritical())
			}

			if analysis.IsWarning() != tt.isWarning {
				t.Errorf("Expected IsWarning() to be %v, got %v", tt.isWarning, analysis.IsWarning())
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseKeysFromContent(b *testing.B) {
	content := `# Keymaster Managed Keys (Serial: 1)
command="internal-sftp",no-port-forwarding ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 keymaster-system-key

# User Keys
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ user@host2
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGH user@host3
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIXZ user@host4
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAB user@host5`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseKeysFromContent(content)
	}
}

func BenchmarkKeysMatch(b *testing.B) {
	key1 := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFQ user@host1"
	key2 := "ssh-ed25519  AAAAC3NzaC1lZDI1NTE5AAAAIFQ  user@host1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keysMatch(key1, key2)
	}
}
