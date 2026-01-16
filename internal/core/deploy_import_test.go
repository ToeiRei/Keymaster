package core

import (
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

type fakeDeployerForImport struct {
	content []byte
	ferr    error
}

func (f *fakeDeployerForImport) DeployAuthorizedKeys(content string) error { return nil }
func (f *fakeDeployerForImport) GetAuthorizedKeys() ([]byte, error)        { return f.content, f.ferr }
func (f *fakeDeployerForImport) Close()                                    {}

type fakeImporter struct {
	created []*model.PublicKey
	ferr    error
}

func (f *fakeImporter) AddPublicKeyAndGetModel(algorithm, keyData, comment string, isGlobal bool, expiresAt time.Time) (*model.PublicKey, error) {
	if f.ferr != nil {
		return nil, f.ferr
	}
	pk := &model.PublicKey{ID: len(f.created) + 1, Algorithm: algorithm, KeyData: keyData, Comment: comment}
	f.created = append(f.created, pk)
	return pk, nil
}

// krBadSerial returns nil for GetSystemKeyBySerial to simulate missing serial key
type krBadSerial struct{}

func (k *krBadSerial) GetAllPublicKeys() ([]model.PublicKey, error) { return nil, nil }
func (k *krBadSerial) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "p", PrivateKey: "priv", IsActive: true}, nil
}
func (k *krBadSerial) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) { return nil, nil }

func TestImportRemoteKeys_ImporterNil_SkipsAll(t *testing.T) {
	i18n.Init("en")
	// no key reader -> warning
	SetDefaultKeyReader(nil)

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeDeployerForImport{content: []byte("ssh-rsa AAA comment\ninvalidline")}, nil
	}

	// ensure importer is nil
	origImporter := DefaultKeyImporter()
	SetDefaultKeyImporter(nil)
	defer SetDefaultKeyImporter(origImporter)

	acct := model.Account{ID: 21, Username: "u", Hostname: "h"}
	imported, skipped, warning, err := ImportRemoteKeys(acct)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(imported) != 0 {
		t.Fatalf("expected no imported keys when importer nil, got %d", len(imported))
	}
	if skipped == 0 {
		t.Fatalf("expected skipped > 0, got 0")
	}
	if warning == "" {
		t.Fatalf("expected warning when no key reader")
	}
}

func TestImportRemoteKeys_SystemKeyBySerialMissing_Error(t *testing.T) {
	i18n.Init("en")
	// key reader returns nil for serial lookups
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyReader(&krBadSerial{})

	acct := model.Account{ID: 22, Username: "u", Hostname: "h", Serial: 999}
	if _, _, _, err := ImportRemoteKeys(acct); err == nil {
		t.Fatalf("expected error when system key by serial missing, got nil")
	}
}

func TestImportRemoteKeys_SuccessfulImport(t *testing.T) {
	i18n.Init("en")
	// provide a key reader with active key
	SetDefaultKeyReader(&fakeKR{})

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeDeployerForImport{content: []byte("ssh-rsa AAA key1\nssh-rsa BBB key2\n")}, nil
	}

	imp := &fakeImporter{}
	origImp := DefaultKeyImporter()
	SetDefaultKeyImporter(imp)
	defer SetDefaultKeyImporter(origImp)

	acct := model.Account{ID: 23, Username: "u", Hostname: "h"}
	imported, skipped, warning, err := ImportRemoteKeys(acct)
	if err != nil {
		t.Fatalf("ImportRemoteKeys error: %v", err)
	}
	if len(imported) != 2 {
		t.Fatalf("expected 2 imported keys, got %d", len(imported))
	}
	if skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", skipped)
	}
	if warning != "" {
		t.Fatalf("expected no warning, got %s", warning)
	}
}
