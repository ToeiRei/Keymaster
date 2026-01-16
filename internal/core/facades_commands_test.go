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

type callCountingDM struct {
	calls []model.Account
}

func (c *callCountingDM) DeployForAccount(account model.Account, keepFile bool) error {
	c.calls = append(c.calls, account)
	if account.ID == 999 {
		return errors.New("fail")
	}
	return nil
}
func (c *callCountingDM) AuditSerial(account model.Account) error { return nil }
func (c *callCountingDM) AuditStrict(account model.Account) error { return nil }
func (c *callCountingDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (c *callCountingDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (c *callCountingDM) CanonicalizeHostPort(host string) string                   { return host }
func (c *callCountingDM) ParseHostPort(host string) (string, string, error)         { return host, "22", nil }
func (c *callCountingDM) GetRemoteHostKey(host string) (string, error)              { return "hk", nil }
func (c *callCountingDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return nil, nil }
func (c *callCountingDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (c *callCountingDM) IsPassphraseRequired(err error) bool { return false }

// fetchFailDM simulates FetchAuthorizedKeys failure
type fetchFailDM struct{}

func (f *fetchFailDM) DeployForAccount(account model.Account, keepFile bool) error { return nil }
func (f *fetchFailDM) AuditSerial(account model.Account) error                     { return nil }
func (f *fetchFailDM) AuditStrict(account model.Account) error                     { return nil }
func (f *fetchFailDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (f *fetchFailDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (f *fetchFailDM) CanonicalizeHostPort(host string) string           { return host }
func (f *fetchFailDM) ParseHostPort(host string) (string, string, error) { return host, "22", nil }
func (f *fetchFailDM) GetRemoteHostKey(host string) (string, error)      { return "hk", nil }
func (f *fetchFailDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return nil, errors.New("fetch fail")
}
func (f *fetchFailDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fetchFailDM) IsPassphraseRequired(err error) bool { return false }

// hostErrDM simulates GetRemoteHostKey failure
type hostErrDM struct{}

func (h *hostErrDM) DeployForAccount(account model.Account, keepFile bool) error { return nil }
func (h *hostErrDM) AuditSerial(account model.Account) error                     { return nil }
func (h *hostErrDM) AuditStrict(account model.Account) error                     { return nil }
func (h *hostErrDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (h *hostErrDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (h *hostErrDM) CanonicalizeHostPort(host string) string                   { return host }
func (h *hostErrDM) ParseHostPort(host string) (string, string, error)         { return host, "22", nil }
func (h *hostErrDM) GetRemoteHostKey(host string) (string, error)              { return "", errors.New("no") }
func (h *hostErrDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return nil, nil }
func (h *hostErrDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (h *hostErrDM) IsPassphraseRequired(err error) bool { return false }

type simpleStore struct {
	accounts []model.Account
	known    map[string]string
}

func (s *simpleStore) GetAllActiveAccounts() ([]model.Account, error)                 { return s.accounts, nil }
func (s *simpleStore) GetAllAccounts() ([]model.Account, error)                       { return nil, nil }
func (s *simpleStore) GetAccounts() ([]model.Account, error)                          { return nil, nil }
func (s *simpleStore) GetAccount(id int) (*model.Account, error)                      { return nil, nil }
func (s *simpleStore) AddAccount(username, hostname, label, tags string) (int, error) { return 0, nil }
func (s *simpleStore) DeleteAccount(accountID int) error                              { return nil }
func (s *simpleStore) AssignKeyToAccount(keyID, accountID int) error                  { return nil }
func (s *simpleStore) UpdateAccountIsDirty(id int, dirty bool) error                  { return nil }
func (s *simpleStore) CreateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (s *simpleStore) RotateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (s *simpleStore) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "p", PrivateKey: "priv", IsActive: true}, nil
}
func (s *simpleStore) AddKnownHostKey(hostname, key string) error {
	if s.known == nil {
		s.known = map[string]string{}
	}
	s.known[hostname] = key
	return nil
}
func (s *simpleStore) ExportDataForBackup() (*model.BackupData, error) { return nil, nil }
func (s *simpleStore) ImportDataFromBackup(*model.BackupData) error    { return nil }
func (s *simpleStore) IntegrateDataFromBackup(*model.BackupData) error { return nil }

type hfOK struct{ key string }

func (h hfOK) FetchHostKey(canonicalHost string) (string, error) { return h.key, nil }

type hfErr struct{}

func (hfErr) FetchHostKey(canonicalHost string) (string, error) { return "", errors.New("no host") }

func TestDeployAccounts_AllAndIdentifier(t *testing.T) {
	i18n.Init("en")
	acct1 := model.Account{ID: 1, Username: "alice", Hostname: "a.example.com", Label: ""}
	acct2 := model.Account{ID: 2, Username: "bob", Hostname: "b.example.com", Label: "team"}
	st := &simpleStore{accounts: []model.Account{acct1, acct2}}
	dm := &callCountingDM{}

	// all
	res, err := DeployAccounts(context.TODO(), st, dm, nil, nil)
	if err != nil {
		t.Fatalf("DeployAccounts failed: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if len(dm.calls) != 2 {
		t.Fatalf("expected dm called twice, got %d", len(dm.calls))
	}

	// identifier by user@host
	id := "alice@a.example.com"
	dm.calls = nil
	res2, err := DeployAccounts(context.TODO(), st, dm, &id, nil)
	if err != nil {
		t.Fatalf("expected no error for identifier, got %v", err)
	}
	if len(res2) != 1 {
		t.Fatalf("expected 1 result for identifier, got %d", len(res2))
	}
}

func TestDeployAccounts_NotFound(t *testing.T) {
	i18n.Init("en")
	st := &simpleStore{accounts: []model.Account{{ID: 1, Username: "x", Hostname: "h"}}}
	dm := &callCountingDM{}
	id := "noone@nowhere"
	if _, err := DeployAccounts(context.TODO(), st, dm, &id, nil); err == nil {
		t.Fatalf("expected error for missing identifier, got nil")
	}
}

func TestRunAuditForAccount_Modes(t *testing.T) {
	i18n.Init("en")
	dm := &callCountingDM{}
	acct := model.Account{ID: 5, Username: "u", Hostname: "h"}
	if err := RunAuditForAccount(context.TODO(), dm, acct, "serial", nil); err != nil {
		t.Fatalf("audit serial failed: %v", err)
	}
	if err := RunAuditForAccount(context.TODO(), dm, acct, "strict", nil); err != nil {
		t.Fatalf("audit strict failed: %v", err)
	}
	if err := RunAuditForAccount(context.TODO(), dm, acct, "", nil); err != nil {
		t.Fatalf("audit default strict failed: %v", err)
	}
	if err := RunAuditForAccount(context.TODO(), dm, acct, "badmode", nil); err == nil {
		t.Fatalf("expected error for bad mode, got nil")
	}
}

func TestRunImportRemoteCmd_FetchError(t *testing.T) {
	i18n.Init("en")
	dm := &fetchFailDM{}
	if _, _, _, err := RunImportRemoteCmd(context.TODO(), model.Account{ID: 1}, dm, nil, nil); err == nil {
		t.Fatalf("expected error when fetch fails, got nil")
	}
}

func TestRunTrustHostCmd_SaveAndNoSave(t *testing.T) {
	i18n.Init("en")
	st := &simpleStore{}
	dm := &callCountingDM{}
	// success path
	key, err := RunTrustHostCmd(context.TODO(), "host", dm, st, true)
	if err != nil {
		t.Fatalf("expected no error saving hostkey: %v", err)
	}
	if !strings.HasPrefix(key, "hk") && key != "hk" { /* ok */
	}
	if _, ok := st.known["host"]; !ok {
		t.Fatalf("expected known host saved")
	}

	// error path: use hostErrDM
	dm2 := &hostErrDM{}
	if _, err := RunTrustHostCmd(context.TODO(), "h2", dm2, st, false); err == nil {
		t.Fatalf("expected error when GetRemoteHostKey fails, got nil")
	}
}
