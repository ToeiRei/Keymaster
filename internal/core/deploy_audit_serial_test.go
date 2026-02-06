package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/security"
)

// fakeRemote from deploy_audit_test.go reused pattern
type fakeRemoteSerial struct {
	content []byte
	ferr    error
}

func (f *fakeRemoteSerial) DeployAuthorizedKeys(content string) error { return nil }
func (f *fakeRemoteSerial) GetAuthorizedKeys() ([]byte, error)        { return f.content, f.ferr }
func (f *fakeRemoteSerial) Close()                                    {}

func TestAuditAccountSerial_Match_NoError(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 200, Username: "u", Hostname: "h", Serial: 5}

	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	header := "# Keymaster Managed Keys (Serial: 5)\n"
	body := header + "ssh-ed25519 AAAA... comment\n"

	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteSerial{content: []byte(body)}, nil
	}

	if err := AuditAccountSerial(acct); err != nil {
		t.Fatalf("unexpected error from AuditAccountSerial: %v", err)
	}
}

func TestAuditAccountSerial_Mismatch_Error(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 201, Username: "u2", Hostname: "h2", Serial: 7}

	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	header := "# Keymaster Managed Keys (Serial: 3)\n"
	body := header + "ssh-ed25519 AAAA... comment\n"

	orig := NewDeployerFactory
	defer func() { NewDeployerFactory = orig }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteSerial{content: []byte(body)}, nil
	}

	if err := AuditAccountSerial(acct); err == nil {
		t.Fatalf("expected error from AuditAccountSerial on serial mismatch, got nil")
	}
}
