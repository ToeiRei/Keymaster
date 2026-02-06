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
	Focus(help.KeyMap) tea.Cmd
	Blur()
}

func TryFocusTeaModel(m *tea.Model, baseKeyMap help.KeyMap) (tea.Cmd, error) {
	_m := *m
	if focusable, ok := _m.(Focusable); ok {
		cmd := focusable.Focus(baseKeyMap)
		*m = focusable.(tea.Model)
		return cmd, nil
	} else {
		return nil, fmt.Errorf("type %T does not implement Focusable interface", m)
	}
}
func TryBlurTeaModel(m *tea.Model) error {
	_m := *m
	if focusable, ok := _m.(Focusable); ok {
		focusable.Blur()
		*m = focusable.(tea.Model)
		return nil
	} else {
		return fmt.Errorf("type %T does not implement Focusable interface", m)
	}
}

type AnnounceKeyMapMsg struct {
	KeyMap help.KeyMap
}

func AnnounceKeyMapCmd(keyMaps ...help.KeyMap) tea.Cmd {
	return func() tea.Msg {
		return AnnounceKeyMapMsg{KeyMap: MergeKeyMaps(keyMaps...)}
	}
}

func MergeKeyMaps(keyMaps ...help.KeyMap) help.KeyMap {
	if len(keyMaps) == 1 {
		return keyMaps[0]
	}
	return MergedKeyMaps{KeyMaps: keyMaps}
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
