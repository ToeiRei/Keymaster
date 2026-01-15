// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package logging

import log "github.com/charmbracelet/log"

// SetDebug enables or disables debug logging for the application by adjusting
// the global charmbracelet logger level.
func SetDebug(enabled bool) {
	if enabled {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

// Debugf logs a formatted debug message.
func Debugf(format string, v ...any) {
	log.Debugf(format, v...)
}

// Infof logs an informational formatted message.
func Infof(format string, v ...any) {
	log.Infof(format, v...)
}

// Errorf logs an error formatted message.
func Errorf(format string, v ...any) {
	log.Errorf(format, v...)
}

// Printf is a convenience alias for Infof.
func Printf(format string, v ...any) {
	Infof(format, v...)
}
