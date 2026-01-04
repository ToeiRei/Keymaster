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

// AuditSearcher defines a minimal interface for retrieving audit log entries.
type AuditSearcher interface {
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
}

// BunAuditSearcher is a Bun-based implementation of AuditSearcher.
type BunAuditSearcher struct {
	bdb *bun.DB
}

// NewBunAuditSearcher creates a new BunAuditSearcher.
func NewBunAuditSearcher(bdb *bun.DB) AuditSearcher {
	return &BunAuditSearcher{bdb: bdb}
}

// NewAuditSearcherFromStore creates an AuditSearcher from any Store by using
// the underlying Bun DB.
func NewAuditSearcherFromStore(s Store) AuditSearcher {
	return NewBunAuditSearcher(s.BunDB())
}

// GetAllAuditLogEntries delegates to the centralized Bun helper.
func (s *BunAuditSearcher) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return GetAllAuditLogEntriesBun(s.bdb)
}

// DefaultAuditSearcher returns an AuditSearcher backed by the package-level
// `store` if available. It returns nil when the package store is not
// initialized; callers should handle nil by falling back to direct helpers.
func DefaultAuditSearcher() AuditSearcher {
	if defaultAuditSearcher != nil {
		return defaultAuditSearcher
	}
	if store == nil {
		return nil
	}
	return NewAuditSearcherFromStore(store)
}

// package-level override used primarily by tests to inject a fake audit searcher.
var defaultAuditSearcher AuditSearcher

// SetDefaultAuditSearcher sets a package-level AuditSearcher that will be
// returned by DefaultAuditSearcher(). Useful for tests to inject a fake.
func SetDefaultAuditSearcher(s AuditSearcher) {
	defaultAuditSearcher = s
}

// ClearDefaultAuditSearcher clears any previously set package-level audit searcher.
func ClearDefaultAuditSearcher() {
	defaultAuditSearcher = nil
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

// AccountManager defines a minimal interface for managing accounts (add/delete).
// This allows higher-level components to avoid depending directly on the Store
// implementation and enables tests to inject fakes.
type AccountManager interface {
	AddAccount(username, hostname, label, tags string) (int, error)
	DeleteAccount(id int) error
}

// DefaultAccountManager returns an AccountManager backed by the package-level
// `store` if available. Tests may inject a fake via SetDefaultAccountManager.
func DefaultAccountManager() AccountManager {
	if defaultAccountManager != nil {
		return defaultAccountManager
	}
	if store == nil {
		return nil
	}
	// Use the package store as the default AccountManager by delegating to it.
	return &bunAccountManager{bStore: store}
}

// bunAccountManager adapts the existing Store to the AccountManager interface.
type bunAccountManager struct {
	bStore Store
}

func (b *bunAccountManager) AddAccount(username, hostname, label, tags string) (int, error) {
	return b.bStore.AddAccount(username, hostname, label, tags)
}

func (b *bunAccountManager) DeleteAccount(id int) error {
	return b.bStore.DeleteAccount(id)
}

// package-level override used primarily by tests to inject a fake account manager.
var defaultAccountManager AccountManager

// SetDefaultAccountManager sets a package-level AccountManager that will be
// returned by DefaultAccountManager(). Useful for tests to inject a fake.
func SetDefaultAccountManager(m AccountManager) {
	defaultAccountManager = m
}

// ClearDefaultAccountManager clears any previously set package-level account manager.
func ClearDefaultAccountManager() {
	defaultAccountManager = nil
}

// KeyManager defines a minimal interface for managing public keys (add/delete,
// toggle global, assignments, and simple retrievals). This mirrors the Store
// methods but keeps higher-level code decoupled from the concrete Store.
type KeyManager interface {
	AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error
	AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error)
	DeletePublicKey(id int) error
	TogglePublicKeyGlobal(id int) error
	GetAllPublicKeys() ([]model.PublicKey, error)
	GetPublicKeyByComment(comment string) (*model.PublicKey, error)
	GetGlobalPublicKeys() ([]model.PublicKey, error)
	AssignKeyToAccount(keyID, accountID int) error
	UnassignKeyFromAccount(keyID, accountID int) error
	GetKeysForAccount(accountID int) ([]model.PublicKey, error)
	GetAccountsForKey(keyID int) ([]model.Account, error)
}

