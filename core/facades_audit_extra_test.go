package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
)

// fake store that fails UpdateAccountIsDirty
type failingDirtyStore struct {
	accounts []model.Account
}

func (s *failingDirtyStore) GetAccounts() ([]model.Account, error)          { return nil, nil }
func (s *failingDirtyStore) GetAllActiveAccounts() ([]model.Account, error) { return s.accounts, nil }
func (s *failingDirtyStore) GetAllAccounts() ([]model.Account, error)       { return nil, nil }
func (s *failingDirtyStore) GetAccount(id int) (*model.Account, error)      { return nil, nil }
func (s *failingDirtyStore) AddAccount(username, hostname, label, tags string) (int, error) {
	return 0, nil
}
func (s *failingDirtyStore) DeleteAccount(accountID int) error             { return nil }
func (s *failingDirtyStore) AssignKeyToAccount(keyID, accountID int) error { return nil }
func (s *failingDirtyStore) UpdateAccountIsDirty(id int, dirty bool) error {
	return errors.New("update fail")
}
func (s *failingDirtyStore) CreateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (s *failingDirtyStore) RotateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (s *failingDirtyStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "sys"}, nil
}
func (s *failingDirtyStore) AddKnownHostKey(hostname, key string) error      { return nil }
func (s *failingDirtyStore) ExportDataForBackup() (*model.BackupData, error) { return nil, nil }
func (s *failingDirtyStore) ImportDataFromBackup(*model.BackupData) error    { return nil }
func (s *failingDirtyStore) IntegrateDataFromBackup(*model.BackupData) error { return nil }

// satisfy updated Store interface
func (s *failingDirtyStore) ToggleAccountStatus(id int, enabled bool) error      { return nil }
func (s *failingDirtyStore) UpdateAccountHostname(id int, hostname string) error { return nil }
func (s *failingDirtyStore) UpdateAccountLabel(id int, label string) error       { return nil }
func (s *failingDirtyStore) UpdateAccountTags(id int, tags string) error         { return nil }

// fake DM that returns mismatched content
type mismatchDM struct{}

func (m *mismatchDM) DeployForAccount(account model.Account, keepFile bool) error { return nil }
func (m *mismatchDM) AuditSerial(account model.Account) error                     { return nil }
func (m *mismatchDM) AuditStrict(account model.Account) error                     { return nil }
func (m *mismatchDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (m *mismatchDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (m *mismatchDM) CanonicalizeHostPort(host string) string           { return host }
func (m *mismatchDM) ParseHostPort(host string) (string, string, error) { return host, "22", nil }
func (m *mismatchDM) GetRemoteHostKey(host string) (string, error)      { return "hostkey", nil }
func (m *mismatchDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return []byte("different content"), nil
}
func (m *mismatchDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (m *mismatchDM) IsPassphraseRequired(err error) bool { return false }

// serialDM records whether AuditSerial was invoked
type serialDM struct{ Called *bool }

func (s *serialDM) DeployForAccount(account model.Account, keepFile bool) error { return nil }
func (s *serialDM) AuditSerial(account model.Account) error {
	if s.Called != nil {
		*s.Called = true
	}
	return nil
}
func (s *serialDM) AuditStrict(account model.Account) error { return nil }
func (s *serialDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (s *serialDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (s *serialDM) CanonicalizeHostPort(host string) string                   { return host }
func (s *serialDM) ParseHostPort(host string) (string, string, error)         { return host, "22", nil }
func (s *serialDM) GetRemoteHostKey(host string) (string, error)              { return "hostkey", nil }
func (s *serialDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return nil, nil }
func (s *serialDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (s *serialDM) IsPassphraseRequired(err error) bool { return false }

func TestAuditAccounts_MarkDirtyFailure_LogsFailure(t *testing.T) {
	// store with one active account
	acct := model.Account{ID: 42, Username: "u", Hostname: "h", Serial: 1, IsActive: true}
	store := &failingDirtyStore{accounts: []model.Account{acct}}
	dm := &mismatchDM{}

	aw := &spyAuditWriter{}
	SetDefaultAuditWriter(aw)
	defer SetDefaultAuditWriter(nil)

	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	res, err := AuditAccounts(context.TODO(), store, dm, "strict", nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result")
	}
	// ensure mark-dirty failed event logged
	found := false
	for _, a := range aw.actions {
		if strings.HasPrefix(a, "AUDIT_HASH_MARK_DIRTY_FAILED") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected AUDIT_HASH_MARK_DIRTY_FAILED in audit actions: %v", aw.actions)
	}
}

// Test that AuditAccounts in 'serial' mode delegates to DeployerManager.AuditSerial
func TestAuditAccounts_SerialMode_Delegates(t *testing.T) {
	acct := model.Account{ID: 99, Username: "u", Hostname: "h", Serial: 3, IsActive: true}
	store := &simpleFakeStore{accounts: []model.Account{acct}}
	called := false
	dm2 := &serialDM{Called: &called}
	_, err := AuditAccounts(context.TODO(), store, dm2, "serial", nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatalf("expected AuditSerial to be called")
	}
}
