package tui

import (
	"strings"
	"unicode/utf8"
)

// AlignFooter returns a single-line string where `right` is right-aligned
// within `width` columns and `left` is at the start. If width is too small
// a single space separates the tokens.
func AlignFooter(left, right string, width int) string {
	leftLen := utf8.RuneCountInString(left)
	rightLen := utf8.RuneCountInString(right)
	spaces := width - leftLen - rightLen
	if spaces < 1 {
		spaces = 1
	}
	return left + strings.Repeat(" ", spaces) + right
}
