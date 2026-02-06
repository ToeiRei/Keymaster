// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/core/bootstrap"
)

// CreateTransferBootstrap creates a bootstrap session for transfer/handover.
// It persists the session (public key) and returns the session ID and PEM
// encoded private key which the caller must export securely out-of-band.
func CreateTransferBootstrap(username, hostname, label, tags string) (sessionID string, privateKeyPEM string, err error) {
	s, err := bootstrap.NewBootstrapSession(username, hostname, label, tags)
	if err != nil {
		return "", "", fmt.Errorf("create bootstrap session: %w", err)
	}

	// Persist the session (stores public key, expiry, status)
	if err := s.Save(); err != nil {
		return "", "", fmt.Errorf("save bootstrap session: %w", err)
	}

	// Register in active sessions so signal handling / cleanup can find it.
	bootstrap.RegisterSession(s)

	// Return session id and private key PEM for out-of-band transfer.
	return s.ID, string(s.TempKeyPair.GetPrivateKeyPEM()), nil
}
