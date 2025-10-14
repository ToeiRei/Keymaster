package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds the application's configuration, loaded from file/env/flags.
type Config struct {
	Database struct {
		Type string `mapstructure:"type"`
		Dsn  string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Language string `mapstructure:"language"`
}

// GetConfigPath returns the full path for the configuration file.
func GetConfigPath(system bool) (string, error) {
	var configDir string
	var err error

	if system {
		// System-wide configuration paths
		switch runtime.GOOS {
		case "windows":
			configDir = filepath.Join(os.Getenv("ProgramData"), "Keymaster")
		default: // Linux, macOS, etc.
			configDir = "/etc/keymaster"
		}
	} else {
		// User-specific configuration paths
		configDir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("could not get user config directory: %w", err)
		}
		configDir = filepath.Join(configDir, "keymaster")
	}

	return filepath.Join(configDir, "keymaster.yaml"), nil
}

func LoadConfig[T any](cmd *cobra.Command, defaults map[string]any, additional_config_file_path *string) (T, error) {
	var c T

	// 1. Set defaults
	for key, value := range defaults {
		viper.SetDefault(key, value)
	}

	// 2. Set up file search paths (new format: keymaster.yaml)
	viper.SetConfigName("keymaster")
	viper.SetConfigType("yaml")

	// 3. Add explicit config file path if provided via --config flag.
	// This has the highest precedence for file-based configuration.
	if additional_config_file_path != nil {
		viper.SetConfigFile(*additional_config_file_path)
	}

	// 3. Add standard config locations
	if userConfigPath, err := GetConfigPath(false); err == nil {
		viper.AddConfigPath(filepath.Dir(userConfigPath))
	}
	if systemConfigPath, err := GetConfigPath(true); err == nil {
		viper.AddConfigPath(filepath.Dir(systemConfigPath))
	}
	viper.AddConfigPath(".") // Look for keymaster.yaml in current dir

	// 5. Read in the primary config file.
	// We declare readErr here so it's available in the function's scope for the final return.
	readErr := viper.ReadInConfig()
	if readErr != nil {
		// It's okay if the file is not found, but other errors are fatal.
		if _, ok := readErr.(viper.ConfigFileNotFoundError); !ok {
			return c, readErr
		}
	}

	// 6. For backward compatibility, check for and merge `.keymaster.yaml` in the current directory.
	mergeLegacyConfig(viper.GetViper())

	// 7. Read from environment variables
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)
	viper.SetEnvPrefix("keymaster")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// cli
	// TODO maybe needs to trigger additional parsing beferohand (most likely nots)
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return c, err
	}

	// parse config
	if err := viper.Unmarshal(&c); err != nil {
		return c, err
	}

	// Return the readErr (which will be nil or ConfigFileNotFoundError)
	return c, readErr
}

// mergeLegacyConfig checks for a `.keymaster.yaml` file in the current directory
// and merges it into the viper configuration if found. This is for backward compatibility.
func mergeLegacyConfig(v *viper.Viper) {
	legacyConfigFile := ".keymaster.yaml"
	if _, err := os.Stat(legacyConfigFile); err == nil {
		// File exists, let's try to merge it.
		v.SetConfigFile(legacyConfigFile)
		// MergeInConfig will not error on file not found, but we already checked.
		// It will error on a malformed file, which is the desired behavior.
		// We can ignore the error for this compatibility layer to avoid breaking startup.
		_ = v.MergeInConfig()
		// Reset the config file path to avoid side effects.
		v.SetConfigFile("")
	}
}

func WriteConfigFile[T any](c *T, system bool) error {
	path, err := GetConfigPath(system)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("could not create config directory %s: %w", configDir, err)
	}

	err = os.WriteFile(path, data, 0600) // Use 0600 for security, as it may contain secrets
	if err != nil {
		return err
	}

	return nil
}

// Save persists the current Viper configuration to the user's config file.
// It unmarshals the current state into the Config struct to ensure the
// file structure (e.g., nested database keys) is preserved.
func Save() error {
	var currentConfig Config
	if err := viper.Unmarshal(&currentConfig); err != nil {
		return fmt.Errorf("failed to unmarshal current config for saving: %w", err)
	}

	// Write the structured config to the user-specific file.
	return WriteConfigFile(&currentConfig, false)
}
