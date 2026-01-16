package core

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

// fake deployer that returns provided content or error
type fakeRemoteFetch struct {
	content []byte
	ferr    error
}

func (f *fakeRemoteFetch) DeployAuthorizedKeys(content string) error { return nil }
func (f *fakeRemoteFetch) GetAuthorizedKeys() ([]byte, error)        { return f.content, f.ferr }
func (f *fakeRemoteFetch) Close()                                    {}

// KeyReader that returns nil for system key by serial to simulate missing key
type fakeKRNil struct{}

func (f *fakeKRNil) GetAllPublicKeys() ([]model.PublicKey, error)              { return nil, nil }
func (f *fakeKRNil) GetActiveSystemKey() (*model.SystemKey, error)             { return nil, nil }
func (f *fakeKRNil) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) { return nil, nil }

func TestFetchAuthorizedKeys_Success(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKR{})

	// override factory to return content
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteFetch{content: []byte("ok-keys")}, nil
	}

	acct := model.Account{ID: 50, Username: "u", Hostname: "h", Serial: 0}
	dm := builtinDeployerManager{}
	got, err := dm.FetchAuthorizedKeys(acct)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if string(got) != "ok-keys" {
		t.Fatalf("unexpected content: %s", string(got))
	}
}

func TestFetchAuthorizedKeys_NoSystemKeyBySerial_Error(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKRNil{})

	acct := model.Account{ID: 51, Username: "u", Hostname: "h", Serial: 99}
	dm := builtinDeployerManager{}
	if _, err := dm.FetchAuthorizedKeys(acct); err == nil {
		t.Fatalf("expected error when no system key for serial, got nil")
	}
}

func TestFetchAuthorizedKeys_DeployerFactoryError(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKR{})

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return nil, errors.New("factory fail")
	}

	acct := model.Account{ID: 52, Username: "u", Hostname: "h", Serial: 0}
	dm := builtinDeployerManager{}
	if _, err := dm.FetchAuthorizedKeys(acct); err == nil {
		t.Fatalf("expected error when factory fails, got nil")
	}
}

func TestFetchAuthorizedKeys_GetAuthorizedKeysError(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKR{})

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteFetch{content: nil, ferr: errors.New("remote err")}, nil
	}

	acct := model.Account{ID: 53, Username: "u", Hostname: "h", Serial: 0}
	dm := builtinDeployerManager{}
	if _, err := dm.FetchAuthorizedKeys(acct); err == nil {
		t.Fatalf("expected error when GetAuthorizedKeys fails, got nil")
	}
}
