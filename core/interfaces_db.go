// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package core contains small, deterministic interface definitions used by
// facade functions. This file contains minimal DB-facing interfaces that
// allow core to remain DB-agnostic. Implementations live in `internal/db`.
package core

import (
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
)

// KeyReader exposes system key reads required by deployment and generation logic.
type KeyReader interface {
	// GetActiveSystemKey returns the currently active system key.
	GetActiveSystemKey() (*model.SystemKey, error)

	// GetSystemKeyBySerial returns the system key with the given serial.
	GetSystemKeyBySerial(serial int) (*model.SystemKey, error)

	// GetAllPublicKeys returns all stored public keys.
	GetAllPublicKeys() ([]model.PublicKey, error)
}

// KeyLister exposes read-only public key access used by generators and other core helpers.
type KeyLister interface {
	// GetGlobalPublicKeys returns public keys that are marked global.
	GetGlobalPublicKeys() ([]model.PublicKey, error)

	// GetKeysForAccount returns public keys assigned to an account.
	GetKeysForAccount(accountID int) ([]model.PublicKey, error)

	// GetAllPublicKeys returns all stored public keys.
	GetAllPublicKeys() ([]model.PublicKey, error)
}

// AccountSerialUpdater is a tiny write interface used by deployment logic
// to update the serial on an account after successful deployment.
type AccountSerialUpdater interface {
	UpdateAccountSerial(accountID int, serial int) error
}

// KeyImporter exposes a minimal write API for importing public keys from
// remote hosts. This keeps core decoupled from DB-specific managers.
type KeyImporter interface {
	AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error)
}

// AccountManager exposes minimal account write operations used by core facades.
type AccountManager interface {
	DeleteAccount(id int) error
}
