package db

import (
	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

// AccountSearcher defines a minimal interface for searching accounts.
// Consumers can depend on this instead of concrete Store implementations.
type AccountSearcher interface {
	SearchAccounts(query string) ([]model.Account, error)
}

// BunAccountSearcher is a Bun-based implementation of AccountSearcher.
type BunAccountSearcher struct {
	bdb *bun.DB
}

// NewBunAccountSearcher creates a new BunAccountSearcher.
func NewBunAccountSearcher(bdb *bun.DB) AccountSearcher {
	return &BunAccountSearcher{bdb: bdb}
}

// NewAccountSearcherFromStore creates an AccountSearcher from any Store by
// using the underlying Bun DB. This is a convenience for POC/transition.
func NewAccountSearcherFromStore(s Store) AccountSearcher {
	return NewBunAccountSearcher(s.BunDB())
}

// SearchAccounts delegates to the centralized Bun search helper.
func (s *BunAccountSearcher) SearchAccounts(q string) ([]model.Account, error) {
	return SearchAccountsBun(s.bdb, q)
}

// DefaultAccountSearcher returns an AccountSearcher backed by the package-level
// `store` if available. It returns nil when the package store is not
// initialized; callers should handle nil by falling back to local filtering.
func DefaultAccountSearcher() AccountSearcher {
	// If a test or other code has injected a default searcher, prefer that.
	if defaultSearcher != nil {
		return defaultSearcher
	}
	if store == nil {
		return nil
	}
	return NewAccountSearcherFromStore(store)
}

// package-level override used primarily by tests to inject a fake searcher.
var defaultSearcher AccountSearcher

// SetDefaultAccountSearcher sets a package-level AccountSearcher that will be
// returned by DefaultAccountSearcher(). Useful for tests to inject a fake.
func SetDefaultAccountSearcher(s AccountSearcher) {
	defaultSearcher = s
}

// ClearDefaultAccountSearcher clears any previously set package-level searcher.
func ClearDefaultAccountSearcher() {
	defaultSearcher = nil
}
