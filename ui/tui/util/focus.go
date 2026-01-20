// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	"fmt"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/util/slicest"
)

type Focusable interface {
	Focus() (tea.Cmd, help.KeyMap)
	Blur()
}

func TryFocusModel(m *Model) (tea.Cmd, help.KeyMap, error) {
	_m := *m
	if focusable, ok := _m.(Focusable); ok {
		cmd, keyMap := focusable.Focus()
		*m = focusable.(Model)
		return cmd, keyMap, nil
	} else {
		return nil, nil, fmt.Errorf("type %T does not implement Focusable interface", m)
	}
}
func TryBlurModel(m *Model) error {
	_m := *m
	if focusable, ok := _m.(Focusable); ok {
		focusable.Blur()
		*m = focusable.(Model)
		return nil
	} else {
		return fmt.Errorf("type %T does not implement Focusable interface", m)
	}
}

type AnnounceKeyMapMsg struct {
	KeyMap help.KeyMap
}

func AnnounceKeyMapCmd(k help.KeyMap) tea.Cmd {
	return func() tea.Msg {
		return AnnounceKeyMapMsg{KeyMap: k}
	}
}

func MergeKeyMaps(keymaps ...help.KeyMap) help.KeyMap {
	return MergedKeyMaps{KeyMaps: keymaps}
}

type MergedKeyMaps struct {
	KeyMaps []help.KeyMap
}

func (m MergedKeyMaps) ShortHelp() []key.Binding {
	bindings := slicest.Map(m.KeyMaps, func(k help.KeyMap) []key.Binding {
		if k != nil {
			return k.ShortHelp()
		}
		return nil
	})
	return slices.Concat(bindings...)
}

func (m MergedKeyMaps) FullHelp() [][]key.Binding {
	groups := slicest.Map(m.KeyMaps, func(k help.KeyMap) [][]key.Binding {
		if k != nil {
			return k.FullHelp()
		}
		return nil
	})
	return slices.Concat(groups...)
}

var _ help.KeyMap = (*MergedKeyMaps)(nil)
