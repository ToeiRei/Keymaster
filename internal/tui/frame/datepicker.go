// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package frame

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// DatePicker represents a date/time picker dialog.
type DatePicker struct {
	selectedDate time.Time
	Focused      int // 0 = year, 1 = month, 2 = day, 3 = hour, 4 = minute, 5 = ok, 6 = cancel
	Width        int
	Height       int
	showTime     bool // whether to show time fields
}

// NewDatePicker creates a new date picker starting at the given date.
// If date is zero, uses current time. showTime determines if time fields are shown.
func NewDatePicker(date time.Time, showTime bool) *DatePicker {
	if date.IsZero() {
		date = time.Now()
	}
	return &DatePicker{
		selectedDate: date,
		Focused:      0,
		Width:        60,
		Height:       15,
		showTime:     showTime,
	}
}

// GetDate returns the currently selected date.
func (dp *DatePicker) GetDate() time.Time {
	return dp.selectedDate
}

// SetShowTime enables or disables time selection.
func (dp *DatePicker) SetShowTime(show bool) {
	dp.showTime = show
}

// IncrementField increases the currently focused field.
func (dp *DatePicker) IncrementField() {
	switch dp.Focused {
	case 0: // year
		dp.selectedDate = dp.selectedDate.AddDate(1, 0, 0)
	case 1: // month
		dp.selectedDate = dp.selectedDate.AddDate(0, 1, 0)
	case 2: // day
		dp.selectedDate = dp.selectedDate.AddDate(0, 0, 1)
	case 3: // hour
		dp.selectedDate = dp.selectedDate.Add(time.Hour)
	case 4: // minute
		dp.selectedDate = dp.selectedDate.Add(time.Minute)
	}
}

// DecrementField decreases the currently focused field.
func (dp *DatePicker) DecrementField() {
	switch dp.Focused {
	case 0: // year
		dp.selectedDate = dp.selectedDate.AddDate(-1, 0, 0)
	case 1: // month
		dp.selectedDate = dp.selectedDate.AddDate(0, -1, 0)
	case 2: // day
		dp.selectedDate = dp.selectedDate.AddDate(0, 0, -1)
	case 3: // hour
		dp.selectedDate = dp.selectedDate.Add(-time.Hour)
	case 4: // minute
		dp.selectedDate = dp.selectedDate.Add(-time.Minute)
	}
}

// FocusNext moves focus to the next field.
func (dp *DatePicker) FocusNext() {
	maxFocus := 4 // year, month, day, ok, cancel
	if dp.showTime {
		maxFocus = 6 // year, month, day, hour, minute, ok, cancel
	}
	dp.Focused++
	if dp.Focused > maxFocus {
		dp.Focused = 0
	}
}

// FocusPrev moves focus to the previous field.
func (dp *DatePicker) FocusPrev() {
	maxFocus := 4
	if dp.showTime {
		maxFocus = 6
	}
	dp.Focused--
	if dp.Focused < 0 {
		dp.Focused = maxFocus
	}
}

// FocusOk moves focus to OK button.
func (dp *DatePicker) FocusOk() {
	if dp.showTime {
		dp.Focused = 5
	} else {
		dp.Focused = 3
	}
}

// FocusCancel moves focus to Cancel button.
func (dp *DatePicker) FocusCancel() {
	if dp.showTime {
		dp.Focused = 6
	} else {
		dp.Focused = 4
	}
}

// IsFocusedOk returns true if OK button is focused.
func (dp *DatePicker) IsFocusedOk() bool {
	if dp.showTime {
		return dp.Focused == 5
	}
	return dp.Focused == 3
}

// IsFocusedCancel returns true if Cancel button is focused.
func (dp *DatePicker) IsFocusedCancel() bool {
	if dp.showTime {
		return dp.Focused == 6
	}
	return dp.Focused == 4
}

