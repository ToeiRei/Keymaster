// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
// Package popup provides popup components and helpers for the TUI.
package popup

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

type PopupModel interface {
	util.Model
}

type openMsg struct {
	Model   *util.Model
	OnClose func(*util.Model) tea.Cmd
}

type closeMsg struct{}

// TODO add possibility to pass title
// TODO add option parameters for title callback and other popup extensions
func Open(m *util.Model) tea.Cmd {
	// Open opens a popup with the given message.
	return func() tea.Msg { return openMsg{Model: m} }
}
func OpenWithCallback(m *util.Model, cb func(*util.Model) tea.Cmd) tea.Cmd {
	return func() tea.Msg { return openMsg{Model: m, OnClose: cb} }
}

func Close() tea.Cmd {
	return func() tea.Msg { return closeMsg{} }
}
