package core

import "strings"

// ContainsIgnoreCase reports whether substr is within s, case-insensitive.
func ContainsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
