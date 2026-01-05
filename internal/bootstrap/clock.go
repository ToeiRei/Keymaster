// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import "time"

// Clock provides an abstraction over time.Now for testability.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

var defaultClock Clock = systemClock{}

// SetClock replaces the global clock used by the package. Tests may set a fake clock.
func SetClock(c Clock) { defaultClock = c }

// ResetClock restores the default system clock.
func ResetClock() { defaultClock = systemClock{} }
