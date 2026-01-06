// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"
)

// renderResultBlock builds a vertical block containing a primary message,
// optional warnings, and an error if present. Callers provide already-
// localized `primary` and `warnings` strings.
func renderResultBlock(primary string, warnings []string, err error) string {
	var parts []string
	if primary != "" {
		parts = append(parts, primary)
	}
	if len(warnings) > 0 {
		parts = append(parts, "")
		parts = append(parts, "Warnings:")
		for _, w := range warnings {
			parts = append(parts, "  "+w)
		}
	}
	if err != nil {
		parts = append(parts, "")
		parts = append(parts, "Error: "+err.Error())
	}
	return strings.Join(parts, "\n")
}
