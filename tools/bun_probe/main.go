//go:build tools_probe
// +build tools_probe

// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import "fmt"

func main() {
	// Simple placeholder probe file used only when explicitly built with the
	// `tools_probe` build tag. Keeps the tools package parseable.
	fmt.Println("bun_probe: noop")
}
