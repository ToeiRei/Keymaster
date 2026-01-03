package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFlattenYAMLAndLoadKeys(t *testing.T) {
	// Create nested map and flatten
	m := map[string]interface{}{
		"top": map[string]interface{}{
			"sub": "value",
			"arr": []interface{}{"one", "two"},
		},
		"other": "v",
	}
	keys := make(map[string]struct{})
	flattenYAML("", m, keys)
	if _, ok := keys["top.sub"]; !ok {
		t.Fatalf("expected top.sub in keys")
	}
	if _, ok := keys["top.arr[0]"]; !ok {
		t.Fatalf("expected top.arr[0] in keys")
	}

	// Write YAML to temp file and load via loadKeysFromLocale
	dir := t.TempDir()
	p := filepath.Join(dir, "test.yaml")
	data, _ := yaml.Marshal(m)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	got, err := loadKeysFromLocale(p)
	if err != nil {
		t.Fatalf("loadKeysFromLocale failed: %v", err)
	}
	if _, ok := got["top.sub"]; !ok {
		t.Fatalf("expected loaded key top.sub")
	}
}

func TestFindUsedKeysAndUntranslatedStrings(t *testing.T) {
	dir := t.TempDir()
	// Create a Go file that contains i18n.T and some string literals
	src := `package foo
func f(){
	_ = i18n.T("my.key")
	foo("Visible message")
	bar("ok")
}`
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := filepath.Join(dir, "sub", "a.go")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatalf("write go: %v", err)
	}

	used, err := findUsedKeys(dir)
	if err != nil {
		t.Fatalf("findUsedKeys failed: %v", err)
	}
	if _, ok := used["my.key"]; !ok {
		t.Fatalf("expected my.key found in used keys")
	}

	// Prepare primary keys map (simulate loaded keys)
	all := map[string]struct{}{"my.key": {}}

	untranslated, err := findUntranslatedStrings(dir, used, all)
	if err != nil {
		t.Fatalf("findUntranslatedStrings failed: %v", err)
	}
	// Should find "Visible message"
	if _, ok := untranslated["Visible message"]; !ok {
		t.Fatalf("expected Visible message to be flagged as untranslated")
	}
	// Short string should be ignored
	if _, ok := untranslated["Short"]; ok {
		t.Fatalf("did not expect Short to be flagged")
	}
}
