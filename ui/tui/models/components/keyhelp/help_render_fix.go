// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
// Package keyhelp contains helper views used to render key help in the TUI.
package keyhelp

import (
	"strings"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// FIX help.Model.ShortHelpView is bugged
// ShortHelpView returns a compact help view for short displays.
func ShortHelpView(m help.Model, bindings []key.Binding) string {
	if len(bindings) == 0 {
		return ""
	}

	var b strings.Builder
	var usedWidth int
	var items []string
	separator := m.Styles.ShortSeparator.Inline(true).Render(m.ShortSeparator)
	tail := " " + m.Styles.Ellipsis.Inline(true).Render(m.Ellipsis)
	tailLen := lipgloss.Width(tail)

	for i, kb := range bindings {
		if !kb.Enabled() {
			continue
		}

		// Sep
		var sep string
		if i > 0 {
			sep = separator
		}

		// Item
		str := sep +
			m.Styles.ShortKey.Inline(true).Render(kb.Help().Key) + " " +
			m.Styles.ShortDesc.Inline(true).Render(kb.Help().Desc)

		items = append(items, str)
	}

	for i, item := range items {
		itemLen := lipgloss.Width(item)
		if i < len(items)-1 {
			// when not last
			if usedWidth+itemLen+tailLen <= m.Width {
				// when next items and at least the tail fit
				usedWidth += itemLen
				b.WriteString(item)
			} else {
				// else just add the tail
				usedWidth += tailLen
				b.WriteString(tail)
				break
			}
		} else {
			// when last
			if usedWidth+itemLen <= m.Width {
				// last item fits
				b.WriteString(item)
			} else if usedWidth+tailLen <= m.Width {
				// tail fits
				b.WriteString(tail)
			}
			// nothing fits
		}
	}

	return b.String()
}

// FIX help.Model.FullHelpView is bugged
// FullHelpView returns the full help view content.
func FullHelpView(m help.Model, groups [][]key.Binding) string {
	if len(groups) == 0 {
		return ""
	}

	var cols []string
	var result []string
	var usedWidth int
	separator := m.Styles.FullSeparator.Inline(true).Render(m.FullSeparator)
	tail := " " + m.Styles.Ellipsis.Inline(true).Render(m.Ellipsis)
	tailLen := lipgloss.Width(tail)

	// Iterate over groups to build columns
	for i, group := range groups {
		if group == nil || !slices.ContainsFunc(group, func(binding key.Binding) bool {
			return binding.Enabled()
		}) {
			// ignore groups with unly disabled bindings
			continue
		}
		var (
			sep          string
			keys         []string
			descriptions []string
		)

		// Sep
		if i > 0 {
			sep = separator
		}

		// Separate keys and descriptions into different slices
		for _, binding := range group {
			if !binding.Enabled() {
				// ignore disabled bindings
				continue
			}
			keys = append(keys, binding.Help().Key)
			descriptions = append(descriptions, binding.Help().Desc)
		}

		// Column
		col := lipgloss.JoinHorizontal(lipgloss.Top,
			sep,
			m.Styles.FullKey.Render(lipgloss.JoinVertical(lipgloss.Left, keys...)),
			" ",
			m.Styles.FullDesc.Render(lipgloss.JoinVertical(lipgloss.Left, descriptions...)),
		)

		cols = append(cols, col)
	}

	for i, col := range cols {
		colLen := lipgloss.Width(col)
		if i < len(cols)-1 {
			// when not last
			if usedWidth+colLen+tailLen <= m.Width {
				// when next items and at least the tail fit
				usedWidth += colLen
				result = append(result, col)
			} else {
				// else just add the tail
				usedWidth += tailLen
				result = append(result, tail)
				break
			}
		} else {
			// when last
			if usedWidth+colLen <= m.Width {
				// last item fits
				result = append(result, col)
			} else if usedWidth+tailLen <= m.Width {
				// tail fits
				result = append(result, tail)
			}
			// nothing fits
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, result...)
}
