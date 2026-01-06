// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package frame

import (
	"github.com/charmbracelet/lipgloss"
)

// Dialog represents a modal dialog box with title, message, and two buttons.
type Dialog struct {
	title       string
	message     string
	buttonLeft  string
	buttonRight string
	focused     bool // which button is focused (false = left, true = right)
	width       int
	height      int
}

// NewDialog creates a new dialog with the given title, message, and button labels.
func NewDialog(title, message, buttonLeft, buttonRight string) *Dialog {
	return &Dialog{
		title:       title,
		message:     message,
		buttonLeft:  buttonLeft,
		buttonRight: buttonRight,
		focused:     false, // left button focused by default
		width:       60,
		height:      12,
	}
}

// SetSize sets the dialog dimensions.
func (d *Dialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// FocusRight moves focus to the right button.
func (d *Dialog) FocusRight() {
	d.focused = true
}

// FocusLeft moves focus to the left button.
func (d *Dialog) FocusLeft() {
	d.focused = false
}

// IsFocusedRight returns true if the right button is focused.
func (d *Dialog) IsFocusedRight() bool {
	return d.focused
}

// Render produces the dialog box output with auto-calculated height.
func (d *Dialog) Render() string {
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Bold(true).
		Width(d.width)

	header := headerStyle.Render(" " + d.title)

	// Message area with padding
	messageStyle := lipgloss.NewStyle().
		Width(d.width-4).
		Padding(1, 2, 0, 2)

	message := messageStyle.Render(d.message)

	// Button area
	buttonArea := d.renderButtonArea()

	// Compose dialog: header + message + buttons
	dialog := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		message,
		buttonArea,
	)

	// Wrap in a styled box with auto height
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(d.width)

	return boxStyle.Render(dialog)
}

// renderButtonArea produces the button row with styled buttons.
func (d *Dialog) renderButtonArea() string {
	// Button styling - actual button appearance with borders
	leftStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("239")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("239")).
		Padding(0, 3, 0, 3)

	rightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("239")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("239")).
		Padding(0, 3, 0, 3)

	// Highlight the focused button with blue background and border
	if d.focused {
		rightStyle = rightStyle.
			Background(lipgloss.Color("60")).
			BorderForeground(lipgloss.Color("60"))
	} else {
		leftStyle = leftStyle.
			Background(lipgloss.Color("60")).
			BorderForeground(lipgloss.Color("60"))
	}

	leftBtn := leftStyle.Render(d.buttonLeft)
	rightBtn := rightStyle.Render(d.buttonRight)

	// Render buttons horizontally with space between
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, leftBtn, "  ", rightBtn)

	// Pad and center the button area
	buttonAreaStyle := lipgloss.NewStyle().
		Padding(1, 2, 1, 2)

	return buttonAreaStyle.Render(buttonRow)
}
