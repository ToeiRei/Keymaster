// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

type LogLevel int

type Config struct {
	// add all options needed to initialize the client and its db connection here
	LogLevel    LogLevel
	DatabaseUri string
	// ...
}
