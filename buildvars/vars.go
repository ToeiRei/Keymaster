// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package buildvars contains variables injected at build time.
package buildvars

import (
	"runtime/debug"
)

// Version is set at link time via `-ldflags -X github.com/toeirei/keymaster/buildvars.Version=...`.
var Version string

func init() {
	if Version == "" {
		Version = "unknown"

		if buildInfo, ok := debug.ReadBuildInfo(); ok {
			var vcs, vcsRevision, vcsModified string
			for _, setting := range buildInfo.Settings {
				switch setting.Key {
				case "vcs":
					vcs = setting.Value
				case "vcs.revision":
					vcsRevision = setting.Value
				case "vcs.modified":
					vcsModified = setting.Value
				}
			}

			if vcs != "" && vcsRevision != "" {
				if vcs == "git" {
					vcsRevision = vcsRevision[:7]
				}

				Version = vcs + ":" + vcsRevision

				if vcsModified == "true" {
					Version += "*"
				}
			}
		}
	}
}
