package core

import (
	"fmt"
	"strings"
)

// ValidateBootstrapParams checks the minimal required fields for a bootstrap
// operation. It performs pure, deterministic validation and returns a
// non-nil error when input is invalid.
func ValidateBootstrapParams(username, hostname, label, tags string) error {
	u := strings.TrimSpace(username)
	h := strings.TrimSpace(hostname)

	if u == "" || h == "" {
		return fmt.Errorf("username and hostname cannot be empty")
	}
	return nil
}