// DefaultKeyManager returns a KeyManager backed by the package-level `store`.
// Tests can inject a fake via SetDefaultKeyManager.
func DefaultKeyManager() KeyManager {
	if defaultKeyManager != nil {
		return defaultKeyManager
	}
	if store == nil {
		return nil
	}
	return &bunKeyManager{bStore: store}
}

// bunKeyManager adapts the Store to KeyManager.
type bunKeyManager struct{ bStore Store }

func (b *bunKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	return b.bStore.AddPublicKey(algorithm, keyData, comment, isGlobal)
}

func (b *bunKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
	return b.bStore.AddPublicKeyAndGetModel(algorithm, keyData, comment, isGlobal)
}

func (b *bunKeyManager) DeletePublicKey(id int) error { return b.bStore.DeletePublicKey(id) }
func (b *bunKeyManager) TogglePublicKeyGlobal(id int) error {
	return b.bStore.TogglePublicKeyGlobal(id)
}
func (b *bunKeyManager) GetAllPublicKeys() ([]model.PublicKey, error) {
	return b.bStore.GetAllPublicKeys()
}
func (b *bunKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return b.bStore.GetPublicKeyByComment(comment)
}
func (b *bunKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return b.bStore.GetGlobalPublicKeys()
}
func (b *bunKeyManager) AssignKeyToAccount(keyID, accountID int) error {
	return b.bStore.AssignKeyToAccount(keyID, accountID)
}
func (b *bunKeyManager) UnassignKeyFromAccount(keyID, accountID int) error {
	return b.bStore.UnassignKeyFromAccount(keyID, accountID)
}
func (b *bunKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return b.bStore.GetKeysForAccount(accountID)
}
func (b *bunKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return b.bStore.GetAccountsForKey(keyID)
}

// package-level override used by tests
var defaultKeyManager KeyManager

// SetDefaultKeyManager sets a package-level KeyManager for DefaultKeyManager().
func SetDefaultKeyManager(m KeyManager) { defaultKeyManager = m }

// ClearDefaultKeyManager clears any previously set package-level key manager.
func ClearDefaultKeyManager() { defaultKeyManager = nil }

// AuditWriter defines a minimal interface for recording audit log events.
type AuditWriter interface {
	LogAction(action string, details string) error
}

// BunAuditWriter is a Bun-based implementation of AuditWriter.
type BunAuditWriter struct {
	bdb *bun.DB
}

// NewBunAuditWriter creates a new BunAuditWriter.
func NewBunAuditWriter(bdb *bun.DB) AuditWriter {
	return &BunAuditWriter{bdb: bdb}
}

// NewAuditWriterFromStore creates an AuditWriter from any Store by using
// the underlying Bun DB.
func NewAuditWriterFromStore(s Store) AuditWriter {
	return NewBunAuditWriter(s.BunDB())
}

// LogAction delegates to the centralized Bun helper.
func (s *BunAuditWriter) LogAction(action string, details string) error {
	return LogActionBun(s.bdb, action, details)
}

// DefaultAuditWriter returns an AuditWriter backed by the package-level
// `store` if available. It returns nil when the package store is not
// initialized; callers should handle nil by falling back to direct helpers.
func DefaultAuditWriter() AuditWriter {
	if defaultAuditWriter != nil {
		return defaultAuditWriter
	}
	if store == nil {
		return nil
	}
	return NewAuditWriterFromStore(store)
}

// package-level override used primarily by tests to inject a fake audit writer.
var defaultAuditWriter AuditWriter

// SetDefaultAuditWriter sets a package-level AuditWriter that will be
// returned by DefaultAuditWriter(). Useful for tests to inject a fake.
func SetDefaultAuditWriter(w AuditWriter) {
	defaultAuditWriter = w
}

// ClearDefaultAuditWriter clears any previously set package-level audit writer.
func ClearDefaultAuditWriter() {
	defaultAuditWriter = nil
}
