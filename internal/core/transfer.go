// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/toeirei/keymaster/internal/bootstrap"
	"github.com/toeirei/keymaster/internal/security"
)

// TransferPackage represents the JSON transfer package format.
type TransferPackage struct {
	Magic              string `json:"magic"`
	User               string `json:"user"`
	Host               string `json:"host"`
	HostKey            string `json:"host_key"`
	TransferPrivateKey string `json:"transfer_private_key"`
	CRC                string `json:"crc"`
}

// BuildTransferPackage creates a transfer bootstrap session for the given
// account and returns the marshaled JSON package. It will attempt to fetch
// the remote host key via DefaultDeployerManager; if unavailable the host_key
// field will be empty. The returned package includes a crc (sha256 hex) over
// the compact JSON payload (everything except the crc field).
func BuildTransferPackage(username, hostname, label, tags string) ([]byte, error) {
	// Create an in-memory bootstrap session so we have a temporary keypair.
	s, err := bootstrap.NewBootstrapSession(username, hostname, label, tags)
	if err != nil {
		return nil, fmt.Errorf("create bootstrap session: %w", err)
	}

	// Get host key if possible via deployer manager
	var hostKey string
	if DefaultDeployerManager != nil {
		if hk, herr := DefaultDeployerManager.GetRemoteHostKey(hostname); herr == nil {
			hostKey = hk
		}
	}

	// Encode private key as base64
	privB64 := base64.StdEncoding.EncodeToString(s.TempKeyPair.GetPrivateKeyPEM())

	// Build payload without CRC
	payload := map[string]string{
		"magic":                "keymaster-transfer-v1",
		"user":                 username,
		"host":                 hostname,
		"host_key":             hostKey,
		"transfer_private_key": privB64,
	}
	compact, err := json.Marshal(payload)
	if err != nil {
		s.Cleanup()
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	sum := sha256.Sum256(compact)
	crc := hex.EncodeToString(sum[:])

	pkg := TransferPackage{
		Magic:              payload["magic"],
		User:               payload["user"],
		Host:               payload["host"],
		HostKey:            payload["host_key"],
		TransferPrivateKey: payload["transfer_private_key"],
		CRC:                crc,
	}

	out, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		s.Cleanup()
		return nil, fmt.Errorf("marshal package: %w", err)
	}
	// Note: do not cleanup session here since caller may have persisted session
	// However this function created an in-memory session; wipe sensitive memory now.
	s.Cleanup()
	return out, nil
}

// AcceptTransferPackage validates the package CRC, decodes the private key and
// performs a bootstrap deployment using the provided BootstrapDeps. It returns
// the bootstrap result from PerformBootstrapDeployment.
func AcceptTransferPackage(ctx context.Context, pkgBytes []byte, deps BootstrapDeps) (BootstrapResult, error) {
	var pkg TransferPackage
	if err := json.Unmarshal(pkgBytes, &pkg); err != nil {
		return BootstrapResult{}, fmt.Errorf("invalid transfer package: %w", err)
	}
	if pkg.Magic != "keymaster-transfer-v1" {
		return BootstrapResult{}, fmt.Errorf("unsupported transfer package: %s", pkg.Magic)
	}

	// Recompute CRC over payload fields
	payload := map[string]string{
		"magic":                pkg.Magic,
		"user":                 pkg.User,
		"host":                 pkg.Host,
		"host_key":             pkg.HostKey,
		"transfer_private_key": pkg.TransferPrivateKey,
	}
	compact, err := json.Marshal(payload)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("internal crc error: %w", err)
	}
	sum := sha256.Sum256(compact)
	expect := hex.EncodeToString(sum[:])
	if pkg.CRC != expect {
		return BootstrapResult{}, fmt.Errorf("transfer package CRC mismatch")
	}

	// Decode private key
	priv, err := base64.StdEncoding.DecodeString(pkg.TransferPrivateKey)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("invalid base64 private key: %w", err)
	}

	params := BootstrapParams{
		Username:       pkg.User,
		Hostname:       pkg.Host,
		Label:          "",
		Tags:           "",
		TempPrivateKey: security.FromBytes(priv),
		HostKey:        pkg.HostKey,
	}

	return PerformBootstrapDeployment(ctx, params, deps)
}
