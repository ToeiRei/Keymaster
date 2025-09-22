package sshkey

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

// ParseSerial extracts the Keymaster serial number from a comment line.
func ParseSerial(line string) (int, error) {
	// Expected format: # Keymaster System Key (Serial: 123)
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "# Keymaster System Key") {
		return 0, fmt.Errorf("not a keymaster key comment line")
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
// if it's considered weak or deprecated.
func CheckHostKeyAlgorithm(key ssh.PublicKey) string {
	keyType := key.Type()
	switch keyType {
	case ssh.KeyAlgoDSA:
		return "SECURITY WARNING: Host key uses deprecated and insecure ssh-dss (DSA) algorithm."
	case ssh.KeyAlgoRSA:
		return "SECURITY WARNING: Host key uses ssh-rsa, which is disabled by default in modern OpenSSH. Consider upgrading the host's keys."
	default:
		return ""
	}
}
