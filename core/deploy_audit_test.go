package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/security"
	"github.com/toeirei/keymaster/internal/i18n"
)

type fakeRemote struct {
	content []byte
	ferr    error
}

func (f *fakeRemote) DeployAuthorizedKeys(content string) error { return nil }
func (f *fakeRemote) GetAuthorizedKeys() ([]byte, error)        { return f.content, f.ferr }
func (f *fakeRemote) Close()                                    {}

func TestAuditAccountStrict_Match_NoError(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 100, Username: "u", Hostname: "h", Serial: 1}

	// ensure key reader/lister produce consistent expected content
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	expected, err := GenerateKeysContent(acct.ID)
	if err != nil {
		t.Fatalf("GenerateKeysContent: %v", err)
	}

	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemote{content: []byte(expected)}, nil
	}

	if err := AuditAccountStrict(acct); err != nil {
		t.Fatalf("unexpected error from AuditAccountStrict: %v", err)
	}
}

func TestAuditAccountStrict_Mismatch_Error(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 101, Username: "u2", Hostname: "h2", Serial: 1}

	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemote{content: []byte("other content")}, nil
	}

	if err := AuditAccountStrict(acct); err == nil {
		t.Fatalf("expected error from AuditAccountStrict on mismatch, got nil")
	}
}
