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

// KeySearcher defines a minimal interface for searching public keys.
type KeySearcher interface {
	SearchPublicKeys(query string) ([]model.PublicKey, error)
}

// BunKeySearcher is a Bun-based implementation of KeySearcher.
type BunKeySearcher struct {
	bdb *bun.DB
}

// NewBunKeySearcher creates a new BunKeySearcher.
func NewBunKeySearcher(bdb *bun.DB) KeySearcher {
	return &BunKeySearcher{bdb: bdb}
}

// NewKeySearcherFromStore creates a KeySearcher from any Store by using
// the underlying Bun DB. This is a convenience for POC/transition.
func NewKeySearcherFromStore(s Store) KeySearcher {
	return NewBunKeySearcher(s.BunDB())
}

// SearchPublicKeys delegates to the centralized Bun search helper.
func (s *BunKeySearcher) SearchPublicKeys(q string) ([]model.PublicKey, error) {
	return SearchPublicKeysBun(s.bdb, q)
}

// DefaultKeySearcher returns a KeySearcher backed by the package-level
// `store` if available. It returns nil when the package store is not
// initialized; callers should handle nil by falling back to local filtering.
func DefaultKeySearcher() KeySearcher {
	if defaultKeySearcher != nil {
		return defaultKeySearcher
	}
	if store == nil {
		return nil
	}
	return NewKeySearcherFromStore(store)
}

// package-level override used primarily by tests to inject a fake key searcher.
var defaultKeySearcher KeySearcher

// SetDefaultKeySearcher sets a package-level KeySearcher that will be
// returned by DefaultKeySearcher(). Useful for tests to inject a fake.
func SetDefaultKeySearcher(s KeySearcher) {
	defaultKeySearcher = s
}

// ClearDefaultKeySearcher clears any previously set package-level key searcher.
func ClearDefaultKeySearcher() {
	defaultKeySearcher = nil
}
