// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/core/keys"
	"github.com/toeirei/keymaster/core/model"
)

// computeAccountKeyHashTx computes a deterministic fingerprint of the authorized_keys
// content for the given account using the provided Bun query runner (tx or db).
func computeAccountKeyHashTx(ctx context.Context, q execRawProvider, accountID int) (string, error) {
	// Active system key
	var skm SystemKeyModel
	if err := QueryRawInto(ctx, q, &skm, "SELECT id, serial, public_key, private_key, is_active FROM system_keys WHERE is_active = 1 LIMIT 1"); err != nil {
		if err != sql.ErrNoRows {
			return "", err
		}
	}
	var sk *model.SystemKey
	if skm.ID != 0 {
		m := systemKeyModelToModel(skm)
		sk = &m
	}

	// Global keys
	var gks []PublicKeyModel
	if err := QueryRawInto(ctx, q, &gks, "SELECT id, algorithm, key_data, comment, expires_at, is_global FROM public_keys WHERE is_global = 1 ORDER BY comment"); err != nil {
		return "", err
	}
	globals := make([]model.PublicKey, 0, len(gks))
	for _, p := range gks {
		globals = append(globals, publicKeyModelToModel(p))
	}

	// Account keys
	var aks []PublicKeyModel
	if err := QueryRawInto(ctx, q, &aks, "SELECT p.id, p.algorithm, p.key_data, p.comment, p.expires_at, p.is_global FROM public_keys p JOIN account_keys ak ON ak.key_id = p.id WHERE ak.account_id = ? ORDER BY p.comment", accountID); err != nil {
		return "", err
	}
	accountKeys := make([]model.PublicKey, 0, len(aks))
	for _, p := range aks {
		accountKeys = append(accountKeys, publicKeyModelToModel(p))
	}

	content, err := keys.BuildAuthorizedKeysContent(sk, globals, accountKeys)
	if err != nil {
		return "", err
	}
	if content == "" {
		return "", nil
	}
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum[:]), nil
}

// MaybeMarkAccountDirtyTx computes the account key hash and sets `is_dirty = true`
// and updates `key_hash` only if the new fingerprint differs from the stored one.
// q may be a *bun.DB or a transaction (bun.Tx).
func MaybeMarkAccountDirtyTx(ctx context.Context, q execRawProvider, accountID int) error {
	newHash, err := computeAccountKeyHashTx(ctx, q, accountID)
	if err != nil {
		return err
	}
	var cur sql.NullString
	if err := QueryRawInto(ctx, q, &cur, "SELECT key_hash FROM accounts WHERE id = ?", accountID); err != nil {
		return err
	}
	if !cur.Valid || cur.String != newHash {
		if _, err := ExecRaw(ctx, q, "UPDATE accounts SET key_hash = ?, is_dirty = ? WHERE id = ?", newHash, true, accountID); err != nil {
			return MapDBError(err)
		}
		// Record audit entry with the new fingerprint instead of storing/printing full authorized_keys
		details := fmt.Sprintf("account:%d key_hash:%s", accountID, newHash)
		if _, err := ExecRaw(ctx, q, "INSERT INTO audit_log (username, action, details) VALUES (?, ?, ?)", "system", "ACCOUNT_KEY_HASH_UPDATED", details); err != nil {
			return MapDBError(err)
		}
	}
	return nil
}

// HashAuthorizedKeysContent normalizes a raw authorized_keys payload and
// returns the SHA256 hex fingerprint using the same basic normalization
// rules we expect on-disk: normalize CRLF to LF and trim trailing whitespace
// on each line so hashes computed from files transferred between platforms
// remain stable.
func HashAuthorizedKeysContent(raw []byte) string {
	s := string(raw)
	// Normalize CRLF -> LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Trim trailing spaces/tabs per-line
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	norm := strings.Join(lines, "\n")
	sum := sha256.Sum256([]byte(norm))
	return fmt.Sprintf("%x", sum[:])
}
