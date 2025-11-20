//go:build tools_probe
// +build tools_probe

package main

import "fmt"

func main() {
	// Simple placeholder probe file used only when explicitly built with the
	// `tools_probe` build tag. Keeps the tools package parseable.
	fmt.Println("bun_probe: noop")
}
