// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import "github.com/toeirei/keymaster/internal/model"

// Package-level defaults for DB-facing readers. Tests or initialization
// code can inject implementations via SetDefault* functions.
var (
	defaultKeyReader KeyReader
	defaultKeyLister KeyLister
)

// DefaultKeyReader returns the package-level KeyReader if set, else nil.
func DefaultKeyReader() KeyReader { return defaultKeyReader }

// SetDefaultKeyReader sets the package-level KeyReader used by core helpers.
func SetDefaultKeyReader(r KeyReader) { defaultKeyReader = r }

// DefaultKeyLister returns the package-level KeyLister if set, else nil.
func DefaultKeyLister() KeyLister { return defaultKeyLister }

// SetDefaultKeyLister sets the package-level KeyLister used by core helpers.
func SetDefaultKeyLister(l KeyLister) { defaultKeyLister = l }

// Deprecated helper kept for small convenience: convert model.SystemKey to a
// secret-like type. Implementations are intentionally left to callers.
func SystemKeyExists(sk *model.SystemKey) bool { return sk != nil }
