// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package i18n_test

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/ui/i18n"
)

func TestLocalizedError_LazyResolution(t *testing.T) {
	i18n.Init("en")

	// A missing key falls back to the id itself, so this exercises the plumbing
	// without depending on a specific locale value.
	err := i18n.NewError("errors.__missing_test_key__")
	if err.Error() != "errors.__missing_test_key__" {
		t.Fatalf("unexpected Error(): %q", err.Error())
	}
	if err.MessageID() != "errors.__missing_test_key__" {
		t.Fatalf("unexpected MessageID(): %q", err.MessageID())
	}
}

func TestLocalizedError_FollowsLanguage(t *testing.T) {
	i18n.Init("en")

	// "all" exists in both en ("All") and de ("Alle"); build under en, resolve under de.
	err := i18n.NewError("all")
	if got := err.Error(); got != "All" {
		t.Fatalf("expected 'All', got %q", got)
	}
	i18n.SetLang("de")
	if got := err.Error(); got != "Alle" {
		t.Fatalf("expected lazy resolution to 'Alle', got %q", got)
	}
	i18n.SetLang("en")
}

func TestWrapError_UnwrapAndArgs(t *testing.T) {
	i18n.Init("en")

	sentinel := errors.New("boom")
	err := i18n.WrapError(sentinel, "errors.__missing_wrap_key__")
	if !errors.Is(err, sentinel) {
		t.Fatal("expected wrapped sentinel to be discoverable via errors.Is")
	}
}

func TestNewError_IsByPointerIdentity(t *testing.T) {
	var sentinel error = i18n.NewError("errors.__missing_sentinel__")
	wrapped := i18n.WrapError(sentinel, "errors.__missing_outer__")
	if !errors.Is(wrapped, sentinel) {
		t.Fatal("expected pointer-identity errors.Is to match the sentinel")
	}
}

func TestTAuditAction(t *testing.T) {
	i18n.Init("en")

	tests := []struct {
		code string
		want string
	}{
		{"", "-"},
		{"__UNKNOWN_ACTION__", "Unknown Action"},        // humanized fallback (uppercase)
		{"some.unknown.action", "Some Unknown Action"},  // humanized fallback (dotted)
	}
	for _, tc := range tests {
		if got := i18n.TAuditAction(tc.code); got != tc.want {
			t.Fatalf("TAuditAction(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}
