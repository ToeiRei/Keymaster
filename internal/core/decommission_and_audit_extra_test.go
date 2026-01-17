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

// Test when removeSystemKey=false and no excludes: non-keymaster content should be preserved
type fakeDeployerPreserve struct {
	content  []byte
	deployed string
}

func (f *fakeDeployerPreserve) DeployAuthorizedKeys(content string) error {
	f.deployed = content
	return nil
}
func (f *fakeDeployerPreserve) GetAuthorizedKeys() ([]byte, error) { return f.content, nil }
func (f *fakeDeployerPreserve) Close()                             {}

func TestRemoveSelectiveKeymasterContent_RemoveSystemKeyFalse_PreservesNonKeymaster(t *testing.T) {
	auth := "keepthis\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA key1\n# end\nkeepthat\n"
	fd := &fakeDeployerPreserve{content: []byte(auth)}
	res := &DecommissionResult{}

	if err := removeSelectiveKeymasterContent(fd, res, 77, nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone true")
	}
	if !strings.Contains(fd.deployed, "keepthis") || !strings.Contains(fd.deployed, "keepthat") {
		t.Fatalf("expected non-keymaster content preserved, got %q", fd.deployed)
	}
	if strings.Contains(fd.deployed, "ssh-ed25519") {
		t.Fatalf("expected keymaster-managed lines removed, got %q", fd.deployed)
	}
}

// Test AuditAccounts branch where UpdateAccountIsDirty fails and AUDIT_HASH_MARK_DIRTY_FAILED is logged
type fakeStoreAudit struct {
	accounts  []model.Account
	updateErr error
}

func (f *fakeStoreAudit) GetAllActiveAccounts() ([]model.Account, error) { return f.accounts, nil }

// stub the rest of Store interface
func (f *fakeStoreAudit) GetAccounts() ([]model.Account, error)     { return nil, nil }
func (f *fakeStoreAudit) GetAllAccounts() ([]model.Account, error)  { return nil, nil }
func (f *fakeStoreAudit) GetAccount(id int) (*model.Account, error) { return nil, nil }
func (f *fakeStoreAudit) AddAccount(username, hostname, label, tags string) (int, error) {
	return 0, nil
}
func (f *fakeStoreAudit) DeleteAccount(accountID int) error                         { return nil }
func (f *fakeStoreAudit) AssignKeyToAccount(keyID, accountID int) error             { return nil }
func (f *fakeStoreAudit) UpdateAccountIsDirty(id int, dirty bool) error             { return f.updateErr }
func (f *fakeStoreAudit) CreateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (f *fakeStoreAudit) RotateSystemKey(publicKey, privateKey string) (int, error) { return 0, nil }
func (f *fakeStoreAudit) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "k"}, nil
}
func (f *fakeStoreAudit) AddKnownHostKey(hostname, key string) error      { return nil }
func (f *fakeStoreAudit) ExportDataForBackup() (*model.BackupData, error) { return nil, nil }
func (f *fakeStoreAudit) ImportDataFromBackup(*model.BackupData) error    { return nil }
func (f *fakeStoreAudit) IntegrateDataFromBackup(*model.BackupData) error { return nil }

type fakeDMForAudit struct{}

func (f *fakeDMForAudit) DeployForAccount(model.Account, bool) error { return nil }
func (f *fakeDMForAudit) AuditSerial(model.Account) error            { return nil }
func (f *fakeDMForAudit) AuditStrict(model.Account) error            { return nil }
func (f *fakeDMForAudit) DecommissionAccount(model.Account, security.Secret, interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (f *fakeDMForAudit) BulkDecommissionAccounts([]model.Account, security.Secret, interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (f *fakeDMForAudit) CanonicalizeHostPort(host string) string           { return host }
func (f *fakeDMForAudit) ParseHostPort(host string) (string, string, error) { return host, "", nil }
func (f *fakeDMForAudit) GetRemoteHostKey(string) (string, error)           { return "", nil }
func (f *fakeDMForAudit) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return []byte("remote-content"), nil
}
func (f *fakeDMForAudit) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeDMForAudit) IsPassphraseRequired(err error) bool { return false }

type spyAuditW2 struct{ actions []string }

func (s *spyAuditW2) LogAction(action, details string) error {
	s.actions = append(s.actions, action+":"+details)
	return nil
}

func TestAuditAccounts_MarkDirtyFail_LogsMarker(t *testing.T) {
	i18n.Init("en")
	st := &fakeStoreAudit{accounts: []model.Account{{ID: 99, Username: "u", Hostname: "h", Serial: 1}}, updateErr: errors.New("update failed")}
	dm := &fakeDMForAudit{}
	aw := &spyAuditW2{}
	SetDefaultAuditWriter(aw)
	defer SetDefaultAuditWriter(nil)
	// Set key reader/lister so GenerateKeysContent succeeds (but differs from remote)
	SetDefaultKeyReader(&krTest{})
	SetDefaultKeyLister(&klTest{globals: nil, acc: map[int][]model.PublicKey{}})
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	_, err := AuditAccounts(context.TODO(), st, dm, "strict", nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	found := false
	for _, a := range aw.actions {
		if strings.HasPrefix(a, "AUDIT_HASH_MARK_DIRTY_FAILED") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected AUDIT_HASH_MARK_DIRTY_FAILED logged, got %v", aw.actions)
	}
}
