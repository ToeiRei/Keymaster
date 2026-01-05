// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"fmt"
	"strings"
	"time"
)

// ContainsIgnoreCase reports whether s contains sub, case-insensitive.
func ContainsIgnoreCase(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// SuggestTags returns a list of suggested tags based on the current input value.
// It treats tag matching case-insensitively and excludes tags already present
// in the input. The returned suggestions preserve the original tag casing
// provided in allTags.
func SuggestTags(allTags []string, currentVal string) []string {
	parts := strings.Split(currentVal, ",")
	if len(parts) == 0 {
		return nil
	}
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	if lastPart == "" {
		return nil
	}
	lowerLast := strings.ToLower(lastPart)

	// Collect existing tags in a lowercased set for quick exclusion.
	present := make(map[string]struct{})
	for i := 0; i < len(parts)-1; i++ {
		p := strings.ToLower(strings.TrimSpace(parts[i]))
		if p != "" {
			present[p] = struct{}{}
		}
	}

	var suggestions []string
	for _, tag := range allTags {
		lowerTag := strings.ToLower(tag)
		if strings.HasPrefix(lowerTag, lowerLast) {
			if _, ok := present[lowerTag]; !ok {
				suggestions = append(suggestions, tag)
			}
		}
	}
	return suggestions
}

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

// SplitTagsPreserveTrailing splits tags like SplitTags but preserves an empty
// trailing element when the input ends with a comma. Each part is trimmed.
func SplitTagsPreserveTrailing(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// JoinTags joins tags with ", " producing a normalized tags string.
func JoinTags(tags []string) string {
	return strings.Join(tags, ", ")
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
