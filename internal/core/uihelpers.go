// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"
	"strings"
	"time"
)

// SplitTags splits a comma-separated tag string into trimmed, non-empty tags.
func SplitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		tp := strings.TrimSpace(p)
		if tp != "" {
			out = append(out, tp)
		}
	}
	return out
}

// SplitTagsPreserveTrailing exposes the internal split that preserves a
// trailing empty token when the input ends with a comma.
func SplitTagsPreserveTrailing(s string) []string {
	return splitTagsPreserveTrailing(s)
}

// JoinTags joins tags with ", " producing a normalized tags string.
func JoinTags(tags []string) string {
	return joinTags(tags)
}

// ParseExpiryInput parses an expiration input string. Accepts RFC3339 or
// date-only `YYYY-MM-DD`. Returns a time.Time (UTC for date-only inputs)
// or an error if the format is unrecognized. Empty input yields zero time
// and nil error.
func ParseExpiryInput(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t2, err := time.Parse("2006-01-02", s); err == nil {
		return time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("invalid expiry format")
}

// Pad returns the input string right-padded with spaces to reach width.
// If the string is already equal or longer, it is returned unchanged.
func Pad(s string, width int) string {
	if width <= 0 {
		return s
	}
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// FormatLabelPadding formats a label/value pair so the label is left-aligned
// within labelWidth characters and the value follows immediately. This is
// useful for dashboard/status rows where labels should align vertically.
func FormatLabelPadding(label, value string, labelWidth int) string {
	return Pad(label, labelWidth) + " " + value
}
