package db

import (
	"testing"

	"github.com/toeirei/keymaster/internal/model"
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

func (f *fakeKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool) error {
	return nil
}
func (f *fakeKeyManager) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool) (*model.PublicKey, error) {
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
	if _, err := km.AddPublicKeyAndGetModel("alg", "data", "c", false); err != nil {
		t.Fatalf("AddPublicKeyAndGetModel returned error: %v", err)
	}
	ClearDefaultKeyManager()
}
