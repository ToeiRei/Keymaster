package core

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/security"
)

// fake store used elsewhere removed to avoid unused helper warning

// fake KeyManager for ImportAuthorizedKeys (minimal)
type fmKeyManager struct {
	added   []string
	failFor map[string]error
}

func (f *fmKeyManager) AddPublicKey(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) error {
	if f.failFor != nil {
		if e, ok := f.failFor[comment]; ok {
			return e
		}
	}
	f.added = append(f.added, comment)
	return nil
}

func (f *fmKeyManager) GetGlobalPublicKeys() ([]model.PublicKey, error)            { return nil, nil }
func (f *fmKeyManager) GetKeysForAccount(accountID int) ([]model.PublicKey, error) { return nil, nil }
func (f *fmKeyManager) AssignKeyToAccount(keyID, accountID int) error              { return nil }

func TestImportAuthorizedKeys_Basic(t *testing.T) {
	data := "# header\nssh-ed25519 AAAA key-one\ninvalid-line\nssh-ed25519 BBBB key-two\nssh-ed25519 CCCC\n"
	km := &fmKeyManager{}
	r := strings.NewReader(data)
	imported, skipped, err := ImportAuthorizedKeys(context.TODO(), r, km, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imported != 2 {
		t.Fatalf("expected 2 imported, got %d", imported)
	}
	if skipped < 1 {
		t.Fatalf("expected at least 1 skipped, got %d", skipped)
	}
}

func TestExportSSHConfig_And_FindAccount(t *testing.T) {
	// empty
	stEmpty := &simpleStore{accounts: []model.Account{}}
	out, err := ExportSSHConfig(context.TODO(), stEmpty)
	if err != nil {
		t.Fatalf("ExportSSHConfig error: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output for no accounts, got %q", out)
	}

	// non-empty
	a1 := model.Account{ID: 1, Username: "alice", Hostname: "a.example.com", Label: ""}
	a2 := model.Account{ID: 2, Username: "bob", Hostname: "b.example.com", Label: "team"}
	st := &simpleStore{accounts: []model.Account{a1, a2}}
	out2, err := ExportSSHConfig(context.TODO(), st)
	if err != nil {
		t.Fatalf("ExportSSHConfig error: %v", err)
	}
	if !strings.Contains(out2, "HostName a.example.com") || !strings.Contains(out2, "User alice") {
		t.Fatalf("unexpected ssh config output: %q", out2)
	}

	// FindAccountByIdentifier tests
	if acc, err := FindAccountByIdentifier("1", st.accounts); err != nil || acc == nil || acc.ID != 1 {
		t.Fatalf("FindAccountByIdentifier by id failed: %v %v", acc, err)
	}
	if acc, err := FindAccountByIdentifier("alice@a.example.com", st.accounts); err != nil || acc == nil || acc.Username != "alice" {
		t.Fatalf("FindAccountByIdentifier by user@host failed: %v %v", acc, err)
	}
	if acc, err := FindAccountByIdentifier("team", st.accounts); err != nil || acc == nil || acc.Label != "team" {
		t.Fatalf("FindAccountByIdentifier by label failed: %v %v", acc, err)
	}
	if _, err := FindAccountByIdentifier("nope", st.accounts); err == nil {
		t.Fatalf("expected error for missing identifier")
	}
}

func TestParallelRun_CollectsResults(t *testing.T) {
	a1 := model.Account{Username: "u1", Hostname: "h1"}
	a2 := model.Account{Username: "u2", Hostname: "h2"}
	accounts := []model.Account{a1, a2}
	worker := func(a model.Account) error {
		if a.Username == "u2" {
			return errors.New("boom")
		}
		return nil
	}
	res := ParallelRun(context.TODO(), accounts, worker)
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	var errs int
	for _, r := range res {
		if r.Error != nil {
			errs++
		}
	}
	if errs != 1 {
		t.Fatalf("expected 1 error result, got %d", errs)
	}
}

func TestWriteBackup_Compresses(t *testing.T) {
	data := &model.BackupData{SchemaVersion: 1}
	var buf bytes.Buffer
	if err := WriteBackup(context.TODO(), data, &buf); err != nil {
		t.Fatalf("WriteBackup failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected non-empty buffer after WriteBackup")
	}
}

// DeployerManager that returns authorized_keys content
type dmForImport struct{}

func (d *dmForImport) DeployForAccount(account model.Account, keepFile bool) error { return nil }
func (d *dmForImport) AuditSerial(account model.Account) error                     { return nil }
func (d *dmForImport) AuditStrict(account model.Account) error                     { return nil }
func (d *dmForImport) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{}, nil
}
func (d *dmForImport) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	return nil, nil
}
func (d *dmForImport) CanonicalizeHostPort(host string) string           { return host }
func (d *dmForImport) ParseHostPort(host string) (string, string, error) { return host, "22", nil }
func (d *dmForImport) GetRemoteHostKey(host string) (string, error)      { return "hk", nil }
func (d *dmForImport) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return []byte("ssh-ed25519 AAAA key1\nssh-ed25519 BBBB key2\n"), nil
}
func (d *dmForImport) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (d *dmForImport) IsPassphraseRequired(err error) bool { return false }

func TestRunImportRemoteCmd_Success(t *testing.T) {
	dm := &dmForImport{}
	km := &fmKeyManager{}
	imp, skip, warn, err := RunImportRemoteCmd(context.TODO(), model.Account{ID: 1}, dm, km, nil)
	if err != nil {
		t.Fatalf("RunImportRemoteCmd error: %v", err)
	}
	if imp != 2 || skip != 0 || warn != "" {
		t.Fatalf("unexpected import result: imp=%d skip=%d warn=%q", imp, skip, warn)
	}
}
