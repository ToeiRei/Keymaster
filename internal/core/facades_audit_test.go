package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

type simpleFakeStore struct {
	accounts []model.Account
	updates  map[int]bool
}

func (s *simpleFakeStore) GetAccounts() ([]model.Account, error)          { return nil, nil }
func (s *simpleFakeStore) GetAllActiveAccounts() ([]model.Account, error) { return s.accounts, nil }
func (s *simpleFakeStore) GetAllAccounts() ([]model.Account, error)       { return nil, nil }
func (s *simpleFakeStore) GetAccount(id int) (*model.Account, error)      { return nil, nil }
func (s *simpleFakeStore) AddAccount(username, hostname, label, tags string) (int, error) {
	return 0, nil
}
func (s *simpleFakeStore) DeleteAccount(accountID int) error             { return nil }
func (s *simpleFakeStore) AssignKeyToAccount(keyID, accountID int) error { return nil }
func (s *simpleFakeStore) UpdateAccountIsDirty(id int, dirty bool) error {
	if s.updates == nil {
		s.updates = make(map[int]bool)
	}
	s.updates[id] = dirty
	return nil
}
func (s *simpleFakeStore) CreateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (s *simpleFakeStore) RotateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (s *simpleFakeStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "sys-pub", PrivateKey: "sys-priv", IsActive: true}, nil
}
func (s *simpleFakeStore) AddKnownHostKey(hostname, key string) error      { return nil }
func (s *simpleFakeStore) ExportDataForBackup() (*model.BackupData, error) { return nil, nil }
func (s *simpleFakeStore) ImportDataFromBackup(*model.BackupData) error    { return nil }
func (s *simpleFakeStore) IntegrateDataFromBackup(*model.BackupData) error { return nil }

type fakeDeployerManager struct {
	content []byte
	ferr    error
}

func (f *fakeDeployerManager) DeployForAccount(account model.Account, keepFile bool) error {
	return nil
}
func (f *fakeDeployerManager) AuditSerial(account model.Account) error { return nil }
func (f *fakeDeployerManager) AuditStrict(account model.Account) error { return nil }
func (f *fakeDeployerManager) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (f *fakeDeployerManager) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (f *fakeDeployerManager) CanonicalizeHostPort(host string) string { return host }
func (f *fakeDeployerManager) ParseHostPort(host string) (string, string, error) {
	return host, "22", nil
}
func (f *fakeDeployerManager) GetRemoteHostKey(host string) (string, error) { return "hostkey", nil }
func (f *fakeDeployerManager) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return f.content, f.ferr
}
func (f *fakeDeployerManager) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeDeployerManager) IsPassphraseRequired(err error) bool { return false }

type spyAuditWriter struct {
	actions []string
}

func (s *spyAuditWriter) LogAction(action, details string) error {
	s.actions = append(s.actions, action+":"+details)
	return nil
}

// Minimal KeyReader/KeyLister fakes used by GenerateKeysContent
type fakeKR struct{}

func (f *fakeKR) GetAllPublicKeys() ([]model.PublicKey, error) { return nil, nil }
func (f *fakeKR) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "sys-pub", PrivateKey: "sys-priv", IsActive: true}, nil
}
func (f *fakeKR) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return &model.SystemKey{Serial: serial, PublicKey: "sys-pub", PrivateKey: "sys-priv", IsActive: true}, nil
}

type fakeKL struct{}

func (f *fakeKL) GetGlobalPublicKeys() ([]model.PublicKey, error)            { return nil, nil }
func (f *fakeKL) GetKeysForAccount(accountID int) ([]model.PublicKey, error) { return nil, nil }
func (f *fakeKL) GetAllPublicKeys() ([]model.PublicKey, error)               { return nil, nil }

// simple DeployerManager used by DeployDirtyAccounts test
type simpleDM struct{}

