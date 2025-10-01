// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package model

import (
	"strings"
	"testing"
	"time"
)

// TestDriftClassificationConstants tests the drift classification constants
func TestDriftClassificationConstants(t *testing.T) {
	tests := []struct {
		name  string
		value DriftClassification
		str   string
	}{
		{"Critical", DriftCritical, "critical"},
		{"Warning", DriftWarning, "warning"},
		{"Info", DriftInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.str {
				t.Errorf("Expected %s, got %s", tt.str, string(tt.value))
			}
		})
	}
}

// TestDriftAnalysisIsCritical tests the IsCritical method
func TestDriftAnalysisIsCritical(t *testing.T) {
	tests := []struct {
		name           string
		classification DriftClassification
		expected       bool
	}{
		{"Critical is critical", DriftCritical, true},
		{"Warning is not critical", DriftWarning, false},
		{"Info is not critical", DriftInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &DriftAnalysis{
				Classification: tt.classification,
			}
			if analysis.IsCritical() != tt.expected {
				t.Errorf("Expected IsCritical() to be %v, got %v", tt.expected, analysis.IsCritical())
			}
		})
	}
}

// TestDriftAnalysisIsWarning tests the IsWarning method
func TestDriftAnalysisIsWarning(t *testing.T) {
	tests := []struct {
		name           string
		classification DriftClassification
		expected       bool
	}{
		{"Warning is warning", DriftWarning, true},
		{"Critical is not warning", DriftCritical, false},
		{"Info is not warning", DriftInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &DriftAnalysis{
				Classification: tt.classification,
			}
			if analysis.IsWarning() != tt.expected {
				t.Errorf("Expected IsWarning() to be %v, got %v", tt.expected, analysis.IsWarning())
			}
		})
	}
}

// TestDriftAnalysisSummary tests the Summary method
func TestDriftAnalysisSummary(t *testing.T) {
	tests := []struct {
		name        string
		analysis    DriftAnalysis
		shouldContain []string
	}{
		{
			name: "No drift",
			analysis: DriftAnalysis{
				HasDrift: false,
			},
			shouldContain: []string{"No drift detected"},
		},
		{
			name: "Critical drift with serial mismatch",
			analysis: DriftAnalysis{
				HasDrift:       true,
				Classification: DriftCritical,
				SerialMismatch: true,
				ExpectedSerial: 2,
				ActualSerial:   1,
			},
			shouldContain: []string{"critical", "serial mismatch"},
		},
		{
			name: "Critical drift with missing header",
			analysis: DriftAnalysis{
				HasDrift:               true,
				Classification:         DriftCritical,
				MissingKeymasterHeader: true,
			},
			shouldContain: []string{"critical", "header missing"},
		},
		{
			name: "Warning drift with missing keys",
			analysis: DriftAnalysis{
				HasDrift:       true,
				Classification: DriftWarning,
				MissingKeys: []PublicKey{
					{ID: 1, Comment: "key1"},
					{ID: 2, Comment: "key2"},
				},
			},
			shouldContain: []string{"warning", "missing"},
		},
		{
			name: "Info drift with extra keys",
			analysis: DriftAnalysis{
				HasDrift:       true,
				Classification: DriftInfo,
				ExtraKeys:      []string{"extra1", "extra2", "extra3"},
			},
			shouldContain: []string{"info", "extra"},
		},
		{
			name: "Multiple drift indicators",
			analysis: DriftAnalysis{
				HasDrift:               true,
				Classification:         DriftCritical,
				SerialMismatch:         true,
				MissingKeymasterHeader: true,
				ModifiedSystemKey:      true,
				MissingKeys:            []PublicKey{{ID: 1}},
				ExtraKeys:              []string{"extra"},
			},
			shouldContain: []string{"critical", "serial mismatch", "header missing", "system key modified"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.analysis.Summary()
			summaryLower := strings.ToLower(summary)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(summaryLower, strings.ToLower(expected)) {
					t.Errorf("Expected summary to contain '%s', but got: %s", expected, summary)
				}
			}
		})
	}
}

// TestDriftEventDefaults tests default values for DriftEvent
func TestDriftEventDefaults(t *testing.T) {
	event := DriftEvent{
		ID:         1,
		AccountID:  10,
		DetectedAt: time.Now(),
		DriftType:  DriftCritical,
		Details:    "test details",
	}

	if event.WasRemediated {
		t.Error("Expected WasRemediated to default to false")
	}

	if event.RemediatedAt != nil {
		t.Error("Expected RemediatedAt to default to nil")
	}
}

