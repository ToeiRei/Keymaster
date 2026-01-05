package ui

import (
	"github.com/toeirei/keymaster/internal/db"
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
