// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package debug

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toeirei/keymaster/internal/tui/frame"
)

// Launch runs the development-only test screen. It returns immediately if the
// environment variable KEYMASTER_TUI_TEST is not set to "1".
func Launch() {
	if os.Getenv("KEYMASTER_TUI_TEST") != "1" {
		return
	}
	m := newTestModel()
	if _, err := tea.NewProgram(&m).Run(); err != nil {
		// On failure, print to stderr and exit non-zero to aid debugging.
		os.Stderr.WriteString("test screen error: " + err.Error() + "\n")
		os.Exit(1)
	}
}

type testModel struct {
	vp             viewport.Model
	width          int
	height         int
	menu           *frame.ListView
	dialog         *frame.Dialog
	showDialog     bool
	filePicker     *frame.FilePicker
	showFilePicker bool
	datePicker     *frame.DatePicker
	showDatePicker bool
}

func newTestModel() testModel {
	m := testModel{
		vp: viewport.New(20, 5),
	}
	// Populate the viewport with sample multi-line content including unicode.
	sample := "This is a TUI framework test screen.\n"
	sample += "It shows header, scrollable body and footer.\n"
	sample += "Unicode test: ✓ ✅ ✨ — 漢字 — テスト\n"
	for i := 0; i < 40; i++ {
		sample += fmt.Sprintf("Line %c — The quick brown fox jumps over the lazy dog.\n", 'A'+(i%26))
	}
	m.vp.SetContent(sample)
	// Create a simple left-hand menu list for the dashboard mock
	menuItems := []string{"Overview", "Accounts", "Public Keys", "Deploy", "Audit", "Tags"}
	ml := frame.NewList(menuItems)
	m.menu = ml
	// Create a test dialog
	m.dialog = frame.NewDialog(
		"⚠️  Confirm Action",
		"Are you sure you want to proceed with this operation?",
		"Cancel",
		"Ok",
	)
	m.showDialog = false

	// Create a file picker starting from home directory
	home, _ := os.UserHomeDir()
	m.filePicker = frame.NewFilePicker(home)
	m.showFilePicker = false

	// Create a date picker (default to 30 days from now, with time)
	futureDate := time.Now().AddDate(0, 0, 30)
	m.datePicker = frame.NewDatePicker(futureDate, true)
	m.showDatePicker = false

	return m
}

func (m testModel) Init() tea.Cmd { return nil }

