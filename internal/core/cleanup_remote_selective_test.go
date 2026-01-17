package core

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

type failFactoryDeployer struct{}

func (f *failFactoryDeployer) DeployAuthorizedKeys(content string) error { return nil }
func (f *failFactoryDeployer) GetAuthorizedKeys() ([]byte, error)        { return nil, nil }
func (f *failFactoryDeployer) Close()                                    {}

func TestCleanupRemoteAuthorizedKeysSelective_FactoryError(t *testing.T) {
	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return nil, errors.New("connect fail")
	}

	acct := model.Account{ID: 1, Username: "u", Hostname: "h"}
	var res DecommissionResult
	err := cleanupRemoteAuthorizedKeysSelective(acct, nil, DecommissionOptions{SelectiveKeys: nil, KeepFile: false}, &res)
	if err == nil {
		t.Fatalf("expected error when NewDeployerFactory fails")
	}
}

func TestCleanupRemoteAuthorizedKeysSelective_KeepFile_Delegates(t *testing.T) {
	// Return a deployer whose GetAuthorizedKeys returns content; ensure delegated removal runs
	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteDeployer{getContent: []byte("pre\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA key1\n# end\npost\n")}, nil
	}

	// ensure GenerateSelectiveKeysContent can run by setting readers
	SetDefaultKeyReader(&krTest{})
	SetDefaultKeyLister(&klTest{globals: nil, acc: map[int][]model.PublicKey{1: {{ID: 100, Algorithm: "ssh-ed25519", KeyData: "key1", Comment: "c"}}}})
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	acct := model.Account{ID: 1, Username: "u", Hostname: "h"}
	var res DecommissionResult
	err := cleanupRemoteAuthorizedKeysSelective(acct, nil, DecommissionOptions{SelectiveKeys: nil, KeepFile: true}, &res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone true")
	}
}