// TestRemediationResultDefaults tests default values for RemediationResult
func TestRemediationResultDefaults(t *testing.T) {
	result := RemediationResult{}

	if result.Success {
		t.Error("Expected Success to default to false")
	}

	if result.Error != nil {
		t.Error("Expected Error to default to nil")
	}

	if result.DeployedKeys != 0 {
		t.Error("Expected DeployedKeys to default to 0")
	}
}

// TestRemediationResultWithSuccess tests a successful remediation
func TestRemediationResultWithSuccess(t *testing.T) {
	result := RemediationResult{
		Success:               true,
		Error:                 nil,
		DeployedKeys:          5,
		FixedDriftType:        DriftWarning,
		PreRemediationSerial:  1,
		PostRemediationSerial: 2,
		Details:               "Successfully remediated warning drift",
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Error != nil {
		t.Error("Expected Error to be nil for successful remediation")
	}

	if result.DeployedKeys != 5 {
		t.Errorf("Expected DeployedKeys to be 5, got %d", result.DeployedKeys)
	}

	if result.FixedDriftType != DriftWarning {
		t.Errorf("Expected FixedDriftType to be DriftWarning, got %s", result.FixedDriftType)
	}
}

// TestRemediationResultWithError tests a failed remediation
func TestRemediationResultWithError(t *testing.T) {
	testErr := &MockError{message: "connection timeout"}

	result := RemediationResult{
		Success:        false,
		Error:          testErr,
		DeployedKeys:   0,
		FixedDriftType: DriftCritical,
		Details:        "Failed to deploy: connection timeout",
	}

	if result.Success {
		t.Error("Expected Success to be false for failed remediation")
	}

	if result.Error == nil {
		t.Error("Expected Error to be set for failed remediation")
	}

	if result.Error.Error() != "connection timeout" {
		t.Errorf("Expected error message 'connection timeout', got '%s'", result.Error.Error())
	}
}

// TestAccountDriftStatsDefaults tests AccountDriftStats
func TestAccountDriftStatsDefaults(t *testing.T) {
	stats := AccountDriftStats{
		Account: Account{
			ID:       1,
			Username: "testuser",
			Hostname: "testhost",
		},
		DriftCount:    5,
		LastDriftAt:   time.Now(),
		LastDriftType: DriftCritical,
	}

	if stats.DriftCount != 5 {
		t.Errorf("Expected DriftCount to be 5, got %d", stats.DriftCount)
	}

	if stats.LastDriftType != DriftCritical {
		t.Errorf("Expected LastDriftType to be DriftCritical, got %s", stats.LastDriftType)
	}
}

// TestDriftAnalysisHasDriftFlag tests the HasDrift flag
func TestDriftAnalysisHasDriftFlag(t *testing.T) {
	tests := []struct {
		name     string
		analysis DriftAnalysis
		expected bool
	}{
		{
			name: "No drift",
			analysis: DriftAnalysis{
				HasDrift: false,
			},
			expected: false,
		},
		{
			name: "Has drift",
			analysis: DriftAnalysis{
				HasDrift:       true,
				Classification: DriftCritical,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.analysis.HasDrift != tt.expected {
				t.Errorf("Expected HasDrift to be %v, got %v", tt.expected, tt.analysis.HasDrift)
			}
		})
	}
}

// TestDriftAnalysisRawContent tests raw content storage
func TestDriftAnalysisRawContent(t *testing.T) {
	expectedContent := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 user@host"
	actualContent := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ user@host"

	analysis := DriftAnalysis{
		HasDrift:            true,
		RawExpectedContent:  expectedContent,
		RawActualContent:    actualContent,
	}

	if analysis.RawExpectedContent != expectedContent {
		t.Errorf("RawExpectedContent mismatch")
	}

	if analysis.RawActualContent != actualContent {
		t.Errorf("RawActualContent mismatch")
	}

	// Content should be different
	if analysis.RawExpectedContent == analysis.RawActualContent {
		t.Error("Expected and actual content should be different")
	}
}

// Mock error for testing
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// Benchmark tests
func BenchmarkDriftAnalysisSummary(b *testing.B) {
	analysis := DriftAnalysis{
		HasDrift:               true,
		Classification:         DriftCritical,
		SerialMismatch:         true,
		MissingKeymasterHeader: true,
		ModifiedSystemKey:      true,
		MissingKeys: []PublicKey{
			{ID: 1, Comment: "key1"},
			{ID: 2, Comment: "key2"},
		},
		ExtraKeys: []string{"extra1", "extra2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis.Summary()
	}
}

func BenchmarkDriftAnalysisIsCritical(b *testing.B) {
	analysis := DriftAnalysis{
		Classification: DriftCritical,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analysis.IsCritical()
	}
}
