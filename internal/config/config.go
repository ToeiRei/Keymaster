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

// getConfigPath returns the full path for the configuration file.
func getConfigPath(system bool) (string, error) {
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
	v := viper.New()

	// 1. Set defaults
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	// 2. Set up file search paths (new format: keymaster.yaml)
	v.SetConfigName("keymaster")
	v.SetConfigType("yaml")

	// 3. Add explicit config file path if provided via --config flag.
	// This has the highest precedence for file-based configuration.
	if additional_config_file_path != nil {
		v.SetConfigFile(*additional_config_file_path)
	}

	// 3. Add standard config locations
	if userConfigPath, err := getConfigPath(false); err == nil {
		v.AddConfigPath(filepath.Dir(userConfigPath))
	}
	if systemConfigPath, err := getConfigPath(true); err == nil {
		v.AddConfigPath(filepath.Dir(systemConfigPath))
	}
	v.AddConfigPath(".") // Look for keymaster.yaml in current dir

	// 5. Read in the primary config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if the file is not found, but other errors are fatal.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return c, err
		}
	}

	// 6. For backward compatibility, check for and merge `.keymaster.yaml` in the current directory.
	mergeLegacyConfig(v)

	// 7. Read from environment variables
	v.AutomaticEnv()
	v.AllowEmptyEnv(true)
	v.SetEnvPrefix("keymaster")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// cli
	// TODO maybe needs to trigger additional parsing beferohand (most likely nots)
	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return c, err
	}

	// parse config
	if err := v.Unmarshal(&c); err != nil {
		return c, err
	}

	return c, nil
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
	path, err := getConfigPath(system)
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
