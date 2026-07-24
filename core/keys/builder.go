// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package keys

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/toeirei/keymaster/core/model"
	"golang.org/x/crypto/ssh"
)

// BuildAuthorizedKeysContent constructs the authorized_keys content given the
// system key and lists of global and account-specific public keys. This
// function is pure and deterministic; callers must provide keys fetched from
// their data stores.
func BuildAuthorizedKeysContent(systemKey *model.SystemKey, globalKeys, accountKeys []model.PublicKey) (string, error) {
	if systemKey == nil {
		return "", fmt.Errorf("no active system key provided")
	}

	managedKeyFingerprint := managedKeyFingerprint(systemKey.PublicKey)
	payload, err := buildAuthorizedKeysPayload(systemKey, globalKeys, accountKeys)
	if err != nil {
		return "", err
	}
	payloadHash := hashAuthorizedKeysPayload(payload)

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Keymaster Managed Keys (Serial: %d)\n", systemKey.Serial)
	if managedKeyFingerprint != "" {
		fmt.Fprintf(&sb, "# Managed Key Fingerprint: %s\n", managedKeyFingerprint)
	}
	fmt.Fprintf(&sb, "# Payload Hash: sha256:%s\n", payloadHash)
	fmt.Fprintf(&sb, "# Verify: sha256sum ~/.ssh/authorized_keys | awk '{print $1}'\n")
	fmt.Fprintf(&sb, "%s %s\n", "command=\"internal-sftp\",no-port-forwarding,no-x11-forwarding,no-agent-forwarding,no-pty", systemKey.PublicKey)
	if payload != "" {
		sb.WriteString(payload)
	}
	return sb.String(), nil
}

func buildAuthorizedKeysPayload(systemKey *model.SystemKey, globalKeys, accountKeys []model.PublicKey) (string, error) {
	if systemKey == nil {
		return "", fmt.Errorf("no active system key provided")
	}

	// Helper to filter expired keys
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

	globalKeys = filterExpired(globalKeys)
	accountKeys = filterExpired(accountKeys)

	// Combine and de-duplicate by key ID
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

	for _, k := range globalKeys {
		allMap[k.ID] = keyInfo{id: k.ID, line: formatKey(k), comment: k.Comment}
	}
	for _, k := range accountKeys {
		allMap[k.ID] = keyInfo{id: k.ID, line: formatKey(k), comment: k.Comment}
	}

	var sorted []keyInfo
	for _, v := range allMap {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].comment < sorted[j].comment })

	var sb strings.Builder
	if len(sorted) > 0 {
		sb.WriteString("\n\n# User Keys\n")
		for i, ki := range sorted {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(ki.line)
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func managedKeyFingerprint(publicKey string) string {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return ""
	}
	return ssh.FingerprintSHA256(key)
}

func hashAuthorizedKeysPayload(payload string) string {
	payload = strings.ReplaceAll(payload, "\r\n", "\n")
	lines := strings.Split(payload, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// SSHKeyTypeToVerifyCommand maps an SSH public key type to a sensible
// ssh-keygen command that can be used to verify host keys on typical Linux
// distributions. This is pure and deterministic.
func SSHKeyTypeToVerifyCommand(keyType string) string {
	switch keyType {
	case "ssh-rsa":
		return "ssh-keygen -lf /etc/ssh/ssh_host_rsa_key.pub"
	case "ssh-dss":
		return "ssh-keygen -lf /etc/ssh/ssh_host_dsa_key.pub"
	case "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		return "ssh-keygen -lf /etc/ssh/ssh_host_ecdsa_key.pub"
	case "ssh-ed25519":
		return "ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub"
	default:
		return "ssh-keygen -lf /etc/ssh/ssh_host_*_key.pub"
	}
}
