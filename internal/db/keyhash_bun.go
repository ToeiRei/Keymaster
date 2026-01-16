// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"

	"sort"
	"strings"
	"time"

	"github.com/toeirei/keymaster/internal/model"
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

	// Build authorized_keys content deterministically (allow nil system key).
	var sb strings.Builder
	if sk != nil {
		sb.WriteString(fmt.Sprintf("# Keymaster Managed Keys (Serial: %d)\n", sk.Serial))
		restrictedSystemKey := fmt.Sprintf("%s %s", "command=\"internal-sftp\",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty", sk.PublicKey)
		sb.WriteString(restrictedSystemKey)
	} else {
		sb.WriteString("# Keymaster Managed Keys (Serial: 0)\n")
	}

	// Filter expired keys
	filterExpired := func(keys []model.PublicKey) []model.PublicKey {
		var out []model.PublicKey
		now := time.Now().UTC()
		for _, k := range keys {
			if k.ExpiresAt.IsZero() || k.ExpiresAt.After(now) {
				out = append(out, k)
			}
		}
		return out
	}

	globals = filterExpired(globals)
	accountKeys = filterExpired(accountKeys)

	// Combine and de-duplicate by key ID, sort by comment
	type keyInfo struct {
		id      int
		line    string
		comment string
	}
	allMap := make(map[int]keyInfo)
	formatKey := func(k model.PublicKey) string {
		if k.Comment != "" {
			return fmt.Sprintf("%s %s %s", k.Algorithm, k.KeyData, k.Comment)
		}
		return fmt.Sprintf("%s %s", k.Algorithm, k.KeyData)
	}
	for _, k := range globals {
		allMap[k.ID] = keyInfo{id: k.ID, line: formatKey(k), comment: k.Comment}
	}
	for _, k := range accountKeys {
		allMap[k.ID] = keyInfo{id: k.ID, line: formatKey(k), comment: k.Comment}
	}
	var sorted []keyInfo
	for _, v := range allMap {
		sorted = append(sorted, v)
	}
	// sort by comment
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].comment < sorted[j].comment })

	if len(sorted) > 0 {
		sb.WriteString("\n\n# User Keys\n")
		for i, ki := range sorted {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(ki.line)
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
	}
	content := sb.String()
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
