// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

// Package-level defaults for DB-facing readers. Tests or initialization
// code can inject implementations via SetDefault* functions.
var (
	defaultKeyReader            KeyReader
	defaultKeyLister            KeyLister
	defaultAccountSerialUpdater AccountSerialUpdater
	defaultKeyImporter          KeyImporter
	defaultAuditWriter          AuditWriter
	defaultAccountManager       AccountManager
	defaultDBInit               func(dbType, dsn string) error
	defaultDBIsInitialized      func() bool
)

// DefaultKeyReader returns the package-level KeyReader if set, else nil.
func DefaultKeyReader() KeyReader { return defaultKeyReader }

// SetDefaultKeyReader sets the package-level KeyReader used by core helpers.
func SetDefaultKeyReader(r KeyReader) { defaultKeyReader = r }

// DefaultKeyLister returns the package-level KeyLister if set, else nil.
func DefaultKeyLister() KeyLister { return defaultKeyLister }

// SetDefaultKeyLister sets the package-level KeyLister used by core helpers.
func SetDefaultKeyLister(l KeyLister) { defaultKeyLister = l }

// DefaultAccountSerialUpdater returns the package-level AccountSerialUpdater if set.
func DefaultAccountSerialUpdater() AccountSerialUpdater { return defaultAccountSerialUpdater }

// SetDefaultAccountSerialUpdater sets the package-level AccountSerialUpdater used by core helpers.
func SetDefaultAccountSerialUpdater(u AccountSerialUpdater) { defaultAccountSerialUpdater = u }

// DefaultKeyImporter returns the package-level KeyImporter if set, else nil.
func DefaultKeyImporter() KeyImporter { return defaultKeyImporter }

// SetDefaultKeyImporter sets the package-level KeyImporter used by core helpers.
func SetDefaultKeyImporter(k KeyImporter) { defaultKeyImporter = k }

// DefaultAuditWriter returns the package-level AuditWriter if set.
func DefaultAuditWriter() AuditWriter { return defaultAuditWriter }

// SetDefaultAuditWriter sets the package-level AuditWriter used by core helpers.
func SetDefaultAuditWriter(w AuditWriter) { defaultAuditWriter = w }

// DefaultAccountManager returns the package-level AccountManager if set.
func DefaultAccountManager() AccountManager { return defaultAccountManager }

// SetDefaultAccountManager sets the package-level AccountManager used by core helpers.
func SetDefaultAccountManager(a AccountManager) { defaultAccountManager = a }

// DefaultInitDB delegates DB initialization to the injected function if present.
func DefaultInitDB(dbType, dsn string) error {
	if defaultDBInit == nil {
		return fmt.Errorf("no DB init handler registered")
	}
	return defaultDBInit(dbType, dsn)
}

// SetDefaultDBInit registers a function to perform DB initialization.
func SetDefaultDBInit(fn func(dbType, dsn string) error) { defaultDBInit = fn }

// DefaultIsDBInitialized delegates to injected check if present.
func DefaultIsDBInitialized() bool {
	if defaultDBIsInitialized == nil {
		return false
	}
	return defaultDBIsInitialized()
}

// SetDefaultDBIsInitialized registers a function used to check DB init state.
func SetDefaultDBIsInitialized(fn func() bool) { defaultDBIsInitialized = fn }

// SystemKeyToSecret converts a stored `model.SystemKey` into a `security.Secret`.
// This mirrors the helper placed in `internal/db` but keeps conversion available
// to core callers without importing `internal/db` directly.
func SystemKeyToSecret(sk *model.SystemKey) security.Secret {
	if sk == nil {
		return nil
	}
	return security.FromString(sk.PrivateKey)
}
