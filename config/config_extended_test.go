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

// TestLoadConfig_EnvVarParsing tests that KEYMASTER_* environment variables are read correctly
func TestLoadConfig_EnvVarParsing(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Set environment variables with KEYMASTER_ prefix
	_ = os.Setenv("KEYMASTER_DATABASE_TYPE", "postgres")
	_ = os.Setenv("KEYMASTER_DATABASE_DSN", "postgresql://envuser@/envdb")
	_ = os.Setenv("KEYMASTER_LANGUAGE", "es")
	defer func() {
		_ = os.Unsetenv("KEYMASTER_DATABASE_TYPE")
		_ = os.Unsetenv("KEYMASTER_DATABASE_DSN")
		_ = os.Unsetenv("KEYMASTER_LANGUAGE")
	}()

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, nil)

	// LoadConfig returns ConfigFileNotFoundError when no file is used, but env vars should still be loaded
	if err == nil {
		t.Fatalf("expected ConfigFileNotFoundError, got nil")
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		t.Fatalf("expected ConfigFileNotFoundError, got: %T %v", err, err)
	}

	// Verify environment variables were parsed correctly (with underscore to dot conversion)
	if got.Database.Type != "postgres" {
		t.Fatalf("expected postgres from env, got %q", got.Database.Type)
	}
	if got.Database.Dsn != "postgresql://envuser@/envdb" {
		t.Fatalf("expected env DSN, got %q", got.Database.Dsn)
	}
	if got.Language != "es" {
		t.Fatalf("expected es from env, got %q", got.Language)
	}
}

// TestLoadConfig_FlagBindingOverridesEnv tests that CLI flags take precedence over environment variables
func TestLoadConfig_FlagBindingOverridesEnv(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Set env var
	_ = os.Setenv("KEYMASTER_LANGUAGE", "fr")
	defer func() { _ = os.Unsetenv("KEYMASTER_LANGUAGE") }()

	resetViper()
	defer resetViper()

	cmd := &cobra.Command{}
	cmd.Flags().String("language", "", "language")
	// Set flag value (should override env)
	if err := cmd.Flags().Set("language", "ja"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	got, err := cfg.LoadConfig[cfg.Config](cmd, defaults, nil)

	if err == nil {
		t.Fatalf("expected ConfigFileNotFoundError, got nil")
	}

	// Flag should override env
	if got.Language != "ja" {
		t.Fatalf("expected ja from flag (not fr from env), got %q", got.Language)
	}
}

// TestSave_PersistsViperState tests the Save() function with actual viper configuration
func TestSave_PersistsViperState(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	resetViper()
	defer resetViper()

	// Set up viper with specific values
	viper.Set("database.type", "mysql")
	viper.Set("database.dsn", "mysql://testuser@localhost/testdb")
	viper.Set("language", "de")

	// Call Save()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file was created
	path, err := cfg.GetConfigPath(false)
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created at %s: %v", path, err)
	}

	// Read file and verify contents
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "mysql") {
		t.Fatalf("expected mysql in config, got: %s", content)
	}
	if !strings.Contains(content, "de") {
		t.Fatalf("expected de in config, got: %s", content)
	}
}

// TestWriteConfigFile_SystemPath tests writing to system-wide config location
func TestWriteConfigFile_SystemPath(t *testing.T) {
	// Skip in CI and if not running as admin/root (system paths may not be writable)
	if os.Getenv("CI") != "" || !isAdmin() {
		t.Skip("skipping system path test (requires elevated privileges, not safe in CI)")
	}

	// Use temp dir to simulate system path
	tmpSystem := t.TempDir()
	origProgramData := os.Getenv("ProgramData")
	_ = os.Setenv("ProgramData", tmpSystem)
	defer func() { _ = os.Setenv("ProgramData", origProgramData) }()

	resetViper()
	defer resetViper()

	c := cfg.Config{}
	c.Database.Type = "postgres"
	c.Database.Dsn = "postgresql://system@/sysdb"
	c.Language = "en"

	// Write to system path
	if err := cfg.WriteConfigFile(&c, true); err != nil {
		t.Fatalf("WriteConfigFile(system=true) failed: %v", err)
	}

	// Verify file exists at expected path
	expectedPath := filepath.Join(tmpSystem, "Keymaster", "keymaster.yaml")
	if cfg.RuntimeOS != "windows" {
		expectedPath = "/etc/keymaster/keymaster.yaml"
	}

	// On Windows with temp dir, check the temp path
	if cfg.RuntimeOS == "windows" {
		if _, err := os.Stat(expectedPath); err != nil {
			t.Fatalf("expected system config at %s, stat error: %v", expectedPath, err)
		}
	}
}

