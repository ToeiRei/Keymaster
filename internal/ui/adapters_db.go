// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ui

import (
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/model"
)

type dbSearcherAdapter struct {
	s db.AccountSearcher
}

func (d *dbSearcherAdapter) SearchAccounts(q string) ([]model.Account, error) {
	if d.s == nil {
		return nil, nil
	}
	return d.s.SearchAccounts(q)
}

// DefaultAccountSearcher returns an AccountSearcher that wraps the package
// default DB searcher. It may return nil if no DB searcher is configured.
func DefaultAccountSearcher() AccountSearcher {
	s := db.DefaultAccountSearcher()
	if s == nil {
		return nil
	}
	return &dbSearcherAdapter{s: s}
}

type dbKeySearcherAdapter struct {
	s db.KeySearcher
}

func (d *dbKeySearcherAdapter) SearchPublicKeys(q string) ([]model.PublicKey, error) {
	if d.s == nil {
		return nil, nil
	}
	return d.s.SearchPublicKeys(q)
}

// DefaultKeySearcher returns a KeySearcher that wraps the package default DB key searcher.
// It may return nil if no DB key searcher is configured.
func DefaultKeySearcher() KeySearcher {
	s := db.DefaultKeySearcher()
	if s == nil {
		return nil
	}
	return &dbKeySearcherAdapter{s: s}
}

type dbAuditSearcherAdapter struct {
	s db.AuditSearcher
}

func (d *dbAuditSearcherAdapter) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	if d.s == nil {
		return nil, nil
	}
	return d.s.GetAllAuditLogEntries()
}

// DefaultAuditSearcher returns an AuditSearcher that wraps the package default
// DB audit searcher. It may return nil if no DB audit searcher is configured.
func DefaultAuditSearcher() AuditSearcher {
	s := db.DefaultAuditSearcher()
	if s == nil {
		return nil
	}
	return &dbAuditSearcherAdapter{s: s}
}
