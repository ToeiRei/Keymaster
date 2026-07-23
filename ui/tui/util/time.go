// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	timeLayout1 string = "2006.01.02 15:04:05"
	timeLayout2 string = "2006.01.02 15:04"
	timeLayout3 string = "2006.01.02"
	timeLayout4 string = "02.01.2006 15:04:05"
	timeLayout5 string = "02.01.2006 15:04"
	timeLayout6 string = "02.01.2006"
)

// ParseTime parses a date string. An empty (or whitespace-only) input is
// treated as "no time" and returns the zero value without error, so optional
// time fields can be left blank. A non-empty but unparseable value errors.
func ParseTime(timeStr string) (time.Time, error) {
	if strings.TrimSpace(timeStr) == "" {
		return time.Time{}, nil
	}

	var result time.Time
	var err error
	for _, layout := range []string{timeLayout1, timeLayout2, timeLayout3, timeLayout4, timeLayout5, timeLayout6} {
		result, err = time.Parse(layout, timeStr)
		if err == nil {
			break
		}
	}

	return result, err
}

// StringifyTime renders a time, mirroring [ParseTime]: the zero value renders
// as an empty string so optional/absent times display and round-trip as blank.
func StringifyTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(timeLayout1)
}

// RenderExpiry renders an expiry time for display in tables: an unset (zero)
// expiry shows a greyed-out, italic "never" instead of a blank cell.
func RenderExpiry(value time.Time) string {
	if value.IsZero() {
		return lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("240")).Render("never")
	}
	return StringifyTime(value)
}
