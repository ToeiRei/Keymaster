// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cfg "github.com/toeirei/keymaster/config"
)

func resetViper() {
	// Reset global viper state between tests
	viper.Reset()
}

func TestLoadConfig_EmptyCandidate_TreatedAsNotFound(t *testing.T) {
	tmp := t.TempDir()
	// Force user config dir to tmp by setting XDG_CONFIG_HOME
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create the directory but write a zero-length file
	cfgDir := filepath.Join(tmp, "keymaster")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	emptyPath := filepath.Join(cfgDir, "keymaster.yaml")
	f, err := os.Create(emptyPath)
	if err != nil {
		t.Fatalf("create empty file: %v", err)
	}
	_ = f.Close()

	resetViper()
	defer resetViper()

	var _ cfg.Config
	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	_, err = cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, &emptyPath)
	if err == nil {
		t.Fatalf("expected ConfigFileNotFoundError for empty candidate, got nil")
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		t.Fatalf("expected ConfigFileNotFoundError, got: %T %v", err, err)
	}
}

func TestWriteConfigFile_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	resetViper()
	defer resetViper()

	c := cfg.Config{}
	c.Database.Type = "sqlite"
	c.Database.Dsn = "./keymaster.db"
	c.Language = "en"

	if err := cfg.WriteConfigFile(&c, false); err != nil {
		t.Fatalf("WriteConfigFile failed: %v", err)
	}

	path, err := cfg.GetConfigPath(false)
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file at %s, stat error: %v", path, err)
	}
}

func TestLoadConfig_ReadsExplicitFile(t *testing.T) {
	tmp := t.TempDir()
	yaml := "database:\n  type: postgres\n  dsn: postgresql://user@/db\nlanguage: de\n"
	file := filepath.Join(tmp, "cfg.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, &file)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if got.Database.Type != "postgres" {
		t.Fatalf("expected postgres, got %q", got.Database.Type)
	}
	if got.Language != "de" {
		t.Fatalf("expected de, got %q", got.Language)
	}
}

func TestLoadConfig_BrokenConfig_ReturnsParseError(t *testing.T) {
	tmp := t.TempDir()
	// Write a file containing a control character (0x01) which YAML forbids
	yaml := "language: en\n" + string([]byte{0x01}) + "\n"
	file := filepath.Join(tmp, "broken.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write broken file: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	_, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, &file)
	if err == nil {
		t.Fatalf("expected parse error for broken yaml, got nil")
	}
	if !strings.Contains(err.Error(), "control characters are not allowed") {
		t.Fatalf("expected control characters error, got: %v", err)
	}
}

func TestGetConfigPath(t *testing.T) {
	// Save original env vars to restore them later
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origProgData := os.Getenv("ProgramData")
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")

	defer func() {
		_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		_ = os.Setenv("ProgramData", origProgData)
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("USERPROFILE", origUserProfile)
	}()

	// Create a temporary directory to act as a fake home
	mockUserHome := t.TempDir()

	cases := []struct {
		name   string
		system bool
		setup  func() (string, error) // Returns expected path
		goos   string
	}{
		{
			name:   "user-linux-xdg",
			system: false,
			goos:   "linux",
			setup: func() (string, error) {
				_ = os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
				return "/tmp/xdg/keymaster/keymaster.yaml", nil
			},
		},
		{
			name:   "user-linux-no-xdg",
			system: false,
			goos:   "linux",
			setup: func() (string, error) {
				_ = os.Setenv("XDG_CONFIG_HOME", "")
				_ = os.Setenv("HOME", mockUserHome)
				// Manually construct the expected path based on a mocked home
				return filepath.Join(mockUserHome, ".config", "keymaster", "keymaster.yaml"), nil
			},
		},
		{
			name:   "system-linux",
			system: true,
			goos:   "linux",
			setup: func() (string, error) {
				return "/etc/keymaster/keymaster.yaml", nil
			},
		},
		{
			name:   "user-windows-xdg",
			system: false,
			goos:   "windows",
			setup: func() (string, error) {
				_ = os.Setenv("XDG_CONFIG_HOME", "C:\\tmp\\xdg")
				return "C:\\tmp\\xdg\\keymaster\\keymaster.yaml", nil
			},
		},
		{
			name:   "user-windows-no-xdg",
			system: false,
			goos:   "windows",
			setup: func() (string, error) {
				_ = os.Setenv("XDG_CONFIG_HOME", "")
				// On Windows, os.UserConfigDir() often relies on APPDATA.
				// Set it to a temp dir to ensure the test is isolated.
				mockAppData := t.TempDir()
				_ = os.Setenv("APPDATA", mockAppData)
				return filepath.Join(mockAppData, "keymaster", "keymaster.yaml"), nil
			},
		},
		{
			name:   "system-windows",
			system: true,
			goos:   "windows",
			setup: func() (string, error) {
				_ = os.Setenv("ProgramData", "C:\\ProgramData")
				return "C:\\ProgramData\\Keymaster\\keymaster.yaml", nil
			},
		},
	}

	for _, tt := range cases {
		// Only run tests that match the current OS
		if tt.goos != cfg.RuntimeOS {
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			// Reset env before each run
			_ = os.Setenv("XDG_CONFIG_HOME", "")
			_ = os.Setenv("ProgramData", "")
			_ = os.Setenv("HOME", "")
			_ = os.Setenv("USERPROFILE", "")

			expected, err := tt.setup()
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			path, err := cfg.GetConfigPath(tt.system)
			if err != nil {
				t.Fatalf("GetConfigPath() error = %v", err)
			}
			if path != expected {
				t.Errorf("GetConfigPath() = %v, want %v", path, expected)
			}
		})
	}
}
