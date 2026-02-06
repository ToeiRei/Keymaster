// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package i18n

import (
	"testing"
)

func TestInitAndAvailableLocales(t *testing.T) {
	Init("en")
	if GetLang() != "en" {
		t.Fatalf("expected lang 'en', got %q", GetLang())
	}

	av := GetAvailableLocales()
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
	Init("en")

	if got := T("all"); got != "All" {
		t.Fatalf("expected 'All', got %q", got)
	}

	// fmt-style formatting via non-map template args
	got := T("dashboard.hosts_current_key", 7)
	if got != "Hosts using current key: 7" {
		t.Fatalf("unexpected formatted translation: %q", got)
	}

	// switch language to German
	SetLang("de")
	if GetLang() != "de" {
		t.Fatalf("expected lang 'de', got %q", GetLang())
	}
	if got := T("all"); got != "Alle" {
		t.Fatalf("expected German 'Alle', got %q", got)
	}
}
