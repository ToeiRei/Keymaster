package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/security"
)

type fakeRemoteSelectiveFail struct {
	content   []byte
	deployErr error
}

func (f *fakeRemoteSelectiveFail) DeployAuthorizedKeys(content string) error { return f.deployErr }
func (f *fakeRemoteSelectiveFail) GetAuthorizedKeys() ([]byte, error)        { return f.content, nil }
func (f *fakeRemoteSelectiveFail) Close()                                    {}

func TestCleanupRemoteAuthorizedKeysSelective_SelectiveDeployFail_ReturnsError(t *testing.T) {
	acct := model.Account{ID: 55, Username: "u", Hostname: "h"}
	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteSelectiveFail{content: []byte("# Keymaster Managed Keys\nssh-ed25519 AAA\n"), deployErr: errors.New("deploy fail")}, nil
	}

	res := &DecommissionResult{}
	opts := DecommissionOptions{SelectiveKeys: []int{1}}
	err := cleanupRemoteAuthorizedKeysSelective(acct, nil, opts, res)
	if err == nil {
		t.Fatalf("expected error when deploy fails during selective cleanup")
	}
	if !strings.Contains(err.Error(), "failed to update authorized_keys") && !strings.Contains(err.Error(), "failed to remove empty authorized_keys file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
