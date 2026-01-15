// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package frame

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// FilePicker represents a file selection dialog.
type FilePicker struct {
	currentPath string
	files       []os.FileInfo
	selected    int // index of selected file (includes . and ..)
	Focused     int // 0 = file list, 1 = filename input, 2 = ok button, 3 = cancel button
	Width       int
	Height      int
	showFiles   bool
	filter      string         // file extension filter (e.g., ".txt")
	scrollPos   int            // scroll position in file list
	vp          viewport.Model // viewport for scrolling
	filename    string         // typed filename for save mode
	saveMode    bool           // true = save mode (show input), false = load mode
}

// NewFilePicker creates a new file picker starting at the given path.
func NewFilePicker(path string) *FilePicker {
	fp := &FilePicker{
		currentPath: path,
		selected:    0,
		Focused:     0,
		Width:       60,
		Height:      20,
		showFiles:   true,
		filter:      "",
		scrollPos:   0,
		vp:          viewport.New(60, 18),
		filename:    "",
		saveMode:    false,
	}
	fp.loadFiles()
	return fp
}

// SetSaveMode enables save mode, which shows a filename input field.
func (fp *FilePicker) SetSaveMode(saveMode bool) {
	fp.saveMode = saveMode
	if saveMode {
		fp.filename = ""
	}
}

// SetFilename sets the filename for save mode.
func (fp *FilePicker) SetFilename(name string) {
	fp.filename = name
}

// GetFilename returns the current filename.
func (fp *FilePicker) GetFilename() string {
	return fp.filename
}

// IsSaveMode returns whether the picker is in save mode.
func (fp *FilePicker) IsSaveMode() bool {
	return fp.saveMode
}

// TypeChar adds a character to the filename (in save mode).
func (fp *FilePicker) TypeChar(ch rune) {
	if fp.Focused == 1 && fp.saveMode {
		fp.filename += string(ch)
	}
}

// Backspace removes the last character from the filename.
func (fp *FilePicker) Backspace() {
	if fp.Focused == 1 && fp.saveMode && len(fp.filename) > 0 {
		fp.filename = fp.filename[:len(fp.filename)-1]
	}
}

// SetFilter sets a file extension filter (e.g., ".txt" or "").
func (fp *FilePicker) SetFilter(filter string) {
	fp.filter = filter
	fp.loadFiles()
}

// loadFiles reads the current directory and updates the file list.
// It includes . (current) and .. (parent) entries at the top.
func (fp *FilePicker) loadFiles() {
	entries, err := os.ReadDir(fp.currentPath)
	if err != nil {
		fp.files = nil
		return
	}

	var files []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		// Filter by extension if set
		if fp.filter != "" && !info.IsDir() {
			if !strings.HasSuffix(info.Name(), fp.filter) {
				continue
			}
		}
		files = append(files, info)
	}

	// Sort: directories first, then files
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir() != files[j].IsDir() {
			return files[i].IsDir()
		}
		return files[i].Name() < files[j].Name()
	})

	// Prepend . and ..
	currDir := &directoryEntry{name: "."}
	parentDir := &directoryEntry{name: ".."}
	fp.files = append([]os.FileInfo{currDir, parentDir}, files...)

	if fp.selected >= len(fp.files) {
		fp.selected = 0
	}
	// Update viewport height
	fp.vp.Height = fp.Height - 3 // Account for header
}

// directoryEntry is a fake os.FileInfo for . and .. entries.
type directoryEntry struct {
	name string
}

func (d *directoryEntry) Name() string       { return d.name }
func (d *directoryEntry) Size() int64        { return 0 }
func (d *directoryEntry) Mode() os.FileMode  { return os.ModeDir }
func (d *directoryEntry) ModTime() time.Time { return time.Time{} }
func (d *directoryEntry) IsDir() bool        { return true }
func (d *directoryEntry) Sys() interface{}   { return nil }

