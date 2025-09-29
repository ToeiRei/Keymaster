package config

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Database struct {
		Type string `mapstructure:"type"`
		Dsn  string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Language string `mapstructure:"language"`
}

var Defauls = map[string]any{
	"database.type": "sqlite",
	"database.dsn":  "./keymaster.db",
	"language":      "en",
}

func LoadConfig[T any](cmd *cobra.Command, defaults map[string]any) (T, error) {
	var c T

	// defaults
	for key, value := range defaults {
		viper.SetDefault(key, value)
	}

	// files (first file found wins)
	viper.SetConfigName("keymaster")
	viper.SetConfigType("yaml")
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(home + "/.config")
	}
	viper.AddConfigPath("/etc/keymaster")

	// env
	viper.AutomaticEnv()
	viper.SetEnvPrefix("keymaster")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// cli
	// TODO: maybe needs to trigger additional parsing beferohand
	viper.BindPFlags(cmd.Flags())

	// TODO: maybe not needed
	// if err := viper.ReadInConfig(); err != nil {
	// 	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
	// 		return nil, err
	// 	}
	// }

	// parse config
	if err := viper.Unmarshal(&c); err != nil {
		return c, err
	}

	return c, nil
}
