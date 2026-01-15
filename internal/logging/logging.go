// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package logging

import "log"

var debugEnabled bool

// SetDebug enables or disables debug logging for the application.
// SetDebug controls whether calls to Debugf will emit output.
func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// Debugf logs a formatted debug message when debug is enabled.
// Debugf is a no-op when debug is disabled.
func Debugf(format string, v ...any) {
	if debugEnabled {
		log.Printf(format, v...)
	}
}

// Infof logs an informational formatted message.
func Infof(format string, v ...any) {
	log.Printf(format, v...)
}

// Errorf logs an error formatted message.
func Errorf(format string, v ...any) {
	log.Printf(format, v...)
}

// Printf is a convenience alias for Infof.
func Printf(format string, v ...any) {
	Infof(format, v...)
}
