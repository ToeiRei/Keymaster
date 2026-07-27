// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package formpopup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/helpers/form"
)

type innerMsg struct{}

func innerCmd() tea.Msg { return innerMsg{} }

// New wraps the hosted form's callbacks so the popup closes itself. popup's
// close message is unexported, so these assert the wrapper's contract: that it
// calls through, and that it only appends to the command when the form resolved.

func TestNew_SubmitWrapsTheCallersCommand(t *testing.T) {
	var got string
	popupForm := New(form.New(
		form.WithOnSubmit(func(result map[string]string, err error) (tea.Cmd, bool) {
			got = result["username"]
			return innerCmd, true
		}),
	))

	cmd, valid := popupForm.form.OnSubmit(map[string]string{"username": "root"}, nil)
	if !valid {
		t.Error("a valid submit was reported invalid")
	}
	if got != "root" {
		t.Errorf("the caller's OnSubmit saw result %q, want %q", got, "root")
	}
	if cmd == nil {
		t.Fatal("expected a command closing the popup")
	}
	if _, ok := cmd().(innerMsg); ok {
		t.Error("expected the caller's command to be sequenced after the close, not returned bare")
	}
}

func TestNew_InvalidSubmitLeavesThePopupOpen(t *testing.T) {
	popupForm := New(form.New(
		form.WithOnSubmit(func(result map[string]string, err error) (tea.Cmd, bool) {
			return innerCmd, false
		}),
	))

	cmd, valid := popupForm.form.OnSubmit(nil, nil)
	if valid {
		t.Error("an invalid submit was reported valid")
	}
	if cmd == nil {
		t.Fatal("expected the caller's command to survive an invalid submit")
	}
	// Returned bare: nothing was appended, so the popup stays open.
	if _, ok := cmd().(innerMsg); !ok {
		t.Errorf("expected only the caller's command, got %T", cmd())
	}
}

func TestNew_CancelWrapsTheCallersCommand(t *testing.T) {
	var called bool
	popupForm := New(form.New(
		form.WithOnCancel[map[string]string](func() tea.Cmd {
			called = true
			return innerCmd
		}),
	))

	cmd := popupForm.form.OnCancel()
	if !called {
		t.Error("the caller's OnCancel was not called")
	}
	if cmd == nil {
		t.Fatal("expected a command closing the popup")
	}
	if _, ok := cmd().(innerMsg); ok {
		t.Error("expected the caller's command to be sequenced after the close, not returned bare")
	}
}

// A form built without callbacks must still dismiss its popup; several callers
// only want the close, and choicepopup sets neither callback at all.
func TestNew_ClosesWithoutCallbacks(t *testing.T) {
	popupForm := New(form.New[map[string]string]())

	if cmd, valid := popupForm.form.OnSubmit(nil, nil); cmd == nil || !valid {
		t.Errorf("OnSubmit without a caller callback: cmd=%v valid=%v", cmd, valid)
	}
	if cmd := popupForm.form.OnCancel(); cmd == nil {
		t.Error("OnCancel without a caller callback returned no command")
	}
}