func (s *simpleDM) DeployForAccount(account model.Account, keepFile bool) error {
	if account.ID == 11 {
		return errors.New("deploy fail")
	}
	return nil
}
func (s *simpleDM) AuditSerial(account model.Account) error { return nil }
func (s *simpleDM) AuditStrict(account model.Account) error { return nil }
func (s *simpleDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (s *simpleDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (s *simpleDM) CanonicalizeHostPort(host string) string                   { return host }
func (s *simpleDM) ParseHostPort(host string) (string, string, error)         { return host, "22", nil }
func (s *simpleDM) GetRemoteHostKey(host string) (string, error)              { return "hostkey", nil }
func (s *simpleDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return nil, nil }
func (s *simpleDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (s *simpleDM) IsPassphraseRequired(err error) bool { return false }

func TestAuditAccounts_StrictMatch_NoDirty(t *testing.T) {
	i18n.Init("en")
	// prepare account
	acct := model.Account{ID: 1, Username: "u", Hostname: "h", Serial: 1, IsActive: true}

	store := &simpleFakeStore{accounts: []model.Account{acct}}
	dm := &fakeDeployerManager{}
	// provide expected content that matches generated content
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})
	expected, err := GenerateKeysContent(acct.ID)
	if err != nil {
		t.Fatalf("GenerateKeysContent failed: %v", err)
	}
	dm.content = []byte(expected)

	aw := &spyAuditWriter{}
	SetDefaultAuditWriter(aw)

	res, err := AuditAccounts(context.TODO(), store, dm, "strict", nil)
	if err != nil {
		t.Fatalf("AuditAccounts returned err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	if res[0].Error != nil {
		t.Fatalf("expected no audit error, got %v", res[0].Error)
	}
	if len(aw.actions) != 0 {
		t.Fatalf("expected no audit actions logged, got %v", aw.actions)
	}
	if len(store.updates) != 0 {
		t.Fatalf("expected no UpdateAccountIsDirty calls, got %v", store.updates)
	}
}

func TestAuditAccounts_StrictMismatch_LogsAndMarksDirty(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 2, Username: "u2", Hostname: "h2", Serial: 1, IsActive: true}
	store := &simpleFakeStore{accounts: []model.Account{acct}}
	dm := &fakeDeployerManager{content: []byte("mismatched content")}

	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})
	aw := &spyAuditWriter{}
	SetDefaultAuditWriter(aw)

	res, err := AuditAccounts(context.TODO(), store, dm, "strict", nil)
	if err != nil {
		t.Fatalf("AuditAccounts returned err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	if res[0].Error == nil {
		t.Fatalf("expected audit error for mismatch, got nil")
	}
	found := false
	for _, a := range aw.actions {
		if strings.HasPrefix(a, "AUDIT_HASH_MISMATCH") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected AUDIT_HASH_MISMATCH logged, got %v", aw.actions)
	}
	if v, ok := store.updates[acct.ID]; !ok || !v {
		t.Fatalf("expected UpdateAccountIsDirty called with true, got %v", store.updates)
	}
}

func TestDeployDirtyAccounts_ClearsDirtyOnSuccess(t *testing.T) {
	acct1 := model.Account{ID: 10, Username: "a", Hostname: "h", IsActive: true, IsDirty: true}
	acct2 := model.Account{ID: 11, Username: "b", Hostname: "h2", IsActive: true, IsDirty: true}
	store := &simpleFakeStore{accounts: []model.Account{acct1, acct2}}

	simple := &simpleDM{}
	results, err := DeployDirtyAccounts(context.TODO(), store, simple, nil)
	if err != nil {
		t.Fatalf("DeployDirtyAccounts returned err: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// acct1 succeeded -> should clear dirty (UpdateAccountIsDirty called with false)
	if v, ok := store.updates[acct1.ID]; !ok || v {
		t.Fatalf("expected acct1 cleared (false), got %v", store.updates)
	}
	// acct2 failed -> should not clear dirty
	if v, ok := store.updates[acct2.ID]; ok && !v {
		t.Fatalf("expected acct2 still dirty (no update or true), got %v", store.updates)
	}
}