// Render produces the date picker output.
func (dp *DatePicker) Render() string {
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Bold(true).
		Width(dp.Width - 2).
		Align(lipgloss.Center)

	header := headerStyle.Render("📅 Select Date" + func() string {
		if dp.showTime {
			return " & Time"
		}
		return ""
	}())

	// Date fields
	dateDisplay := dp.renderDateFields()

	// Calendar preview (optional - shows current selection nicely)
	preview := dp.renderPreview()

	// Buttons
	buttons := dp.renderButtons()

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(1, 2)
	help := helpStyle.Render("↑/↓ adjust | Tab navigate | Enter confirm")

	// Compose dialog
	dialog := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		dateDisplay,
		"",
		preview,
		"",
		buttons,
		help,
	)

	// Wrap in a styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(dp.Width)

	return boxStyle.Render(dialog)
}

// renderDateFields produces the date/time field selectors.
func (dp *DatePicker) renderDateFields() string {
	year := dp.selectedDate.Year()
	month := int(dp.selectedDate.Month())
	day := dp.selectedDate.Day()
	hour := dp.selectedDate.Hour()
	minute := dp.selectedDate.Minute()

	// Style for focused field
	focusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Bold(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Padding(0, 1)

	// Year field
	yearStr := fmt.Sprintf("%04d", year)
	if dp.Focused == 0 {
		yearStr = focusedStyle.Render(yearStr)
	} else {
		yearStr = normalStyle.Render(yearStr)
	}

	// Month field
	monthStr := fmt.Sprintf("%02d", month)
	if dp.Focused == 1 {
		monthStr = focusedStyle.Render(monthStr)
	} else {
		monthStr = normalStyle.Render(monthStr)
	}

	// Day field
	dayStr := fmt.Sprintf("%02d", day)
	if dp.Focused == 2 {
		dayStr = focusedStyle.Render(dayStr)
	} else {
		dayStr = normalStyle.Render(dayStr)
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("-")
	dateLine := lipgloss.JoinHorizontal(lipgloss.Center, yearStr, sep, monthStr, sep, dayStr)

	if !dp.showTime {
		container := lipgloss.NewStyle().
			Padding(1, 2).
			Align(lipgloss.Center)
		return container.Render(dateLine)
	}

	// Hour field
	hourStr := fmt.Sprintf("%02d", hour)
	if dp.Focused == 3 {
		hourStr = focusedStyle.Render(hourStr)
	} else {
		hourStr = normalStyle.Render(hourStr)
	}

	// Minute field
	minuteStr := fmt.Sprintf("%02d", minute)
	if dp.Focused == 4 {
		minuteStr = focusedStyle.Render(minuteStr)
	} else {
		minuteStr = normalStyle.Render(minuteStr)
	}

	timeSep := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(":")
	timeLine := lipgloss.JoinHorizontal(lipgloss.Center, hourStr, timeSep, minuteStr)

	spacer := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("  ")
	combined := lipgloss.JoinHorizontal(lipgloss.Center, dateLine, spacer, timeLine)

	container := lipgloss.NewStyle().
		Padding(1, 2).
		Align(lipgloss.Center)

	return container.Render(combined)
}

// renderPreview shows the formatted date string.
func (dp *DatePicker) renderPreview() string {
	previewStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(0, 2).
		Align(lipgloss.Center)

	var formatted string
	if dp.showTime {
		formatted = dp.selectedDate.Format("2006-01-02 15:04 MST")
	} else {
		formatted = dp.selectedDate.Format("2006-01-02")
	}

	return previewStyle.Render(formatted)
}

// renderButtons produces OK and Cancel buttons.
func (dp *DatePicker) renderButtons() string {
	okStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("239")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("239")).
		Padding(0, 2)

	cancelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("239")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("239")).
		Padding(0, 2)

	if dp.IsFocusedOk() {
		okStyle = okStyle.Background(lipgloss.Color("60")).BorderForeground(lipgloss.Color("60"))
	}
	if dp.IsFocusedCancel() {
		cancelStyle = cancelStyle.Background(lipgloss.Color("60")).BorderForeground(lipgloss.Color("60"))
	}

	okBtn := okStyle.Render("OK")
	cancelBtn := cancelStyle.Render("Cancel")

	return lipgloss.JoinHorizontal(lipgloss.Center, okBtn, "  ", cancelBtn)
}