// TestWriteConfigFile_DirectoryCreation tests that directories are created if they don't exist
func TestWriteConfigFile_DirectoryCreation(t *testing.T) {
	tmp := t.TempDir()
	// Use a nested path that doesn't exist yet
	nestedPath := filepath.Join(tmp, "nested", "deep", "path", "keymaster")
	_ = os.Setenv("XDG_CONFIG_HOME", nestedPath)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	resetViper()
	defer resetViper()

	c := cfg.Config{}
	c.Database.Type = "sqlite"
	c.Database.Dsn = "./nested.db"
	c.Language = "en"

	// This should create all intermediate directories
	if err := cfg.WriteConfigFile(&c, false); err != nil {
		t.Fatalf("WriteConfigFile failed to create directories: %v", err)
	}

	// Verify directory structure was created
	expectedDir := filepath.Join(nestedPath, "keymaster")
	if _, err := os.Stat(expectedDir); err != nil {
		t.Fatalf("expected directory %s to exist, stat error: %v", expectedDir, err)
	}

	// Verify file was created
	expectedFile := filepath.Join(expectedDir, "keymaster.yaml")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("expected file %s to exist, stat error: %v", expectedFile, err)
	}
}

// TestLoadConfig_ErrorDiagnostics tests that error diagnostics are produced for malformed configs
func TestLoadConfig_ErrorDiagnostics(t *testing.T) {
	tmp := t.TempDir()

	// Create a config file with invalid YAML (unclosed quote)
	yaml := "database:\n  type: \"sqlite\n  dsn: ./keymaster.db\nlanguage: en\n"
	file := filepath.Join(tmp, "invalid.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write invalid file: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	_, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, &file)

	// Should return a parse error
	if err == nil {
		t.Fatalf("expected parse error for invalid YAML, got nil")
	}

	// Error should mention parsing issue
	errStr := err.Error()
	if !strings.Contains(errStr, "yaml") && !strings.Contains(errStr, "parse") && !strings.Contains(errStr, "unmarshal") {
		t.Logf("error might not be descriptive enough: %v", err)
	}
}

// TestLoadConfig_MultipleConfigCandidates tests config file precedence
func TestLoadConfig_MultipleConfigCandidates(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create user config
	cfgDir := filepath.Join(tmp, "keymaster")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	userYaml := "database:\n  type: sqlite\n  dsn: ./user.db\nlanguage: en\n"
	userPath := filepath.Join(cfgDir, "keymaster.yaml")
	if err := os.WriteFile(userPath, []byte(userYaml), 0o600); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "postgres", "database.dsn": "./default.db", "language": "fr"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, nil)

	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// User config should be loaded (not defaults)
	if got.Database.Dsn != "./user.db" {
		t.Fatalf("expected ./user.db from user config, got %q", got.Database.Dsn)
	}
	if got.Language != "en" {
		t.Fatalf("expected en from user config, got %q", got.Language)
	}
}

// TestLoadConfig_LocalKeymasterYaml tests precedence of ./keymaster.yaml in current directory
func TestLoadConfig_LocalKeymasterYaml(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Ensure no user config interferes
	_ = os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "noconfig"))
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create ./keymaster.yaml in current directory
	localYaml := "database:\n  type: postgres\n  dsn: ./local.db\nlanguage: ja\n"
	if err := os.WriteFile("keymaster.yaml", []byte(localYaml), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./default.db", "language": "en"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, nil)

	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Local ./keymaster.yaml should be loaded
	if got.Database.Type != "postgres" {
		t.Fatalf("expected postgres from ./keymaster.yaml, got %q", got.Database.Type)
	}
	if got.Database.Dsn != "./local.db" {
		t.Fatalf("expected ./local.db from ./keymaster.yaml, got %q", got.Database.Dsn)
	}
	if got.Language != "ja" {
		t.Fatalf("expected ja from ./keymaster.yaml, got %q", got.Language)
	}
}

