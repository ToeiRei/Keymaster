// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package frame

import (
	"strings"
)

// ListView is a minimal reusable list component for TUI views. It renders a
// vertical list of items with a selection cursor. It is intentionally small
// and focused to match the needs of the debug/dashboard sandbox.
type ListView struct {
	Items    []string
	Selected int
	Width    int
	Height   int
}

// NewList creates a new ListView populated with items.
func NewList(items []string) *ListView {
	return &ListView{Items: items}
}

// SetSize sets the rendering width and height for the list.
func (l *ListView) SetSize(w, h int) {
	l.Width = w
	l.Height = h
}

// MoveUp moves the selection up by one, clamping to bounds.
func (l *ListView) MoveUp() {
	if l.Selected > 0 {
		l.Selected--
	}
}

// MoveDown moves the selection down by one, clamping to bounds.
func (l *ListView) MoveDown() {
	if l.Selected < len(l.Items)-1 {
		l.Selected++
	}
}

// Render returns the textual representation of the list constrained by width
// and height. Each line is prefixed with a cursor marker for the selected item.
func (l *ListView) Render() string {
	var b strings.Builder
	maxLines := l.Height
	if maxLines <= 0 {
		maxLines = len(l.Items)
	}
	for i := 0; i < len(l.Items) && i < maxLines; i++ {
		item := l.Items[i]
		prefix := "  "
		if i == l.Selected {
			prefix = "> "
		}
		// compute available width for the text (including prefix)
		contentWidth := l.Width
		lineText := item
		if contentWidth > 0 {
			// leave room for prefix
			avail := contentWidth - len([]rune(prefix))
			if avail < 0 {
				avail = 0
			}
			if len([]rune(item)) > avail {
				runes := []rune(item)
				if avail > 3 {
					lineText = string(runes[:avail-3]) + "..."
				} else {
					lineText = string(runes[:avail])
				}
			}
		}
		line := prefix + lineText
		// pad to desired width if set
		if contentWidth > 0 {
			pad := contentWidth - len([]rune(line))
			if pad < 0 {
				pad = 0
			}
			b.WriteString(line + strings.Repeat(" ", pad) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

