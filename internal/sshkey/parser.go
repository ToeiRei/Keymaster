package sshkey

import (
	"fmt"
	"strings"
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
