// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package popup

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/toeirei/keymaster/ui/tui/util"
)

const (
	reservedHeight int = 2
	reservedWidth  int = 6
)

type popup struct {
	model   *util.Model
	onClose func(*util.Model) tea.Cmd
}

type Injector struct {
	child  *util.Model
	popups []popup
	size   util.Size
}

func NewInjector(child *util.Model) *Injector {
	return &Injector{
		child: child,
	}
}

func (m Injector) Init() tea.Cmd {
	return (*m.child).Init()
}

func (m *Injector) Update(msg tea.Msg) tea.Cmd {
	if m.size.Update(msg) {
		if len(m.popups) > 0 {
			return tea.Batch(
				(*m.activeModel()).Update(tea.WindowSizeMsg{
					Width:  m.size.Width - reservedWidth,
					Height: m.size.Height - reservedHeight,
				}),
				(*m.child).Update(msg),
			)
		}
		return (*m.child).Update(msg)
	}

	switch msg := msg.(type) {
	case openMsg:
		return m.open(popup{
			model:   msg.Model,
			onClose: msg.OnClose,
		})
	case closeMsg:
		return m.close()
	}

	return (*m.activeModel()).Update(msg)
}

func (m *Injector) applyView(v1, v2 string) string {
	v1_width, v1_height := lipgloss.Size(v1)
	// limit v2 dimensions to v1
	v2 = lipgloss.NewStyle().MaxWidth(v1_width).MaxHeight(v1_height).Render(v2)
	v2_width, v2_height := lipgloss.Size(v2)

	offset_left := (v1_width - v2_width) / 2
	offset_top := (v1_height - v2_height) / 2

	v1_lines := strings.Split(v1, "\n")
	v2_lines := strings.Split(v2, "\n")

	for i := range v2_lines {
		v1_left := ansi.Truncate(v1_lines[i+offset_top], offset_left, "")
		v1_right := ansi.TruncateLeft(v1_lines[i+offset_top], offset_left+v2_width, "")
		v1_lines[i+offset_top] = v1_left + v2_lines[i] + v1_right
	}

	return strings.Join(v1_lines, "\n")
}

func (m Injector) View() string {
	childView := (*m.child).View()

	if len(m.popups) > 0 {
		popupView := lipgloss.
			NewStyle().
			Padding(0, 1).
			Border(lipgloss.NormalBorder()).
			Margin(0, 1).
			Render((*m.activeModel()).View())

		childView = lipgloss.
			NewStyle().
			Foreground(lipgloss.AdaptiveColor{
				Light: "#DDDADA",
				Dark:  "#3C3C3C",
			}).
			Render(ansi.Strip(childView))

		return m.applyView(childView, popupView)
		// return lipgloss.
		// 	NewStyle().
		// 	MaxWidth(m.size.Width).
		// 	MaxHeight(m.size.Height).
		// 	Render(m.applyView(childView, popupView))
	}
	return childView
}

func (m *Injector) Focus() (tea.Cmd, help.KeyMap) {
	return (*m.activeModel()).Focus()
}
func (m *Injector) Blur() {
	(*m.activeModel()).Blur()
}

// *Model implements util.Model
var _ util.Model = (*Injector)(nil)

func (m *Injector) open(popup popup) tea.Cmd {
	// blur active view
	m.blurActiveModel()
	// append new popup
	m.popups = append(m.popups, popup)
	// init and focus new popup
	return tea.Batch(
		(*popup.model).Init(),
		m.focusActiveModel(),
		(*m.activeModel()).Update(m.size.ToMsg()),
	)
}

func (m *Injector) close() tea.Cmd {
	// popup left to close?
	if len(m.popups) > 0 {
		// blur old popup
		m.blurActiveModel()
		// pop old popup
		var onCloseCmd tea.Cmd
		if popup := m.popups[len(m.popups)-1]; popup.onClose != nil {
			onCloseCmd = popup.onClose(popup.model)
		}
		m.popups = m.popups[:len(m.popups)-1]
		// focus underlying view
		return tea.Batch(
			m.focusActiveModel(),
			onCloseCmd,
		)
	}
	return nil
}

func (m *Injector) activeModel() *util.Model {
	if len(m.popups) > 0 {
		return m.popups[len(m.popups)-1].model
	} else {
		return m.child
	}
}
func (m *Injector) focusActiveModel() tea.Cmd {
	cmd, keyMap := m.Focus()
	return tea.Batch(cmd, util.AnnounceKeyMapCmd(keyMap))
}
func (m *Injector) blurActiveModel() {
	m.Blur()
}
