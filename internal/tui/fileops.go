// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package tui

import (
	"os"
	"runtime"
)

// WriteKeyFile writes `content` to `filename` using a secure default file mode.
// On Unix-like systems this uses 0600. On Windows, where POSIX permissions are
// not meaningful, it falls back to 0644 for compatibility.
func WriteKeyFile(filename string, content []byte) error {
	perm := os.FileMode(0600)
	if runtime.GOOS == "windows" {
		perm = 0644
	}
	return os.WriteFile(filename, content, perm)
}
