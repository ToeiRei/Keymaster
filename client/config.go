// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

// TODO move into seperate package
type LogLevel int

const (
	Panic LogLevel = iota + 1
	Error
	Warn
	Info
	Debug
	Trace
)

type Config struct {
	// add all options needed to initialize the client and its db connection here
	LogLevel     LogLevel
	DatabaseType string
	DatabaseUri  string
	// ...
}

func NewDefaultConfig() Config {
	return Config{
		LogLevel:     Info,
		DatabaseType: "sqlite",
		DatabaseUri:  ":memory:",
	}
}
