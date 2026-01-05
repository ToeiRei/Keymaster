// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// Package db contains shared database errors and helpers.
package db

import (
	"errors"
	"strings"
)

// ErrDuplicate is returned when attempting to insert a record that already exists.
var ErrDuplicate = errors.New("duplicate record")

// MapDBError inspects low-level driver errors and maps common constraint
// violations to package-level sentinel errors (like ErrDuplicate). This is a
// conservative, string-based mapping to avoid importing SQL driver packages
// into this package file.
func MapDBError(err error) error {
	if err == nil {
		return nil
	}
	le := strings.ToLower(err.Error())
	// MySQL duplicate entry, Postgres unique violation (23505), SQLite unique constraint
	if strings.Contains(le, "duplicate") || strings.Contains(le, "unique") || strings.Contains(le, "23505") || strings.Contains(le, "1062") {
		return ErrDuplicate
	}
	return err
}
