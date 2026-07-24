// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package locales_test

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/locales"
	"gopkg.in/yaml.v3"
)

// managedPrefixes are the namespaces this test enforces full parity on across
// every locale. Legacy keys predate the parity guarantee and are not checked
// (the three files intentionally differ in their legacy key sets).
var managedPrefixes = []string{"errors.", "audit.action.", "keys.", "crud."}

func loadLocale(t *testing.T, name string) map[string]string {
	t.Helper()
	data, err := locales.LocaleFS.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var m map[string]string
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", name, err)
	}
	return m
}

func managed(key string) bool {
	for _, p := range managedPrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// TestManagedKeyParity ensures every errors.* / audit.action.* key defined in
// English also exists in de and art-x-ang, so a missing translation can never
// surface the raw message id to a non-English user.
func TestManagedKeyParity(t *testing.T) {
	en := loadLocale(t, "active.en.yaml")
	others := map[string]map[string]string{
		"de":        loadLocale(t, "active.de.yaml"),
		"art-x-ang": loadLocale(t, "active.art-x-ang.yaml"),
	}

	var enManaged []string
	for k := range en {
		if managed(k) {
			enManaged = append(enManaged, k)
		}
	}
	if len(enManaged) == 0 {
		t.Fatal("no managed keys found in active.en.yaml")
	}

	for lang, m := range others {
		for _, k := range enManaged {
			if _, ok := m[k]; !ok {
				t.Errorf("locale %q is missing managed key %q", lang, k)
			}
		}
		// And no managed key should exist only in a translation but not en.
		for k := range m {
			if managed(k) {
				if _, ok := en[k]; !ok {
					t.Errorf("locale %q has managed key %q not present in en", lang, k)
				}
			}
		}
	}
}

// TestManagedKeysNotSelfReferential guards against a value accidentally left
// equal to its key (a common copy-paste slip that the i18n fallback hides).
func TestManagedKeysNotSelfReferential(t *testing.T) {
	en := loadLocale(t, "active.en.yaml")
	for k, v := range en {
		if managed(k) && v == k {
			t.Errorf("en value for %q equals its key (untranslated)", k)
		}
	}
}
