// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

import "github.com/toeirei/keymaster/config"

// LogLevel represents the logging verbosity level.
type LogLevel int

const (
	Panic LogLevel = iota + 1
	Error
	Warn
	Info
	Debug
	Trace
)

func NewDefaultConfig() config.Config {
	return config.Config{
		Database: config.ConfigDatabase{
			Type: "sqlite",
			Dsn:  ":memory:",
		},
		Language: "de",
	}
}
