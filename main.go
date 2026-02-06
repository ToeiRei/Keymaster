// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Command-line entrypoint for Keymaster.
//
// Usage:
//
//	go run . [flags]
//	./keymaster [flags]
//
// This launches the Keymaster CLI. See --help for options.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/toeirei/keymaster/ui/cli"
)

// version is set at build time using -ldflags, e.g.:
// go build -ldflags "-X main.version=1.2.3"
var version = "dev"

// main is the entrypoint for the Keymaster CLI.
func main() {
	// Print version info if requested (optional, placeholder for future flag parsing)
	if os.Getenv("KEYMASTER_SHOW_VERSION") == "1" {
		fmt.Fprintf(os.Stderr, "Keymaster version: %s\n", version)
	}

	if err := cli.Execute(); err != nil {
		log.Printf("Keymaster CLI error: %v", err)
		os.Exit(1)
	}
}
