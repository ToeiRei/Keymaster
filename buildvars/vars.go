// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package buildvars contains variables injected at build time.
// Package buildvars contains variables injected at build time.
package buildvars

// Version is set at link time via `-ldflags -X github.com/toeirei/keymaster/buildvars.Version=...`.
// It will be empty for local or development builds.
var Version string

// VersionOrDefault returns `Version` if set, otherwise returns the provided default.
func VersionOrDefault(def string) string {
	if len(Version) > 0 {
		return Version
	}
	return def
}
