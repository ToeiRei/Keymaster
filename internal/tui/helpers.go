// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
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