// MoveUp moves selection up in the file list.
func (fp *FilePicker) MoveUp() {
	if fp.Focused == 0 && fp.selected > 0 {
		fp.selected--
		// Scroll viewport if needed to keep selection visible
		if fp.selected < fp.vp.YOffset {
			fp.vp.ScrollUp(1)
		}
	}
}

// MoveDown moves selection down in the file list.
func (fp *FilePicker) MoveDown() {
	if fp.Focused == 0 && fp.selected < len(fp.files)-1 {
		fp.selected++
		// Scroll viewport if needed to keep selection visible
		if fp.selected >= fp.vp.YOffset+fp.vp.Height {
			fp.vp.ScrollDown(1)
		}
	}
}

// ScrollUp scrolls the viewport up.
func (fp *FilePicker) ScrollUp() {
	if fp.Focused == 0 {
		fp.vp.ScrollUp(1)
	}
}

// ScrollDown scrolls the viewport down.
func (fp *FilePicker) ScrollDown() {
	if fp.Focused == 0 {
		fp.vp.ScrollDown(1)
	}
}

// SelectCurrent selects the current file or enters the directory.
func (fp *FilePicker) SelectCurrent() string {
	if fp.selected >= 0 && fp.selected < len(fp.files) {
		info := fp.files[fp.selected]
		fullPath := filepath.Join(fp.currentPath, info.Name())
		if info.IsDir() {
			fp.currentPath = fullPath
			fp.loadFiles()
			return ""
		}
		return fullPath
	}
	return ""
}

// GoUp navigates up one directory level.
func (fp *FilePicker) GoUp() {
	parent := filepath.Dir(fp.currentPath)
	if parent != fp.currentPath { // not at root
		fp.currentPath = parent
		fp.loadFiles()
	}
}

// GetSelected returns the currently selected file or directory name.
func (fp *FilePicker) GetSelected() string {
	if fp.selected >= 0 && fp.selected < len(fp.files) {
		return fp.files[fp.selected].Name()
	}
	return ""
}

// FocusFileList sets focus to the file list.
func (fp *FilePicker) FocusFileList() {
	fp.Focused = 0
}

// FocusFilename sets focus to the filename input field (save mode only).
func (fp *FilePicker) FocusFilename() {
	if fp.saveMode {
		fp.Focused = 1
	}
}

// FocusOk sets focus to the OK button.
func (fp *FilePicker) FocusOk() {
	if fp.saveMode {
		fp.Focused = 2
	} else {
		fp.Focused = 1
	}
}

// FocusCancel sets focus to the Cancel button.
func (fp *FilePicker) FocusCancel() {
	if fp.saveMode {
		fp.Focused = 3
	} else {
		fp.Focused = 2
	}
}

// Render produces the file picker output.
func (fp *FilePicker) Render() string {
	// Calculate widths: buttons take fixed 14 chars (10 button + 4 for borders/padding), rest for file list
	buttonWidth := 14
	fileListWidth := fp.Width - buttonWidth - 6 // Account for box border and spacing

	// Update viewport dimensions
	heightAdjust := 6
	if fp.saveMode {
		heightAdjust = 9 // More space for filename input
	}
	fp.vp.Width = fileListWidth
	fp.vp.Height = fp.Height - heightAdjust

	// Header with path
	headerContent := fp.renderHeader()

	// Filename input (save mode only)
	var filenameInput string
	if fp.saveMode {
		filenameInput = fp.renderFilenameInput()
	}

	// File list (scrollable)
	fileList := fp.renderFileList()

	// Buttons (vertically stacked)
	buttonArea := fp.renderButtonArea()

	// Compose layout: header + (fileList | buttons) + info
	// File list takes most space, buttons on right side
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		fileList,
		buttonArea,
	)

	// Bottom info bar
	infoBar := fp.renderInfoBar()

	// Compose dialog: header + [filename] + main + info
	var dialog string
	if fp.saveMode {
		dialog = lipgloss.JoinVertical(
			lipgloss.Left,
			headerContent,
			filenameInput,
			mainContent,
			infoBar,
		)
	} else {
		dialog = lipgloss.JoinVertical(
			lipgloss.Left,
			headerContent,
			mainContent,
			infoBar,
		)
	}

	// Wrap in a styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(fp.Width)

	return boxStyle.Render(dialog)
}

