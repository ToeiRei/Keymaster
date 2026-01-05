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
