// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package sshkey provides utilities for parsing and validating SSH key data.
// It includes functions to parse authorized_keys lines, extract Keymaster-specific
// metadata, and check for weak cryptographic algorithms.
package sshkey // import "github.com/toeirei/keymaster/internal/sshkey"

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Parse splits a raw public key string (like one from an authorized_keys file)
// into its three core components: algorithm, key data, and comment.
// It correctly handles leading options in the line (e.g., from="...",command="...").
func Parse(rawKey string) (algorithm, keyData, comment string, err error) {
	fields := strings.Fields(rawKey)
	if len(fields) == 0 {
		err = fmt.Errorf("empty line")
		return
	}

	keyStartIndex := -1
	for i, field := range fields {
		if strings.HasPrefix(field, "ssh-") || strings.HasPrefix(field, "ecdsa-") {
			keyStartIndex = i
			break
		}
	}

	if keyStartIndex == -1 {
		err = fmt.Errorf("no valid SSH key type found in line")
		return
	}

	if len(fields) < keyStartIndex+2 {
		err = fmt.Errorf("invalid public key format: missing key data after algorithm")
		return
	}

	algorithm = fields[keyStartIndex]
	keyData = fields[keyStartIndex+1]
	if len(fields) > keyStartIndex+2 {
		comment = strings.Join(fields[keyStartIndex+2:], " ")
	}

	return
}

// ParseSerial extracts the Keymaster serial number from the header comment line
// of a Keymaster-managed authorized_keys file.
func ParseSerial(line string) (int, error) {
	// Expected format: # Keymaster Managed Keys (Serial: 123)
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "# Keymaster Managed Keys") {
		return 0, fmt.Errorf("not a keymaster managed keys header line")
	}

	re := regexp.MustCompile(`Serial: (\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return 0, fmt.Errorf("serial number not found in comment")
	}

	serial, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse serial number '%s': %w", matches[1], err)
	}
	return serial, nil
}

// CheckHostKeyAlgorithm inspects the public key's algorithm and returns a warning
// message if the algorithm is considered weak or deprecated.
func CheckHostKeyAlgorithm(key ssh.PublicKey) string {
	keyType := key.Type()
	switch keyType {
	case "ssh-dss":
		return "SECURITY WARNING: Host key uses deprecated and insecure ssh-dss (DSA) algorithm."
	case ssh.KeyAlgoRSA:
		return "SECURITY WARNING: Host key uses ssh-rsa, which is disabled by default in modern OpenSSH. Consider upgrading the host's keys."
	default:
		return ""
	}
}