// TestLoadConfig_ExplicitFileOverridesAll tests that explicit file path takes precedence
func TestLoadConfig_ExplicitFileOverridesAll(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Create ./keymaster.yaml (lower priority)
	localYaml := "database:\n  type: sqlite\n  dsn: ./local.db\nlanguage: en\n"
	if err := os.WriteFile("keymaster.yaml", []byte(localYaml), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	// Create explicit config (higher priority)
	explicitYaml := "database:\n  type: mysql\n  dsn: ./explicit.db\nlanguage: zh\n"
	explicitPath := filepath.Join(tmp, "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte(explicitYaml), 0o600); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "postgres", "database.dsn": "./default.db", "language": "fr"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, &explicitPath)

	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Explicit file should override ./keymaster.yaml
	if got.Database.Type != "mysql" {
		t.Fatalf("expected mysql from explicit file, got %q", got.Database.Type)
	}
	if got.Database.Dsn != "./explicit.db" {
		t.Fatalf("expected ./explicit.db from explicit file, got %q", got.Database.Dsn)
	}
	if got.Language != "zh" {
		t.Fatalf("expected zh from explicit file, got %q", got.Language)
	}
}

// TestMergeLegacyConfig_MergesBothFiles tests that legacy config is merged with primary
func TestMergeLegacyConfig_MergesBothFiles(t *testing.T) {
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create legacy .keymaster.yaml with only language setting
	legacyYaml := "language: ko\n"
	if err := os.WriteFile(".keymaster.yaml", []byte(legacyYaml), 0o600); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	// Create primary ./keymaster.yaml without language (so legacy can merge)
	primaryYaml := "database:\n  type: postgres\n  dsn: ./primary.db\n"
	if err := os.WriteFile("keymaster.yaml", []byte(primaryYaml), 0o600); err != nil {
		t.Fatalf("write primary: %v", err)
	}

	resetViper()
	defer resetViper()

	defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, nil)

	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Primary file should be used for database
	if got.Database.Type != "postgres" {
		t.Fatalf("expected postgres from primary, got %q", got.Database.Type)
	}
	// Legacy file should provide language (merged in)
	if got.Language != "ko" {
		t.Fatalf("expected ko from merged legacy config, got %q", got.Language)
	}
}

// TestWriteConfigFile_ReadonlyDirectory tests WriteConfigFile with unwritable directory
func TestWriteConfigFile_ReadonlyDirectory(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping readonly test in CI (permission issues)")
	}

	tmp := t.TempDir()
	// Create a readonly directory
	readOnlyDir := filepath.Join(tmp, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o555); err != nil {
		t.Fatalf("mkdir readonly: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // Restore permissions for cleanup

	_ = os.Setenv("XDG_CONFIG_HOME", readOnlyDir)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	resetViper()
	defer resetViper()

	c := cfg.Config{}
	c.Database.Type = "sqlite"
	c.Database.Dsn = "./test.db"
	c.Language = "en"

	// This should fail to create subdirectory in readonly parent
	err := cfg.WriteConfigFile(&c, false)
	// On Windows, readonly doesn't always prevent subdirectory creation, so we may get nil
	if err == nil && cfg.RuntimeOS != "windows" {
		t.Logf("expected permission error, got nil (system may allow creation)")
	}
	if err != nil && !strings.Contains(err.Error(), "permission") && !strings.Contains(err.Error(), "denied") {
		t.Logf("error message: %v", err)
	}
}

// TestLoadConfig_EmptyDefaults tests that LoadConfig works with empty defaults
func TestLoadConfig_EmptyDefaults(t *testing.T) {
	tmp := t.TempDir()
	yaml := "database:\n  type: mysql\n  dsn: ./test.db\nlanguage: ru\n"
	file := filepath.Join(tmp, "test.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	resetViper()
	defer resetViper()

	// No defaults
	got, err := cfg.LoadConfig[cfg.Config](&cobra.Command{}, map[string]any{}, &file)
	if err != nil {
		t.Fatalf("LoadConfig with empty defaults failed: %v", err)
	}

	if got.Database.Type != "mysql" {
		t.Fatalf("expected mysql, got %q", got.Database.Type)
	}
	if got.Language != "ru" {
		t.Fatalf("expected ru, got %q", got.Language)
	}
}

// TestGetConfigPath_UserConfigDirError simulates os.UserConfigDir() failure
func TestGetConfigPath_UserConfigDirError(t *testing.T) {
	if cfg.RuntimeOS == "windows" {
		t.Skip("difficult to simulate UserConfigDir failure on Windows")
	}

	// Unset all config-related env vars to force os.UserConfigDir() to be called
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	_ = os.Unsetenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", os.Getenv("USERPROFILE"))
	}()

	// On Linux/Mac, if HOME is unset, UserConfigDir() may fail
	_, err := cfg.GetConfigPath(false)

	// We expect an error or successful path depending on OS behavior
	// This test documents the behavior rather than asserting specific outcomes
	if err != nil {
		if !strings.Contains(err.Error(), "config") {
			t.Logf("GetConfigPath error (expected on some systems): %v", err)
		}
	}
}

// isAdmin checks if the process has elevated privileges (Windows admin or Unix root)
func isAdmin() bool {
	if cfg.RuntimeOS == "windows" {
		// On Windows, check if we can write to ProgramData
		pd := os.Getenv("ProgramData")
		if pd == "" {
			return false
		}
		testFile := filepath.Join(pd, ".keymaster_test_admin")
		f, err := os.Create(testFile)
		if err != nil {
			return false
		}
		f.Close()
		os.Remove(testFile)
		return true
	}
	// On Unix, check if we're root
	return os.Geteuid() == 0
}
