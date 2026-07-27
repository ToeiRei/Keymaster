// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"testing"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util/keys"
)

// stubElement stands in for element.Text; the element package cannot be
// imported here, it imports this one.
type stubElement struct{ value string }

func (s *stubElement) Get() any                               { return s.value }
func (s *stubElement) Set(value any)                          { s.value, _ = value.(string) }
func (s *stubElement) Reset()                                 { s.value = "" }
func (s *stubElement) Init() (tea.Cmd, keys.KeyBindingList)   { return nil, nil }
func (s *stubElement) Update(tea.Msg) (tea.Cmd, Action)       { return nil, ActionNone }
func (s *stubElement) View(width int, eager bool) string      { return s.value }
func (s *stubElement) Focusable() bool                        { return true }
func (s *stubElement) Focus(parentKeyMap help.KeyMap) tea.Cmd { return nil }
func (s *stubElement) Blur()                                  {}

// stubButton stands in for element.Button, which reports no value at all.
type stubButton struct{ stubElement }

func (s *stubButton) Get() any      { return nil }
func (s *stubButton) Set(value any) {}

var _ FormElement = (*stubElement)(nil)
var _ FormElement = (*stubButton)(nil)

func newTestForm[T any](opts ...FormOpt[T]) (*Form[T], *stubElement, *stubElement) {
	username, host := &stubElement{}, &stubElement{}
	form := New(append([]FormOpt[T]{
		WithRowItem[T]("username", username),
		WithRowItem[T]("host", host),
		WithRow(WithItem[T]("_submit", &stubButton{})),
		WithOnDiscardGuard[T](func(confirmCmd tea.Cmd) tea.Cmd { return confirmCmd }),
	}, opts...)...)
	return &form, username, host
}

func TestGuardUnsavedChangesMapModel(t *testing.T) {
	// a nil InitialData must not read as dirty: Init normalizes it to the
	// values the elements actually hold
	form, username, _ := newTestForm[map[string]string]()
	form.Init()

	if cmd := form.guardUnsavedChanges(ActionCancel); cmd != nil {
		t.Error("guardUnsavedChanges() fired on an untouched form")
	}

	username.value = "root"
	if cmd := form.guardUnsavedChanges(ActionCancel); cmd == nil {
		t.Error("guardUnsavedChanges() did not fire after a change")
	}
}

func TestGuardUnsavedChangesStructModel(t *testing.T) {
	type model struct {
		Username string `form:"username"`
		Host     string `form:"host"`
		// no form item renders this, so it must not read as dirty
		Orphan string `form:"orphan"`
	}

	form, username, _ := newTestForm(WithInitialData(model{"root", "example.com", "value"}))
	form.Init()

	if cmd := form.guardUnsavedChanges(ActionCancel); cmd != nil {
		t.Error("guardUnsavedChanges() fired on an untouched form")
	}
	if username.value != "root" {
		t.Errorf("element value = %q, want %q", username.value, "root")
	}

	username.value = "admin"
	if cmd := form.guardUnsavedChanges(ActionCancel); cmd == nil {
		t.Error("guardUnsavedChanges() did not fire after a change")
	}
}

func TestGuardUnsavedChangesWithoutGuard(t *testing.T) {
	form := New(
		WithRowItem[map[string]string]("username", &stubElement{"root"}),
	)
	form.Init()

	if cmd := form.guardUnsavedChanges(ActionCancel); cmd != nil {
		t.Error("guardUnsavedChanges() fired without a DiscardGuard")
	}
}

func TestGetMapModel(t *testing.T) {
	form, username, host := newTestForm[map[string]string]()
	form.Init()
	username.value, host.value = "root", "example.com"

	data, err := form.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// _submit reports no value, so it must not appear
	if len(data) != 2 || data["username"] != "root" || data["host"] != "example.com" {
		t.Errorf("Get() = %#v", data)
	}
}

func TestSetMapModel(t *testing.T) {
	form, username, host := newTestForm[map[string]string]()
	form.Init()

	if err := form.Set(map[string]string{"host": "example.com"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if username.value != "" || host.value != "example.com" {
		t.Errorf("username = %q, host = %q", username.value, host.value)
	}
}
