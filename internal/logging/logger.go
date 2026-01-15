// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package logging

import (
	"fmt"

	clog "github.com/charmbracelet/log"
)

// L is the package-level logger. Callers should use the helper functions
// below for compatibility with existing calls.
var L = clog.New()

// Debugf logs a debug-level formatted message.
func Debugf(format string, v ...interface{}) {
	L.Debug(fmt.Sprintf(format, v...))
}

// Infof logs an info-level formatted message.
func Infof(format string, v ...interface{}) {
	L.Info(fmt.Sprintf(format, v...))
}

// Warnf logs a warning-level formatted message.
func Warnf(format string, v ...interface{}) {
	L.Warn(fmt.Sprintf(format, v...))
}

// Errorf logs an error-level formatted message.
func Errorf(format string, v ...interface{}) {
	L.Error(fmt.Sprintf(format, v...))
}