func (m *testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "d":
			m.showDialog = !m.showDialog
		case "t":
			m.showDatePicker = !m.showDatePicker
		case "f":
			m.showFilePicker = !m.showFilePicker
			// Toggle between load and save mode for testing
			if m.showFilePicker {
				m.filePicker.SetSaveMode(false) // Load mode by default, press 's' for save
			}
		case "s":
			if m.showFilePicker {
				// Toggle save mode
				m.filePicker.SetSaveMode(!m.filePicker.IsSaveMode())
			}
		case "tab", "right":
			if m.showDialog {
				m.dialog.FocusRight()
			} else if m.showDatePicker {
				m.datePicker.FocusNext()
			} else if m.showFilePicker {
				// Tab cycles through: file list -> [filename input] -> OK -> Cancel -> file list
				if m.filePicker.IsSaveMode() {
					if m.filePicker.Focused == 0 {
						m.filePicker.FocusFilename()
					} else if m.filePicker.Focused == 1 {
						m.filePicker.FocusOk()
					} else if m.filePicker.Focused == 2 {
						m.filePicker.FocusCancel()
					} else {
						m.filePicker.FocusFileList()
					}
				} else {
					if m.filePicker.Focused == 0 {
						m.filePicker.FocusOk()
					} else if m.filePicker.Focused == 1 {
						m.filePicker.FocusCancel()
					} else {
						m.filePicker.FocusFileList()
					}
				}
			}
		case "shift+tab", "left":
			if m.showDialog {
				m.dialog.FocusLeft()
			} else if m.showDatePicker {
				m.datePicker.FocusPrev()
			} else if m.showFilePicker {
				// Shift+Tab cycles backward
				if m.filePicker.IsSaveMode() {
					if m.filePicker.Focused == 0 {
						m.filePicker.FocusCancel()
					} else if m.filePicker.Focused == 1 {
						m.filePicker.FocusFileList()
					} else if m.filePicker.Focused == 2 {
						m.filePicker.FocusFilename()
					} else {
						m.filePicker.FocusOk()
					}
				} else {
					if m.filePicker.Focused == 0 {
						m.filePicker.FocusCancel()
					} else if m.filePicker.Focused == 1 {
						m.filePicker.FocusFileList()
					} else {
						m.filePicker.FocusOk()
					}
				}
			}
			if m.showDialog {
				m.dialog.FocusLeft()
			} else if m.showFilePicker {
				// Shift+Tab moves through file picker buttons backward
				if m.filePicker.Focused == 0 {
					m.filePicker.FocusCancel()
				} else if m.filePicker.Focused == 1 {
					m.filePicker.FocusFileList()
				} else {
					m.filePicker.FocusOk()
				}
			}
		case "enter", " ":
			if m.showDialog {
				if m.dialog.IsFocusedRight() {
					// Right button (Ok) pressed
					m.showDialog = false
				} else {
					// Left button (Cancel) pressed
					m.showDialog = false
				}
			} else if m.showDatePicker {
				if m.datePicker.IsFocusedOk() {
					// OK pressed - close date picker
					m.showDatePicker = false
				} else if m.datePicker.IsFocusedCancel() {
					// Cancel pressed
					m.showDatePicker = false
				}
			} else if m.showFilePicker && m.filePicker.Focused == 0 {
				// Enter on file list: enter directory or select file
				selected := m.filePicker.SelectCurrent()
				if selected != "" {
					// File selected - close picker
					m.showFilePicker = false
				}
			}
		case "j", "down":
			if !m.showDialog && !m.showFilePicker && !m.showDatePicker {
				m.vp.LineDown(1)
			} else if m.showFilePicker {
				m.filePicker.MoveDown()
			} else if m.showDatePicker {
				m.datePicker.DecrementField()
			}
		case "k", "up":
			if !m.showDialog && !m.showFilePicker && !m.showDatePicker {
				m.vp.LineUp(1)
			} else if m.showFilePicker {
				m.filePicker.MoveUp()
			} else if m.showDatePicker {
				m.datePicker.IncrementField()
			}
		case "u":
			if m.showFilePicker {
				m.filePicker.GoUp()
			}
		case "J":
			if !m.showDialog && !m.showFilePicker && !m.showDatePicker {
				m.menu.MoveDown()
			}
		case "K":
			if !m.showDialog && !m.showFilePicker && !m.showDatePicker {
				m.menu.MoveUp()
			}
		case "backspace":
			if m.showFilePicker && m.filePicker.Focused == 1 {
				m.filePicker.Backspace()
			}
		default:
			// Handle typing in filename input
			if m.showFilePicker && m.filePicker.Focused == 1 && len(msg.String()) == 1 {
				m.filePicker.TypeChar(rune(msg.String()[0]))
			}
		}
	case tea.WindowSizeMsg:
		// Reserve 3 lines for header/footer and padding
		m.width = msg.Width
		m.height = msg.Height
		bodyHeight := msg.Height - 3
		if bodyHeight < 3 {
			bodyHeight = 3
		}
		m.vp.Height = bodyHeight
		m.vp.Width = msg.Width
	}
	return m, nil
}

func (m testModel) View() string {
	if m.width < 10 || m.height < 5 {
		return "Terminal too small"
	}

	// Use full terminal dimensions (colored bars provide visual structure).
	frameW := m.width
	frameH := m.height

	// Calculate layout measurements following the canonical contract.
	navWidth := 24
	if frameW < 60 {
		navWidth = frameW / 3
	}
	sepWidth := 3 // " │ "
	bodyWidth := frameW - navWidth - sepWidth

	// Calculate available height.
	// Account for: header (2) + footer (1) + 2 horizontal separators
	headerHeight := 2
	footerHeight := 1
	separatorLines := 2 // top and bottom box lines
	mainHeight := frameH - headerHeight - footerHeight - separatorLines
	if mainHeight < 3 {
		mainHeight = 3
	}

	// Step 1: Render header block (title + subtitle).
	headerBlock := m.renderHeader()

	// Step 2: Render nav pane.
	navPane := m.renderNav(navWidth, mainHeight)

	// Step 3: Render separator pane.
	sepPane := m.renderSeparator(mainHeight)

	// Step 4: Render body pane.
	bodyPane := m.renderBody(bodyWidth, mainHeight)

	// Step 5: Compose main area (horizontal join of nav, sep, body).
	main := lipgloss.JoinHorizontal(lipgloss.Top, navPane, sepPane, bodyPane)

	// Step 5b: Create horizontal separators for box effect.
	hSep := m.renderHorizontalSeparator()

	// Step 6: Render footer.
	footer := m.renderFooter()

	// Step 7: Compose final layout (vertical join of header, main, footer with separators).
	final := lipgloss.JoinVertical(lipgloss.Left, headerBlock, hSep, main, hSep, footer)

	// Step 8: Render dialog if shown (overlay on top of background).
	if m.showDialog {
		m.dialog.SetSize(60, 0) // width only, height auto-calculated
		dialogOutput := m.dialog.Render()
		return m.overlayDialog(dialogOutput, final)
	}

	// Step 9: Render file picker if shown (overlay on top of background).
	if m.showFilePicker {
		m.filePicker.Width = 70
		m.filePicker.Height = frameH - 4
		pickerOutput := m.filePicker.Render()
		return m.overlayDialog(pickerOutput, final)
	}

	// Step 10: Render date picker if shown (overlay on top of background).
	if m.showDatePicker {
		m.datePicker.Width = 60
		m.datePicker.Height = frameH - 4
		dateOutput := m.datePicker.Render()
		return m.overlayDialog(dateOutput, final)
	}

	return final
}

