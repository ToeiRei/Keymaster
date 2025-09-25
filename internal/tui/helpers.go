// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/i18n"
)

// FilterI18nKeys holds the translation keys for filter status messages.
type FilterI18nKeys struct {
	Filtering    string // e.g., "accounts.filtering"
	FilterActive string // e.g., "accounts.filter_active"
	FilterHint   string // e.g., "accounts.filter_hint"
}

// getFilterStatusLine generates the standard filter status string for footers.
// It takes the filtering state, the filter text, a struct of i18n keys,
// and optional arguments for the format strings (like a column name).
func getFilterStatusLine(isFiltering bool, filterText string, keys FilterI18nKeys, formatArgs ...interface{}) string {
	allArgs := append(formatArgs, filterText)
	if isFiltering {
		return i18n.T(keys.Filtering, allArgs...)
	}
	if filterText != "" {
		return i18n.T(keys.FilterActive, allArgs...)
	}
	return i18n.T(keys.FilterHint)
}

// scrollbar renders a simple vertical scrollbar.
// It returns an empty string if the total number of items is less than or equal
// to the height of the view, as no scrollbar is needed.
func scrollbar(viewHeight, totalItems, start, end int) string {
	if totalItems <= viewHeight {
		return ""
	}

	// The character for the scrollbar track.
	trackChar := "│"
	// The character for the scrollbar thumb (the part that moves).
	thumbChar := "█"

	var sb strings.Builder

	// Calculate the height and position of the thumb.
	thumbHeight := int(float64(viewHeight) / float64(totalItems) * float64(viewHeight))
	if thumbHeight < 1 {
		thumbHeight = 1
	}

	thumbTop := int(float64(start) / float64(totalItems) * float64(viewHeight))

	for i := 0; i < viewHeight; i++ {
		if i >= thumbTop && i < thumbTop+thumbHeight {
			sb.WriteString(thumbChar)
		} else {
			sb.WriteString(trackChar)
		}
		sb.WriteString("\n")
	}

	return lipgloss.NewStyle().MarginLeft(1).Render(sb.String())
}
