// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package i18n_test

import (
	"testing"

	"github.com/toeirei/keymaster/ui/i18n"
)

func TestInitAndAvailableLocales(t *testing.T) {
	i18n.Init("en")
	if i18n.GetLang() != "en" {
		t.Fatalf("expected lang 'en', got %q", i18n.GetLang())
	}

	av := i18n.GetAvailableLocales()
	wantKeys := []string{"en", "de", "art-x-ang"}
	for _, k := range wantKeys {
		if _, ok := av[k]; !ok {
			t.Fatalf("expected available locale %q to be present", k)
		}
	}

	// special display name for art-x-ang
	if name, ok := av["art-x-ang"]; !ok || name != "Ænglisc (Olde English)" {
		t.Fatalf("unexpected display name for art-x-ang: %v", av["art-x-ang"])
	}
}

func TestT_BasicAndFormatting(t *testing.T) {
	i18n.Init("en")

	if got := i18n.T("all"); got != "All" {
		t.Fatalf("expected 'All', got %q", got)
	}

	// fmt-style formatting via non-map template args
	got := i18n.T("dashboard.hosts_current_key", 7)
	if got != "Current: 7" {
		t.Fatalf("unexpected formatted translation: %q", got)
	}

	// switch language to German
	i18n.SetLang("de")
	if i18n.GetLang() != "de" {
		t.Fatalf("expected lang 'de', got %q", i18n.GetLang())
	}
	if got := i18n.T("all"); got != "Alle" {
		t.Fatalf("expected German 'Alle', got %q", got)
	}
}
