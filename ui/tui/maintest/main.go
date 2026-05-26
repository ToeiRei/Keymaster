// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	tui "github.com/toeirei/keymaster/ui/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
