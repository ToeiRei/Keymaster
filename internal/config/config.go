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
	v := viper.New()

	// defaults
	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	// files (first file found wins)
	v.SetConfigName("keymaster")
	v.SetConfigType("yaml")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home + "/.config")
	}
	v.AddConfigPath("/etc/keymaster")

	// env
	v.AutomaticEnv()
	v.SetEnvPrefix("keymaster")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// cli
	// TODO: maybe needs to trigger additional parsing beferohand
	v.BindPFlags(cmd.Flags())

	// TODO: maybe not needed
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
