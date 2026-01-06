// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import "log"

var debugEnabled bool

// SetDebug enables or disables DB debug logging. Disabled by default.
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

func dbLogf(format string, v ...any) {
	if debugEnabled {
		log.Printf(format, v...)
	}
}
