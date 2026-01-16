package core

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

type fakeRemoteDeployer2 struct {
	getContent []byte
	deployErr  error
}

func (f *fakeRemoteDeployer2) DeployAuthorizedKeys(content string) error { return f.deployErr }
func (f *fakeRemoteDeployer2) GetAuthorizedKeys() ([]byte, error)        { return f.getContent, nil }
func (f *fakeRemoteDeployer2) Close()                                    {}

func TestRemoveSelectiveKeymasterContent_GenerateSelectiveKeysContentError(t *testing.T) {
	// Make NewDeployerFactory return a deployer whose GetAuthorizedKeys returns
	// a Keymaster-managed section. Then force GenerateSelectiveKeysContent to
	// fail by clearing the default KeyLister.
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		content := "# Keymaster Managed Keys (Serial: 1)\nssh-rsa AAAAB3Nza...\n# end\nnon-keymaster-line"
		return &fakeRemoteDeployer2{getContent: []byte(content), deployErr: nil}, nil
	}

	// Clear KeyLister to force GenerateSelectiveKeysContent to error.
	SetDefaultKeyLister(nil)
	defer SetDefaultKeyLister(&fakeKL2{})

	var res DecommissionResult
	acct := model.Account{ID: 42, Username: "u", Hostname: "h"}

	err := removeSelectiveKeymasterContent(&fakeRemoteDeployer2{getContent: []byte("# Keymaster Managed Keys\nssh-rsa AAA\n")}, &res, acct.ID, nil, true)
	if err == nil {
		t.Fatalf("expected error when GenerateSelectiveKeysContent fails")
	}
	if !strings.Contains(err.Error(), "failed to generate keys content") && !strings.Contains(err.Error(), "no key lister") {
		t.Fatalf("unexpected error: %v", err)
	}
}
