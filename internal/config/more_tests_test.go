// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cfg "github.com/toeirei/keymaster/internal/config"
)

func TestMergeLegacyConfigViaLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	// Change working dir to tmp so legacy file is in CWD
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Ensure user config dir points at tmp so tests are isolated from
	// any real user config that may exist on the runner.
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create legacy config file
	yaml := "language: fr\n"
	if err := os.WriteFile(filepath.Join(tmp, ".keymaster.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, nil)
	// LoadConfig may return ConfigFileNotFoundError when no explicit candidate
	// was used even if the legacy config was merged. Ensure legacy merge took place.
	if err == nil {
		t.Fatalf("expected ConfigFileNotFoundError (legacy merge returns not-found), got nil")
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		t.Fatalf("expected ConfigFileNotFoundError, got: %T %v", err, err)
	}
	if got.Language != "fr" {
		t.Fatalf("expected language fr from legacy config, got %q", got.Language)
	}
}

func TestSavePersistsViperState(t *testing.T) {
	tmp := t.TempDir()
	// Ensure user config dir points at tmp
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	resetViper()
	defer resetViper()

	// Set some viper values that Save should persist
	viper.Set("database.type", "sqlite")
	viper.Set("database.dsn", "./persisted.db")
	viper.Set("language", "es")

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	path, err := cfg.GetConfigPath(false)
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected saved config at %s, stat error: %v", path, err)
	}
}
