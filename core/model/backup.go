// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package model

// BackupData is a container for all data to be exported for a backup.
// It holds slices of all the core models in Keymaster.
type BackupData struct {
	// SchemaVersion helps in handling migrations during restore.
	SchemaVersion int `json:"schema_version"`

	// Data from each table.
	Accounts          []Account          `json:"accounts"`
	PublicKeys        []PublicKey        `json:"public_keys"`
	AccountKeys       []AccountKey       `json:"account_keys"`
	SystemKeys        []SystemKey        `json:"system_keys"`
	KnownHosts        []KnownHost        `json:"known_hosts"`
	AuditLogEntries   []AuditLogEntry    `json:"audit_log_entries"`
	BootstrapSessions []BootstrapSession `json:"bootstrap_sessions"`
}

// AccountKey represents the many-to-many relationship between accounts and public keys.
type AccountKey struct {
	KeyID     int `json:"key_id"`
	AccountID int `json:"account_id"`
}

// KnownHost represents a trusted host's public key.
type KnownHost struct {
	Hostname string `json:"hostname"`
	Key      string `json:"key"`
}