// renderHeader produces the header with path only (no buttons now).
func (fp *FilePicker) renderHeader() string {
	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("60")).
		Bold(true).
		Width(fp.Width - 2) // Account for border

	path := pathStyle.Render(" 📁 " + fp.currentPath)
	return path
}

// renderFilenameInput produces the filename input field (save mode only).
func (fp *FilePicker) renderFilenameInput() string {
	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("236")).
		Padding(0, 2).
		Width(fp.Width - 6)

	if fp.Focused == 1 {
		inputStyle = inputStyle.Background(lipgloss.Color("60"))
	}

	label := "Filename: "
	cursor := ""
	if fp.Focused == 1 {
		cursor = "█"
	}

	content := label + fp.filename + cursor
	return inputStyle.Render(content)
}

// renderFileList produces the scrollable file list.
func (fp *FilePicker) renderFileList() string {
	if len(fp.files) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Padding(1, 2).
			Width(fp.vp.Width)
		return emptyStyle.Render("(empty directory)")
	}

	var lines []string

	for i, info := range fp.files {
		prefix := "  "
		if i == fp.selected && fp.Focused == 0 {
			prefix = "> " // selected indicator
		}

		// Show folder icon for directories
		icon := "📄"
		if info.IsDir() {
			icon = "📁"
		}

		line := prefix + icon + " " + info.Name()

		// Style the selected line
		if i == fp.selected && fp.Focused == 0 {
			lineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("60")).
				Bold(true)
			line = lineStyle.Render(line)
		}

		lines = append(lines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	fp.vp.SetContent(content)

	// Wrap viewport in a container with fixed width
	container := lipgloss.NewStyle().
		Width(fp.vp.Width).
		Height(fp.vp.Height)

	return container.Render(fp.vp.View())
}

// renderButtonArea produces vertically stacked OK and Cancel buttons.
func (fp *FilePicker) renderButtonArea() string {
	okStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("239")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("239")).
		Padding(0, 2).
		Width(10).
		Align(lipgloss.Center)

	cancelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("239")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("239")).
		Padding(0, 2).
		Width(10).
		Align(lipgloss.Center)

	// Adjust focus indices based on save mode
	okFocusIndex := 1
	cancelFocusIndex := 2
	if fp.saveMode {
		okFocusIndex = 2
		cancelFocusIndex = 3
	}

	if fp.Focused == okFocusIndex {
		okStyle = okStyle.Background(lipgloss.Color("60")).BorderForeground(lipgloss.Color("60"))
	}
	if fp.Focused == cancelFocusIndex {
		cancelStyle = cancelStyle.Background(lipgloss.Color("60")).BorderForeground(lipgloss.Color("60"))
	}

	okBtn := okStyle.Render("OK")
	cancelBtn := cancelStyle.Render("Cancel")

	// Stack buttons vertically with spacing
	buttons := lipgloss.JoinVertical(lipgloss.Left, okBtn, "", cancelBtn)

	// Wrap in container to ensure consistent height
	container := lipgloss.NewStyle().
		Width(14).
		Height(fp.vp.Height)

	return container.Render(buttons)
}

// renderInfoBar produces the status bar at the bottom.
func (fp *FilePicker) renderInfoBar() string {
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(1, 2)

	selected := fp.GetSelected()
	if selected == "" {
		selected = "(none)"
	}

	info := "Selected: " + selected + " | j/k navigate | enter select | u go up"
	return infoStyle.Render(info)
}
