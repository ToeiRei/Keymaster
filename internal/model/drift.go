// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package model defines the core data structures for drift detection and remediation.
package model // import "github.com/toeirei/keymaster/internal/model"

import (
	"time"
)

// DriftClassification represents the severity level of detected configuration drift.
type DriftClassification string

const (
	// DriftCritical indicates a severe drift that requires immediate attention
	// (e.g., system key serial mismatch, missing Keymaster header).
	DriftCritical DriftClassification = "critical"

	// DriftWarning indicates a moderate drift that should be addressed
	// (e.g., missing or extra user keys).
	DriftWarning DriftClassification = "warning"

	// DriftInfo indicates a minor drift that is informational only
	// (e.g., extra keys not managed by Keymaster).
	DriftInfo DriftClassification = "info"
)

// DriftEvent represents a single instance of detected configuration drift.
type DriftEvent struct {
	ID            int                 // The primary key for the drift event.
	AccountID     int                 // The account where drift was detected.
	DetectedAt    time.Time           // When the drift was detected.
	DriftType     DriftClassification // The severity classification of the drift.
	Details       string              // A detailed description of the drift.
	WasRemediated bool                // Whether the drift was automatically fixed.
	RemediatedAt  *time.Time          // When the drift was remediated (nil if not remediated).
}

// DriftAnalysis contains detailed information about detected configuration drift.
type DriftAnalysis struct {
	// Classification is the severity level of the detected drift.
	Classification DriftClassification

	// HasDrift indicates whether any drift was detected at all.
	HasDrift bool

	// SerialMismatch indicates if the remote system key serial doesn't match the database.
	SerialMismatch bool

	// ExpectedSerial is the serial number that should be on the remote host.
	ExpectedSerial int

	// ActualSerial is the serial number found on the remote host (0 if not found or unparseable).
	ActualSerial int

	// MissingKeys are public keys that should be on the host but are not.
	MissingKeys []PublicKey

	// ExtraKeys are public keys found on the host that are not in the database.
	// These are represented as raw strings since they may not be in our database.
	ExtraKeys []string

	// ModifiedSystemKey indicates if the Keymaster system key has been altered.
	ModifiedSystemKey bool

	// MissingKeymasterHeader indicates if the Keymaster management header is missing.
	MissingKeymasterHeader bool

	// RawExpectedContent is the full expected authorized_keys content.
	RawExpectedContent string

	// RawActualContent is the full actual authorized_keys content from the remote host.
	RawActualContent string
}

// RemediationResult contains the outcome of a drift remediation attempt.
type RemediationResult struct {
	// Success indicates whether the remediation was successful.
	Success bool

	// Error contains any error that occurred during remediation.
	Error error

	// DeployedKeys is the number of keys that were deployed to the host.
	DeployedKeys int

	// FixedDriftType is the classification of drift that was remediated.
	FixedDriftType DriftClassification

	// PreRemediationSerial is the system key serial before remediation.
	PreRemediationSerial int

	// PostRemediationSerial is the system key serial after remediation.
	PostRemediationSerial int

	// Details contains additional information about the remediation.
	Details string
}

// AccountDriftStats represents drift statistics for a single account.
type AccountDriftStats struct {
	Account       Account // The account with drift.
	DriftCount    int     // Total number of drift events for this account.
	LastDriftAt   time.Time
	LastDriftType DriftClassification
}

// IsCritical returns true if the drift is classified as critical.
func (d *DriftAnalysis) IsCritical() bool {
	return d.Classification == DriftCritical
}

// IsWarning returns true if the drift is classified as a warning.
func (d *DriftAnalysis) IsWarning() bool {
	return d.Classification == DriftWarning
}

// Summary returns a human-readable summary of the drift analysis.
func (d *DriftAnalysis) Summary() string {
	if !d.HasDrift {
		return "No drift detected"
	}

	summary := string(d.Classification) + " drift: "

	if d.SerialMismatch {
		summary += "System key serial mismatch. "
	}
	if d.MissingKeymasterHeader {
		summary += "Keymaster header missing. "
	}
	if d.ModifiedSystemKey {
		summary += "System key modified. "
	}
	if len(d.MissingKeys) > 0 {
		summary += "Missing keys: " + string(rune(len(d.MissingKeys))) + ". "
	}
	if len(d.ExtraKeys) > 0 {
		summary += "Extra keys: " + string(rune(len(d.ExtraKeys))) + ". "
	}

	return summary
}
