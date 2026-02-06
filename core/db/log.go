// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"fmt"

	log "github.com/charmbracelet/log"
)

var dbDebugEnabled bool

// SetDebug enables or disables DB debug logging. Disabled by default.
func SetDebug(enabled bool) {
	dbDebugEnabled = enabled
}

func dbLogf(format string, v ...any) {
	if dbDebugEnabled {
		log.Info(fmt.Sprintf("[DB] "+format, v...))
	}
}
