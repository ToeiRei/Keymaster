package core

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
)

// --- fakes -----------------------------------------------------------------
type fhf struct {
	key string
	err error
}

func (f fhf) FetchHostKey(host string) (string, error) { return f.key, f.err }

type fStore struct {
	lastKnownHost string
	lastKnownKey  string
	gotExport     *model.BackupData
	activeSK      *model.SystemKey
	accounts      []model.Account
}

func (f *fStore) GetAccounts() ([]model.Account, error)                          { return nil, nil }
func (f *fStore) GetAllActiveAccounts() ([]model.Account, error)                 { return f.accounts, nil }
func (f *fStore) GetAllAccounts() ([]model.Account, error)                       { return nil, nil }
func (f *fStore) GetAccount(id int) (*model.Account, error)                      { return nil, nil }
func (f *fStore) AddAccount(username, hostname, label, tags string) (int, error) { return 0, nil }
func (f *fStore) DeleteAccount(accountID int) error                              { return nil }
func (f *fStore) AssignKeyToAccount(keyID, accountID int) error                  { return nil }
func (f *fStore) UpdateAccountIsDirty(id int, dirty bool) error                  { return nil }
func (f *fStore) CreateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (f *fStore) RotateSystemKey(publicKey, privateKey string) (int, error)      { return 0, nil }
func (f *fStore) GetActiveSystemKey() (*model.SystemKey, error)                  { return f.activeSK, nil }
func (f *fStore) AddKnownHostKey(hostname, key string) error {
	f.lastKnownHost = hostname
	f.lastKnownKey = key
	return nil
}
func (f *fStore) ExportDataForBackup() (*model.BackupData, error)   { return f.gotExport, nil }
func (f *fStore) ImportDataFromBackup(d *model.BackupData) error    { f.gotExport = d; return nil }
func (f *fStore) IntegrateDataFromBackup(d *model.BackupData) error { f.gotExport = d; return nil }

// satisfy updated Store interface
func (f *fStore) ToggleAccountStatus(id int, enabled bool) error      { return nil }
func (f *fStore) UpdateAccountHostname(id int, hostname string) error { return nil }
func (f *fStore) UpdateAccountLabel(id int, label string) error       { return nil }
func (f *fStore) UpdateAccountTags(id int, tags string) error         { return nil }

// small adapters used by tests
type badStore struct{ *fStore }

func (b *badStore) AddKnownHostKey(hostname, key string) error { return errors.New("boom") }

type rs struct{ *fStore }

func (r *rs) RotateSystemKey(publicKey, privateKey string) (int, error) { return 7, nil }

// implement minimal methods to satisfy StoreFactory and DBMaintainer fakes
type fFactory struct {
	target *fStore
	err    error
}

func (f fFactory) NewStoreFromDSN(dbType, dsn string) (Store, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.target, nil
}

type fMaint struct{ gotType, gotDsn string }

func (m *fMaint) RunDBMaintenance(dbType, dsn string) error {
	m.gotType = dbType
	m.gotDsn = dsn
	return nil
}

type fKG struct {
	pub, priv string
	err       error
}

func (k *fKG) GenerateAndMarshalEd25519Key(comment, passphrase string) (string, string, error) {
	return k.pub, k.priv, k.err
}

type fKM struct{ added []string }

func (k *fKM) AddPublicKey(alg string, keyData string, comment string, managed bool, expiresAt time.Time) error {
	if comment == "dup" {
		return errors.New("dup")
	}
	k.added = append(k.added, comment)
	return nil
}
func (k *fKM) AssignKeyToAccount(keyID, accountID int) error     { return nil }
func (k *fKM) UnassignKeyFromAccount(keyID, accountID int) error { return nil }
func (k *fKM) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	if comment == "dup" {
		return nil, errors.New("dup")
	}
	k.added = append(k.added, comment)
	return &model.PublicKey{Algorithm: algorithm, KeyData: keyData, Comment: comment}, nil
}
func (k *fKM) DeletePublicKey(id int) error                                   { return nil }
func (k *fKM) GetAccountsForKey(keyID int) ([]model.Account, error)           { return nil, nil }
func (k *fKM) GetAllPublicKeys() ([]model.PublicKey, error)                   { return nil, nil }
func (k *fKM) GetGlobalPublicKeys() ([]model.PublicKey, error)                { return nil, nil }
func (k *fKM) GetPublicKeyByComment(comment string) (*model.PublicKey, error) { return nil, nil }
func (k *fKM) GetKeysForAccount(accountID int) ([]model.PublicKey, error)     { return nil, nil }
func (k *fKM) SetPublicKeyExpiry(id int, expiresAt time.Time) error           { return nil }
func (k *fKM) TogglePublicKeyGlobal(id int) error                             { return nil }

type fDM struct{ deployed []model.Account }

