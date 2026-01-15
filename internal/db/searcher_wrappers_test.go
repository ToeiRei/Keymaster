// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/uptrace/bun"
)

type fakeAccountSearcher struct{}

func (f *fakeAccountSearcher) SearchAccounts(q string) ([]model.Account, error) {
	return []model.Account{{Username: "f"}}, nil
}

type fakeAuditSearcher struct{}

func (f *fakeAuditSearcher) GetAllAuditLogEntries() ([]model.AuditLogEntry, error) {
	return []model.AuditLogEntry{{ID: 1, Action: "a"}}, nil
}

type fakeKeySearcher struct{}

func (f *fakeKeySearcher) SearchPublicKeys(q string) ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 1, Comment: "c"}}, nil
}

type fakeAccountManager struct{}

func (f *fakeAccountManager) AddAccount(username, hostname, label, tags string) (int, error) {
	return 7, nil
}
func (f *fakeAccountManager) DeleteAccount(id int) error { return nil }

type fakeAuditWriter struct{}

func (f *fakeAuditWriter) LogAction(action string, details string) error { return nil }

// Minimal fake KeyManager implementing all methods with simple responses.
type fakeKeyManager struct{}

func (f *fakeKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	return nil
}
func (f *fakeKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	return &model.PublicKey{ID: 1, Comment: comment}, nil
}
func (f *fakeKeyManager) DeletePublicKey(id int) error       { return nil }
func (f *fakeKeyManager) TogglePublicKeyGlobal(id int) error { return nil }
func (f *fakeKeyManager) GetAllPublicKeys() ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 1}}, nil
}
func (f *fakeKeyManager) GetPublicKeyByComment(comment string) (*model.PublicKey, error) {
	return &model.PublicKey{ID: 1, Comment: comment}, nil
}
func (f *fakeKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 2}}, nil
}
func (f *fakeKeyManager) AssignKeyToAccount(keyID, accountID int) error     { return nil }
func (f *fakeKeyManager) UnassignKeyFromAccount(keyID, accountID int) error { return nil }
func (f *fakeKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return []model.PublicKey{{ID: 3}}, nil
}
func (f *fakeKeyManager) GetAccountsForKey(keyID int) ([]model.Account, error) {
	return []model.Account{{ID: 4}}, nil
}
func (f *fakeKeyManager) SetPublicKeyExpiry(id int, expiresAt time.Time) error { return nil }

func TestSearcherAndManagerWrappers_Injection(t *testing.T) {
	// AccountSearcher
	SetDefaultAccountSearcher(&fakeAccountSearcher{})
	as := DefaultAccountSearcher()
	if as == nil {
		t.Fatal("expected DefaultAccountSearcher to return injected searcher")
	}
	res, err := as.SearchAccounts("x")
	if err != nil || len(res) == 0 || res[0].Username != "f" {
		t.Fatalf("unexpected SearchAccounts result: %v %v", res, err)
	}
	ClearDefaultAccountSearcher()

	// AuditSearcher
	SetDefaultAuditSearcher(&fakeAuditSearcher{})
	aus := DefaultAuditSearcher()
	if aus == nil {
		t.Fatal("expected DefaultAuditSearcher to return injected searcher")
	}
	ares, err := aus.GetAllAuditLogEntries()
	if err != nil || len(ares) == 0 {
		t.Fatalf("unexpected GetAllAuditLogEntries result: %v %v", ares, err)
	}
	ClearDefaultAuditSearcher()

	// KeySearcher
	SetDefaultKeySearcher(&fakeKeySearcher{})
	ks := DefaultKeySearcher()
	if ks == nil {
		t.Fatal("expected DefaultKeySearcher to return injected searcher")
	}
	kres, err := ks.SearchPublicKeys("q")
	if err != nil || len(kres) == 0 {
		t.Fatalf("unexpected SearchPublicKeys result: %v %v", kres, err)
	}
	ClearDefaultKeySearcher()

	// AccountManager
	SetDefaultAccountManager(&fakeAccountManager{})
	am := DefaultAccountManager()
	if am == nil {
		t.Fatal("expected DefaultAccountManager to return injected manager")
	}
	id, err := am.AddAccount("u", "h", "l", "t")
	if err != nil || id != 7 {
		t.Fatalf("unexpected AddAccount result: %d %v", id, err)
	}
	ClearDefaultAccountManager()

	// AuditWriter
	SetDefaultAuditWriter(&fakeAuditWriter{})
	aw := DefaultAuditWriter()
	if aw == nil {
		t.Fatal("expected DefaultAuditWriter to return injected writer")
	}
	if err := aw.LogAction("a", "d"); err != nil {
		t.Fatalf("LogAction returned error: %v", err)
	}
	ClearDefaultAuditWriter()

	// KeyManager
	SetDefaultKeyManager(&fakeKeyManager{})
	km := DefaultKeyManager()
	if km == nil {
		t.Fatal("expected DefaultKeyManager to return injected manager")
	}
	if _, err := km.AddPublicKeyAndGetModel("alg", "data", "c", false, time.Time{}); err != nil {
		t.Fatalf("AddPublicKeyAndGetModel returned error: %v", err)
	}
	ClearDefaultKeyManager()
}