// overlayDialog places the dialog box centered (both horizontally and vertically) with the background visible around it.
func (m testModel) overlayDialog(dialog, background string) string {
	bgLines := strings.Split(strings.TrimRight(background, "\n"), "\n")
	dialogLines := strings.Split(strings.TrimRight(dialog, "\n"), "\n")

	if len(bgLines) == 0 || len(dialogLines) == 0 {
		return dialog
	}

	dialogHeight := len(dialogLines)
	bgHeight := len(bgLines)
	bgWidth := len(bgLines[0])
	dialogWidth := 0

	// Find the widest dialog line
	for _, line := range dialogLines {
		if len(line) > dialogWidth {
			dialogWidth = len(line)
		}
	}

	// Calculate vertical centering
	topSpacing := (bgHeight - dialogHeight) / 2
	if topSpacing < 0 {
		topSpacing = 0
	}

	// Calculate horizontal centering
	leftPadding := (bgWidth - dialogWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// Build the output
	var result []string

	// Add background lines before dialog
	for i := 0; i < topSpacing && i < len(bgLines); i++ {
		result = append(result, bgLines[i])
	}

	// Add dialog lines, centered horizontally
	for _, line := range dialogLines {
		paddingStr := strings.Repeat(" ", leftPadding)
		result = append(result, paddingStr+line)
	}

	// Add background lines after dialog
	for i := topSpacing + dialogHeight; i < len(bgLines); i++ {
		result = append(result, bgLines[i])
	}

	return strings.Join(result, "\n")
}

// renderHeader produces the 2-row header block with background.
func (m testModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Bold(true).
		Width(m.width)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Background(lipgloss.Color("60")).
		Width(m.width)

	title := titleStyle.Render(" 🔑 Keymaster — Layout Test")
	subtitle := subtitleStyle.Render(" An agentless SSH key manager that just does the job.")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle)
}

// renderNav produces the left navigation pane.
func (m testModel) renderNav(width, height int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	header := headerStyle.Render("🗂️  Navigation")

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 1, 0, 1) // top, right, bottom, left

	m.menu.SetSize(width-2, height-4) // account for header, margins, padding, and empty line
	rendered := m.menu.Render()

	// Compose: empty line + header + empty line + menu
	emptyLine := lipgloss.NewStyle().Width(width).Render("")
	nav := lipgloss.JoinVertical(lipgloss.Left, emptyLine, header, emptyLine, rendered)

	return style.Render(nav)
}

// renderSeparator produces the vertical separator pane.
func (m testModel) renderSeparator(height int) string {
	style := lipgloss.NewStyle().
		Width(3).
		Height(height)

	// Pad separator to match height, skip first and last lines for spacing (matching horizontal separators).
	lines := make([]string, height)
	lines[0] = "" // empty line at top (aligns with nav header)
	for i := 1; i < height-1; i++ {
		lines[i] = " │ "
	}
	if height > 1 {
		lines[height-1] = "" // empty line at bottom for spacing
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

// renderBody produces the right body pane with viewport.
func (m testModel) renderBody(width, height int) string {
	// Create header for breathing room and context.
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)

	header := headerStyle.Render("📋 Overview")

	// Render viewport with adjusted height for header and margins.
	headerHeight := 1
	topMargin := 1
	bottomMargin := 1
	paddingWidth := 2  // left + right padding
	paddingHeight := 2 // top + bottom padding
	contentHeight := height - headerHeight - topMargin - bottomMargin - paddingHeight
	if contentHeight < 3 {
		contentHeight = 3
	}

	m.vp.Width = width - paddingWidth
	m.vp.Height = contentHeight

	// Compose: empty line + header + empty line + content
	emptyLine := lipgloss.NewStyle().Width(width - paddingWidth).Render("")
	body := lipgloss.JoinVertical(lipgloss.Left, emptyLine, header, emptyLine, m.vp.View())

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 1, 1, 1) // top, right, bottom, left

	return style.Render(body)
}

// renderFooter produces the bottom footer.
func (m testModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Width(m.width)

	text := " j/k body scroll  J/K menu  q quit"
	return footerStyle.Render(text)
}

// renderHorizontalSeparator produces a shortened horizontal line for box framing.
func (m testModel) renderHorizontalSeparator() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	// Shorter line with 1 char margin on each side for spacing
	sepWidth := m.width - 2
	line := " "
	for i := 0; i < sepWidth; i++ {
		line += "─"
	}
	line += " "
	return style.Render(line)
}
