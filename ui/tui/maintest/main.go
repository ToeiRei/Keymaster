// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	tui "github.com/toeirei/keymaster/ui/tui"
	"github.com/toeirei/keymaster/uiadapters"
)

func main() {
	// Use the store adapter for the test TUI
	store := uiadapters.NewStoreAdapter()
	if err := tui.Run(store); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
