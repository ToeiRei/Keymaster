// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

// Package ui provides small UI adapters. Most logic has moved to
// `internal/core`. These functions are thin wrappers retained for
// compatibility and will delegate to `core` implementations.

import (
	"time"

	"github.com/toeirei/keymaster/internal/core"
)

// ContainsIgnoreCase delegates to core.ContainsIgnoreCase.
func ContainsIgnoreCase(s, sub string) bool {
	return core.ContainsIgnoreCase(s, sub)
}

// SuggestTags delegates to core.SuggestTags.
func SuggestTags(allTags []string, currentVal string) []string {
	return core.SuggestTags(allTags, currentVal)
}

// SplitTags delegates to core.SplitTags.
func SplitTags(s string) []string { return core.SplitTags(s) }

// SplitTagsPreserveTrailing delegates to core.SplitTagsPreserveTrailing.
func SplitTagsPreserveTrailing(s string) []string { return core.SplitTagsPreserveTrailing(s) }

// JoinTags delegates to core.JoinTags.
func JoinTags(tags []string) string { return core.JoinTags(tags) }

// ParseExpiryInput delegates to core.ParseExpiryInput.
func ParseExpiryInput(s string) (time.Time, error) {
	return core.ParseExpiryInput(s)
}

// Pad delegates to core.Pad.
func Pad(s string, width int) string { return core.Pad(s, width) }

// FormatLabelPadding delegates to core.FormatLabelPadding.
func FormatLabelPadding(label, value string, labelWidth int) string {
	return core.FormatLabelPadding(label, value, labelWidth)
}
