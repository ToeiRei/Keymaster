// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"strings"
)

// SuggestTags returns a list of suggested tags based on the current input.
// It expects `allTags` to be a sorted list of available tags. The input is
// the full tags field (e.g. "one, two, thr"). The function returns matching
// suggestions for the last token, preserving case from the tag list.
func SuggestTags(allTags []string, input string) []string {
	parts := splitTagsPreserveTrailing(input)
	if len(parts) == 0 {
		return nil
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	if last == "" {
		return nil
	}

	lower := strings.ToLower(last)
	out := make([]string, 0, 8)
	seen := make(map[string]struct{})
	for _, t := range allTags {
		if _, ok := seen[t]; ok {
			continue
		}
		if strings.HasPrefix(strings.ToLower(t), lower) {
			out = append(out, t)
			seen[t] = struct{}{}
		}
	}
	return out
}

// ApplySuggestion replaces the last tag token in `current` with `suggestion`
// and returns the new tags field value with a trailing ", ".
func ApplySuggestion(current, suggestion string) string {
	parts := splitTagsPreserveTrailing(current)
	if len(parts) == 0 {
		return suggestion + ", "
	}
	parts[len(parts)-1] = suggestion
	return joinTags(parts) + ", "
}

// splitTagsPreserveTrailing splits a comma-separated tags string while
// preserving a trailing empty token if present (e.g. "a, b," -> ["a"," b",""]).
func splitTagsPreserveTrailing(s string) []string {
	if s == "" {
		return nil
	}
	// naive split by comma preserves empties
	parts := strings.Split(s, ",")
	// Trim only left/right spaces when returning; callers expect raw pieces
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func joinTags(parts []string) string {
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, ", ")
}
