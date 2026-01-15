// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"encoding/json"
	"os"
	"strings"

	log "github.com/charmbracelet/log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Dump debug information about config, env, flags and settings",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("--- KEYMASTER DEBUG ---")
		// Config file used
		used := viper.ConfigFileUsed()
		log.Infof("Config file used: %s", used)

		// Viper settings
		settings := viper.AllSettings()
		b, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			log.Errorf("could not marshal viper settings: %v", err)
		} else {
			log.Info("-- viper.AllSettings() --")
			log.Info(string(b))
		}

		// Flags
		log.Info("-- flags --")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			val := f.Value.String()
			log.Infof("%s = %s", f.Name, val)
		})

		// Environment variables of interest
		log.Info("-- environment (KEYMASTER_*, KEYMASTER, CONFIG*) --")
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "KEYMASTER_") || strings.HasPrefix(e, "KEYMASTER") || strings.HasPrefix(e, "CONFIG") {
				log.Info(e)
			}
		}

		// Print GO env hints
		log.Infof("PWD=%s", os.Getenv("PWD"))
		log.Info("--- END DEBUG ---")
	},
}

