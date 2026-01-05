package ui

import "strings"

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
