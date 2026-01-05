// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

// GOOS_RUNTIME is the runtime OS, exposed for testing.
const GOOS_RUNTIME = runtime.GOOS

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
		// Allow XDG_CONFIG_HOME override for testing and cross-platform consistency
		if env := os.Getenv("XDG_CONFIG_HOME"); env != "" {
			configDir = env
		} else {
			configDir, err = os.UserConfigDir()
			if err != nil {
				return "", fmt.Errorf("could not get user config directory: %w", err)
			}
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

	// 2. We'll attempt to read explicit candidate files (with .yaml extension)
	// instead of letting Viper search the filesystem. This prevents Viper from
	// accidentally trying to parse non-config files (e.g., a local `keymaster`
	// binary) that happen to exist in the working directory.
	viper.SetConfigType("yaml")

	// 5. Read in the primary config file.
	// If any candidate config file exists but is zero-length, treat it as
	// "not found" to avoid YAML parse errors on empty files. We declare
	// readErr here so it's available in the function's scope for the final return.
	var readErr error

	// Build a list of candidate config file paths to check for zero-length files.
	var candidateFiles []string
	if additional_config_file_path != nil {
		candidateFiles = append(candidateFiles, *additional_config_file_path)
	} else {
		if userConfigPath, err := GetConfigPath(false); err == nil {
			candidateFiles = append(candidateFiles, userConfigPath)
		}
		if systemConfigPath, err := GetConfigPath(true); err == nil {
			candidateFiles = append(candidateFiles, systemConfigPath)
		}
		candidateFiles = append(candidateFiles, "./keymaster.yaml")
	}

	// Try explicit candidate files in order. If a candidate exists and has
	// non-zero size, tell Viper to use that exact file and read it. If none
	// exist, treat as ConfigFileNotFound.
	foundConfig := false
	for _, p := range candidateFiles {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(p); err == nil {
			if fi.Size() == 0 {
				log.Printf("candidate config: %s (empty)", p)
				// treat empty as absent
				continue
			}
			// Attempt to read this explicit file
			viper.SetConfigFile(p)
			if rerr := viper.ReadInConfig(); rerr != nil {
				// If parsing failed, surface diagnostic information
				log.Printf("failed reading config %s: %v", p, rerr)
				// dump candidate info
				log.Printf("candidate config: %s (size %d)", p, fi.Size())
				return c, rerr
			}
			foundConfig = true
			readErr = nil
			break
		}
	}
	if !foundConfig {
		readErr = viper.ConfigFileNotFoundError{}
	}

	// 6. For backward compatibility, check for and merge `.keymaster.yaml` in the current directory.
	mergeLegacyConfig(viper.GetViper())

	// If a config file was successfully read/merged, log it for easier debugging.
	used := viper.ConfigFileUsed()
	if used != "" {
		log.Printf("using config %s", used)
	} else {
		// If none was used, check whether any candidate existed but was zero-length
		// (helpful for debugging cases where an empty file causes defaults to be used).
		foundEmpty := ""
		for _, p := range candidateFiles {
			if p == "" {
				continue
			}
			if fi, err := os.Stat(p); err == nil {
				if fi.Size() == 0 {
					foundEmpty = p
					break
				}
			}
		}
		if foundEmpty != "" {
			log.Printf("using config none (found zero-length config %s)", foundEmpty)
		} else {
			log.Printf("using config none (defaults)")
		}
	}

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
		log.Printf("viper.Unmarshal error: %v", err)
		if used := viper.ConfigFileUsed(); used != "" {
			log.Printf("viper.ConfigFileUsed: %s", used)
			if data, rerr := os.ReadFile(used); rerr == nil {
				sample := data
				if len(sample) > 512 {
					sample = sample[:512]
				}
				log.Printf("first bytes of %s: %q", used, sample)
			} else {
				log.Printf("could not read used config file %s: %v", used, rerr)
			}
		}
		// Also dump candidate files if present
		for _, p := range candidateFiles {
			if p == "" {
				continue
			}
			if fi, err := os.Stat(p); err == nil {
				if fi.Size() > 0 {
					if data, rerr := os.ReadFile(p); rerr == nil {
						sample := data
						if len(sample) > 256 {
							sample = sample[:256]
						}
						log.Printf("first bytes of candidate %s: %q", p, sample)
					} else {
						log.Printf("could not read candidate %s: %v", p, rerr)
					}
				} else {
					log.Printf("candidate %s is empty", p)
				}
			} else {
				log.Printf("candidate %s not present", p)
			}
		}
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
		if err := v.MergeInConfig(); err != nil {
			log.Printf("error merging legacy config %s: %v", legacyConfigFile, err)
		} else {
			log.Printf("using config %s", legacyConfigFile)
		}
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
