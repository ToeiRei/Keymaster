// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/toeirei/keymaster/core/model"
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

// FindByIdentifier locates an account by an identifier string. The identifier
// can be either an integer ID (as string) or a `user@host` form. This helper
// mirrors the lookup behavior needed by higher layers and is provided here so
// adapters can satisfy the core AccountManager contract without importing core.
func (b *bunAccountManager) FindByIdentifier(ctx context.Context, identifier string) (*model.Account, error) {
	if identifier == "" {
		return nil, fmt.Errorf("empty identifier")
	}
	// Try numeric ID first
	// Avoid importing strconv at package top if not needed elsewhere
	var id int
	if n, err := fmt.Sscanf(identifier, "%d", &id); err == nil && n == 1 {
		if acc, err := GetAccountByIDBun(b.bStore.BunDB(), id); err == nil && acc != nil {
			return acc, nil
		}
	}

	// Try user@host form
	for _, a := range mustGetAllAccounts(b.bStore) {
		if a.Username+"@"+a.Hostname == identifier {
			aa := a
			return &aa, nil
		}
	}
	return nil, fmt.Errorf("account not found: %s", identifier)
}

// SetActive sets the account's active flag to the provided state. It will
// toggle the stored status only when a change is required.
func (b *bunAccountManager) SetActive(ctx context.Context, accountID int, active bool) error {
	acc, err := GetAccountByIDBun(b.bStore.BunDB(), accountID)
	if err != nil {
		return err
	}
	if acc == nil {
		return fmt.Errorf("account not found: %d", accountID)
	}
	if acc.IsActive == active {
		return nil
	}
	// SetAccountActive will set the account active flag to the desired state.
	return SetAccountActive(accountID, active)
}

// GetAll returns all accounts from the underlying store.
func (b *bunAccountManager) GetAll(ctx context.Context) ([]model.Account, error) {
	return b.bStore.GetAllAccounts()
}

// mustGetAllAccounts is a small helper that attempts to fetch all accounts
// and returns an empty slice on error to simplify callers where absence is
// handled upstream.
func mustGetAllAccounts(s Store) []model.Account {
	if s == nil {
		return nil
	}
	accts, err := s.GetAllAccounts()
	if err != nil {
		return nil
	}
	return accts
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
	AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error
	AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error)
	DeletePublicKey(id int) error
	TogglePublicKeyGlobal(id int) error
	// SetPublicKeyExpiry sets or clears the expires_at for a public key. A zero
	// time value will clear the expiration (set NULL).
	SetPublicKeyExpiry(id int, expiresAt time.Time) error
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

func (b *bunKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	err := AddPublicKeyBun(b.bStore.BunDB(), algorithm, keyData, comment, isGlobal, expiresAt)
	if err == nil {
		_ = b.bStore.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return err
}

func (b *bunKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	pk, err := AddPublicKeyAndGetModelBun(b.bStore.BunDB(), algorithm, keyData, comment, isGlobal, expiresAt)
	if err == nil && pk != nil {
		_ = b.bStore.LogAction("ADD_PUBLIC_KEY", fmt.Sprintf("comment: %s", comment))
	}
	return pk, err
}

func (b *bunKeyManager) DeletePublicKey(id int) error {
	details := fmt.Sprintf("id: %d", id)
	if pk, _ := GetPublicKeyByIDBun(b.bStore.BunDB(), id); pk != nil {
		details = fmt.Sprintf("comment: %s", pk.Comment)
	}
	err := DeletePublicKeyBun(b.bStore.BunDB(), id)
	if err == nil {
		_ = b.bStore.LogAction("DELETE_PUBLIC_KEY", details)
	}
	return err
}

func (b *bunKeyManager) TogglePublicKeyGlobal(id int) error {
	err := TogglePublicKeyGlobalBun(b.bStore.BunDB(), id)
	if err == nil {
		_ = b.bStore.LogAction("TOGGLE_KEY_GLOBAL", fmt.Sprintf("key_id: %d", id))
	}
	return err
}

func (b *bunKeyManager) SetPublicKeyExpiry(id int, expiresAt time.Time) error {
	err := SetPublicKeyExpiryBun(b.bStore.BunDB(), id, expiresAt)
	if err == nil {
		_ = b.bStore.LogAction("SET_KEY_EXPIRES", fmt.Sprintf("key_id: %d expires_at: %v", id, expiresAt))
	}
	return err
}

func (b *bunKeyManager) GetAllPublicKeys() ([]model.PublicKey, error) {
	return GetAllPublicKeysBun(b.bStore.BunDB())
}

func (b *bunKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return GetPublicKeyByCommentBun(b.bStore.BunDB(), comment)
}

func (b *bunKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return GetGlobalPublicKeysBun(b.bStore.BunDB())
}
func (b *bunKeyManager) AssignKeyToAccount(keyID, accountID int) error {
	err := AssignKeyToAccountBun(b.bStore.BunDB(), keyID, accountID)
	if err == nil {
		var keyComment, accUser, accHost string
		if pk, _ := GetPublicKeyByIDBun(b.bStore.BunDB(), keyID); pk != nil {
			keyComment = pk.Comment
		}
		if acc, _ := GetAccountByIDBun(b.bStore.BunDB(), accountID); acc != nil {
			accUser = acc.Username
			accHost = acc.Hostname
		}
		details := fmt.Sprintf("key: '%s' to account: %s@%s", keyComment, accUser, accHost)
		_ = b.bStore.LogAction("ASSIGN_KEY", details)
	}
	return err
}

func (b *bunKeyManager) UnassignKeyFromAccount(keyID, accountID int) error {
	var keyComment, accUser, accHost string
	if pk, _ := GetPublicKeyByIDBun(b.bStore.BunDB(), keyID); pk != nil {
		keyComment = pk.Comment
	}
	if acc, _ := GetAccountByIDBun(b.bStore.BunDB(), accountID); acc != nil {
		accUser = acc.Username
		accHost = acc.Hostname
	}
	details := fmt.Sprintf("key: '%s' from account: %s@%s", keyComment, accUser, accHost)
	err := UnassignKeyFromAccountBun(b.bStore.BunDB(), keyID, accountID)
	if err == nil {
		_ = b.bStore.LogAction("UNASSIGN_KEY", details)
	}
	return err
}

func (b *bunKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return GetKeysForAccountBun(b.bStore.BunDB(), accountID)
}

func (b *bunKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return GetAccountsForKeyBun(b.bStore.BunDB(), keyID)
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
