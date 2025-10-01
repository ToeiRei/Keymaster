package config

import (
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func configPath(system bool) (string, error) {
	if system {
		// TODO make os aware
		return "/etc/keymaster", nil
	} else {
		return os.UserConfigDir()
	}
}

func LoadConfig[T any](cmd *cobra.Command, defaults map[string]any, additional_config_file_path *string) (T, error) {
	var c T
	v := viper.New()

	// defaults
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	// files (first file found wins)
	v.SetConfigName("keymaster")
	v.SetConfigType("yaml")
	if additional_config_file_path != nil {
		v.AddConfigPath(*additional_config_file_path)
	}
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home + "/.config")
	}
	v.AddConfigPath("/etc/keymaster")

	// env
	v.AutomaticEnv()
	v.AllowEmptyEnv(true)
	v.SetEnvPrefix("keymaster")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// cli
	// TODO maybe needs to trigger additional parsing beferohand (most likely nots)
	v.BindPFlags(cmd.Flags())

	// TODO maybe not needed
	// if err := v.ReadInConfig(); err != nil {
	// 	if _, ok := err.(v.ConfigFileNotFoundError); !ok {
	// 		return nil, err
	// 	}
	// }

	// parse config
	if err := v.Unmarshal(&c); err != nil {
		return c, err
	}

	return c, nil
}

func WriteConfigFile[T any](c *T, system bool) error {
	path, err := configPath(system)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// TODO recursively create directory if not present

	// TODO review permissions, because database secrets may be saved here in user mode (user restricted read permissions when not in system mode)
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
