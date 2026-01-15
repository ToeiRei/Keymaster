// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
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
		fmt.Println("--- KEYMASTER DEBUG ---")
		// Config file used
		used := viper.ConfigFileUsed()
		fmt.Printf("Config file used: %s\n", used)

		// Viper settings
		settings := viper.AllSettings()
		b, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			log.Errorf("could not marshal viper settings: %v", err)
		} else {
			fmt.Println("-- viper.AllSettings() --")
			fmt.Println(string(b))
		}

		// Flags
		fmt.Println("-- flags --")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			val := f.Value.String()
			fmt.Printf("%s = %s\n", f.Name, val)
		})

		// Environment variables of interest
		fmt.Println("-- environment (KEYMASTER_*, KEYMASTER, CONFIG*) --")
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "KEYMASTER_") || strings.HasPrefix(e, "KEYMASTER") || strings.HasPrefix(e, "CONFIG") {
				fmt.Println(e)
			}
		}

		// Print GO env hints
		fmt.Printf("PWD=%s\n", os.Getenv("PWD"))
		fmt.Println("--- END DEBUG ---")
	},
}