// fakeStore implements Store with minimal methods so we can set the package
// level `store` and exercise Default* wrapper paths that create Bun-backed
// searchers/managers. Methods return zero values and do not require a real DB.
type fakeStore struct{}

func (f *fakeStore) GetAllAccounts() ([]model.Account, error)                       { return nil, nil }
func (f *fakeStore) AddAccount(username, hostname, label, tags string) (int, error) { return 0, nil }
func (f *fakeStore) DeleteAccount(id int) error                                     { return nil }
func (f *fakeStore) UpdateAccountSerial(id, serial int) error                       { return nil }
func (f *fakeStore) ToggleAccountStatus(id int) error                               { return nil }
func (f *fakeStore) UpdateAccountLabel(id int, label string) error                  { return nil }
func (f *fakeStore) UpdateAccountHostname(id int, hostname string) error            { return nil }
func (f *fakeStore) UpdateAccountTags(id int, tags string) error                    { return nil }
func (f *fakeStore) UpdateAccountIsDirty(id int, dirty bool) error                  { return nil }
func (f *fakeStore) GetAllActiveAccounts() ([]model.Account, error)                 { return nil, nil }
func (f *fakeStore) GetKnownHostKey(hostname string) (string, error)                { return "", nil }
func (f *fakeStore) AddKnownHostKey(hostname, key string) error                     { return nil }
func (f *fakeStore) CreateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (f *fakeStore) RotateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (f *fakeStore) GetActiveSystemKey() (*model.SystemKey, error)                  { return nil, nil }
func (f *fakeStore) GetSystemKeyBySerial(serial int) (*model.SystemKey, error)      { return nil, nil }
func (f *fakeStore) HasSystemKeys() (bool, error)                                   { return false, nil }
func (f *fakeStore) SearchAccounts(query string) ([]model.Account, error)           { return nil, nil }
func (f *fakeStore) GetAllAuditLogEntries() ([]model.AuditLogEntry, error)          { return nil, nil }
func (f *fakeStore) LogAction(action string, details string) error                  { return nil }
func (f *fakeStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return nil
}
func (f *fakeStore) GetBootstrapSession(id string) (*model.BootstrapSession, error)  { return nil, nil }
func (f *fakeStore) DeleteBootstrapSession(id string) error                          { return nil }
func (f *fakeStore) UpdateBootstrapSessionStatus(id string, status string) error     { return nil }
func (f *fakeStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) { return nil, nil }
func (f *fakeStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}
func (f *fakeStore) ExportDataForBackup() (*model.BackupData, error) { return nil, nil }
func (f *fakeStore) ImportDataFromBackup(*model.BackupData) error    { return nil }
func (f *fakeStore) IntegrateDataFromBackup(*model.BackupData) error { return nil }
func (f *fakeStore) BunDB() *bun.DB                                  { return nil }

func TestDefaultWrappers_WithStore(t *testing.T) {
	// Preserve original store and restore at the end.
	orig := store
	defer func() { store = orig }()

	store = &fakeStore{}

	if DefaultAccountSearcher() == nil {
		t.Fatal("expected DefaultAccountSearcher to return non-nil when store set")
	}
	if DefaultAuditSearcher() == nil {
		t.Fatal("expected DefaultAuditSearcher to return non-nil when store set")
	}
	if DefaultKeySearcher() == nil {
		t.Fatal("expected DefaultKeySearcher to return non-nil when store set")
	}
	if DefaultAccountManager() == nil {
		t.Fatal("expected DefaultAccountManager to return non-nil when store set")
	}
	if DefaultKeyManager() == nil {
		t.Fatal("expected DefaultKeyManager to return non-nil when store set")
	}
	if DefaultAuditWriter() == nil {
		t.Fatal("expected DefaultAuditWriter to return non-nil when store set")
	}
}

