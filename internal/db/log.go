// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import "github.com/toeirei/keymaster/internal/logging"

// SetDebug enables or disables DB debug logging. Disabled by default.
func SetDebug(enabled bool) {
	logging.SetDebug(enabled)
}

func dbLogf(format string, v ...any) {
	logging.Debugf(format, v...)
}