func (d *fDM) DeployForAccount(account model.Account, keepFile bool) error {
	d.deployed = append(d.deployed, account)
	return nil
}
func (d *fDM) AuditSerial(account model.Account) error { return nil }
func (d *fDM) AuditStrict(account model.Account) error { return nil }
func (d *fDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (DecommissionResult, error) {
	return DecommissionResult{Account: account, AccountID: account.ID}, nil
}
func (d *fDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]DecommissionResult, error) {
	res := make([]DecommissionResult, 0, len(accounts))
	for _, a := range accounts {
		res = append(res, DecommissionResult{Account: a, AccountID: a.ID})
	}
	return res, nil
}
func (d *fDM) CanonicalizeHostPort(host string) string           { return host }
func (d *fDM) ParseHostPort(host string) (string, string, error) { return host, "22", nil }
func (d *fDM) GetRemoteHostKey(host string) (string, error)      { return "rk", nil }
func (d *fDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) {
	return []byte("ssh-ed25519 AAA... test@keymaster"), nil
}
func (d *fDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (d *fDM) IsPassphraseRequired(err error) bool { return false }

// --- tests -----------------------------------------------------------------
func TestTrustHost_SaveSuccess(t *testing.T) {
	st := &fStore{}
	hf := fhf{key: "ssh-rsa AAAKEY"}
	k, err := TrustHost(context.TODO(), "example.com:22", hf, st, true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if k != "ssh-rsa AAAKEY" {
		t.Fatalf("unexpected key: %s", k)
	}
	if st.lastKnownHost != "example.com:22" || st.lastKnownKey != k {
		t.Fatalf("store not updated")
	}
}

func TestTrustHost_SaveFail(t *testing.T) {
	st := &fStore{}
	// adapter type to override AddKnownHostKey with an error
	bs := &badStore{st}
	_, err := TrustHost(context.TODO(), "host", fhf{key: "k"}, bs, true)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunRotateKeyCmd(t *testing.T) {
	kg := &fKG{pub: "pub", priv: "priv"}
	// wrap store to override RotateSystemKey
	st := &fStore{}
	r := &rs{st}
	n, err := RunRotateKeyCmd(context.TODO(), kg, r, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != 7 {
		t.Fatalf("unexpected rotate result: %d", n)
	}
}

func TestRunDeployCmd(t *testing.T) {
	a1 := model.Account{ID: 1, Username: "u", Hostname: "h", IsActive: true}
	st := &fStore{accounts: []model.Account{a1}}
	dm := &fDM{}
	res, err := RunDeployCmd(context.TODO(), st, dm, nil, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result")
	}
	if dm.deployed[0].ID != 1 {
		t.Fatalf("deploy not called")
	}
}

func TestImportAuthorizedKeys_RunImportCmd(t *testing.T) {
	input := "# comment\nssh-ed25519 AAAA test@1\nssh-ed25519 AAAA dup\nnot a key\n"
	km := &fKM{}
	r := strings.NewReader(input)
	imported, skipped, err := RunImportCmd(context.TODO(), r, km, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1 imported, got %d", imported)
	}
	if skipped < 1 {
		t.Fatalf("expected skipped >=1")
	}
}

func TestWriteAndRestoreBackup_Migrate(t *testing.T) {
	data := &model.BackupData{SchemaVersion: 1}
	var buf bytes.Buffer
	if err := RunWriteBackupCmd(context.TODO(), data, &buf); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	st2 := &fStore{}
	if err := RunRestoreCmd(context.TODO(), &buf, RestoreOptions{Full: true}, st2); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if st2.gotExport == nil {
		t.Fatalf("store did not receive backup")
	}
	src := &fStore{gotExport: data}
	tgt := &fStore{}
	fac := fFactory{target: tgt}
	if err := RunMigrateCmd(context.TODO(), fac, src, "sqlite", "dsn"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if tgt.gotExport == nil {
		t.Fatalf("target did not get import")
	}
}

func TestRunDBMaintainCmd(t *testing.T) {
	m := &fMaint{}
	if err := RunDBMaintainCmd(context.TODO(), m, "sqlite", "x", DBMaintenanceOptions{}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if m.gotType != "sqlite" {
		t.Fatalf("unexpected type")
	}
}

func TestRunDecommissionCmd_Single(t *testing.T) {
	acc := model.Account{ID: 5, Username: "u", Hostname: "h"}
	st := &fStore{activeSK: &model.SystemKey{PrivateKey: "pkey"}}
	dm := &fDM{}
	summary, err := RunDecommissionCmd(context.TODO(), []model.Account{acc}, nil, dm, st, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if summary.Successful+summary.Failed+summary.Skipped != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestInitializeServices_NoopAndWrappers(t *testing.T) {
	// InitializeServices currently a noop returning nil
	s, err := InitializeServices(context.TODO(), nil)
	if s != nil || err != nil {
		t.Fatalf("expected nil,nil got %v %v", s, err)
	}

	// RunDeployForAccount wrapper
	dm := &fDM{}
	acc := model.Account{ID: 9}
	if err := RunDeployForAccount(context.TODO(), dm, acc, nil); err != nil {
		t.Fatalf("deploy for account err: %v", err)
	}

	// RunAuditCmd wrapper should delegate (pass-through)
	st := &fStore{accounts: []model.Account{acc}}
	if _, err := RunAuditCmd(context.TODO(), st, dm, "serial", nil); err != nil {
		t.Fatalf("run audit cmd: %v", err)
	}

	// RunExportSSHConfigCmd
	st2 := &fStore{accounts: []model.Account{{ID: 1, Username: "u", Hostname: "h", Label: "lbl"}}}
	cfg, err := RunExportSSHConfigCmd(context.TODO(), st2)
	if err != nil {
		t.Fatalf("export ssh config err: %v", err)
	}
	if !strings.Contains(cfg, "Host") {
		t.Fatalf("unexpected config: %s", cfg)
	}

	// RunBackupCmd
	data := &model.BackupData{SchemaVersion: 1}
	st3 := &fStore{gotExport: data}
	b, err := RunBackupCmd(context.TODO(), st3)
	if err != nil {
		t.Fatalf("backup cmd err: %v", err)
	}
	if b == nil {
		t.Fatalf("expected backup data")
	}

	// Recover/Cleanup/Signal wrappers — call for coverage (they delegate to bootstrap)
	_ = RecoverFromCrash()
	StartSessionReaper()
	InstallSignalHandler()
	_ = CleanupAllActiveSessions()

	// IsDBInitialized uses package-level setter
	SetDefaultDBIsInitialized(func() bool { return true })
	if !IsDBInitialized() {
		t.Fatalf("expected DB initialized true")
	}
	SetDefaultDBIsInitialized(nil)
}

func TestExtractNonKeymasterContent(t *testing.T) {
	content := "line1\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA... a\n# comment\notherline\nend\n"
	out := extractNonKeymasterContent(content)
	if !strings.Contains(out, "line1") || !strings.Contains(out, "otherline") {
		t.Fatalf("unexpected non-keymaster content: %q", out)
	}
}

type fakeDeployerLocal struct {
	content           []byte
	deployed          string
	getErr, deployErr error
}

func (f *fakeDeployerLocal) DeployAuthorizedKeys(content string) error {
	f.deployed = content
	return f.deployErr
}
func (f *fakeDeployerLocal) GetAuthorizedKeys() ([]byte, error) { return f.content, f.getErr }
func (f *fakeDeployerLocal) Close()                             {}

type fakeKR2 struct {
	active   *model.SystemKey
	bySerial map[int]*model.SystemKey
}

func (f *fakeKR2) GetAllPublicKeys() ([]model.PublicKey, error)  { return nil, nil }
func (f *fakeKR2) GetActiveSystemKey() (*model.SystemKey, error) { return f.active, nil }
func (f *fakeKR2) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	if k, ok := f.bySerial[serial]; ok {
		return k, nil
	}
	return nil, nil
}

type fakeKL2 struct {
	globals []model.PublicKey
	acc     map[int][]model.PublicKey
}

func (f *fakeKL2) GetGlobalPublicKeys() ([]model.PublicKey, error) { return f.globals, nil }
func (f *fakeKL2) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return f.acc[accountID], nil
}
func (f *fakeKL2) GetAllPublicKeys() ([]model.PublicKey, error) {
	var out []model.PublicKey
	out = append(out, f.globals...)
	for _, v := range f.acc {
		out = append(out, v...)
	}
	return out, nil
}

func TestRemoveSelectiveKeymasterContent_Update(t *testing.T) {
	// prepare authorized_keys that contains a keymaster section and non-keymaster lines
	auth := "preline\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA key1\n# end\npostline\n"
	fd := &fakeDeployerLocal{content: []byte(auth)}

	// set key reader/list mocks to return a system key and a public key
	sk := &model.SystemKey{Serial: 1, PublicKey: "ssh-ed25519 AAA key1"}
	kr := &fakeKR2{active: sk, bySerial: map[int]*model.SystemKey{1: sk}}
	kl := &fakeKL2{globals: []model.PublicKey{{ID: 10, Algorithm: "ssh-ed25519", KeyData: "AAA", Comment: "c1"}}, acc: map[int][]model.PublicKey{5: {{ID: 11, Algorithm: "ssh-ed25519", KeyData: "BBB", Comment: "c2"}}}}
	SetDefaultKeyReader(kr)
	SetDefaultKeyLister(kl)
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	res := &DecommissionResult{}
	if err := removeSelectiveKeymasterContent(fd, res, 5, nil, true); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if fd.deployed == "" {
		t.Fatalf("expected deployed content to be set")
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone true")
	}
}

func TestRemoveSelectiveKeymasterContent_NoFile(t *testing.T) {
	fd := &fakeDeployerLocal{getErr: os.ErrNotExist}
	res := &DecommissionResult{}
	if err := removeSelectiveKeymasterContent(fd, res, 5, nil, true); err != nil {
		t.Fatalf("expected nil on no such file, got %v", err)
	}
}
