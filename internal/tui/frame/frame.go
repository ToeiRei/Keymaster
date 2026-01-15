// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package frame

import (
	"strings"
	"unicode/utf8"
)

// Footer builds a one-line footer from left and right tokens, aligning the
// right token to the right edge of a line with the specified width.
// It is rune-aware and will truncate the left side if space is insufficient.
func Footer(left, right string, width int) string {
	if width <= 0 {
		return left + " " + right
	}
	// simple case: left + padding + right
	rl := runeWidth(right)
	ll := runeWidth(left)
	// if both fit with at least one space
	if ll+rl+1 <= width {
		pad := width - ll - rl
		return left + strings.Repeat(" ", pad) + right
	}
	// not enough space -> truncate left
	// leave at least 1 char for left, and place right at end
	maxLeft := width - rl
	if maxLeft <= 0 {
		// no space for left, return right trimmed to width
		return trimToWidth(right, width)
	}
	return trimToWidth(left, maxLeft) + right
}

func runeWidth(s string) int {
	return utf8.RuneCountInString(s)
}

func trimToWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= w {
		return s
	}
	return string(runes[:w])
}

// StatusBar renders a single-line status bar with a background (reverse
// video) style. It aligns left/right tokens like Footer and wraps the line in
// ANSI reverse-video so it appears as a background-colored bar in terminals
// that support ANSI sequences.
func StatusBar(left, right string, width int) string {
	line := Footer(left, right, width)
	// reverse video on/off
	const revOn = "\x1b[7m"
	const revOff = "\x1b[0m"
	return revOn + line + revOff
}

