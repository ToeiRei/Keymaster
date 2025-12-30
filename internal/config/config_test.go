package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cfg "github.com/toeirei/keymaster/internal/config"
)

func resetViper() {
    // Reset global viper state between tests
    viper.Reset()
}

func TestLoadConfig_EmptyCandidate_TreatedAsNotFound(t *testing.T) {
    tmp := t.TempDir()
    // Force user config dir to tmp by setting XDG_CONFIG_HOME
    os.Setenv("XDG_CONFIG_HOME", tmp)
    defer os.Unsetenv("XDG_CONFIG_HOME")

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
    f.Close()

    resetViper()
    defer resetViper()

    var _ cfg.Config
    defaults := map[string]any{"database.type": "sqlite", "database.dsn": "./keymaster.db", "language": "en"}
    _, err = cfg.LoadConfig[cfg.Config](&cobra.Command{}, defaults, nil)
    if err == nil {
        t.Fatalf("expected ConfigFileNotFoundError for empty candidate, got nil")
    }
    if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
        t.Fatalf("expected ConfigFileNotFoundError, got: %T %v", err, err)
    }
}

func TestWriteConfigFile_CreatesFile(t *testing.T) {
    tmp := t.TempDir()
    os.Setenv("XDG_CONFIG_HOME", tmp)
    defer os.Unsetenv("XDG_CONFIG_HOME")

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
