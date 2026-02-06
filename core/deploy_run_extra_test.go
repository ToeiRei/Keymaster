package core

import (
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/security"
	"github.com/toeirei/keymaster/internal/i18n"
)

// fakeKRPartial returns a system key for serial lookups but nil for active key
type fakeKRPartial struct{}

func (f *fakeKRPartial) GetAllPublicKeys() ([]model.PublicKey, error)  { return nil, nil }
func (f *fakeKRPartial) GetActiveSystemKey() (*model.SystemKey, error) { return nil, nil }
func (f *fakeKRPartial) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return &model.SystemKey{Serial: serial, PublicKey: "p", PrivateKey: "priv", IsActive: false}, nil
}

// fakeKRGood returns a valid active system key
type fakeKRGood struct{}

func (f *fakeKRGood) GetAllPublicKeys() ([]model.PublicKey, error) { return nil, nil }
func (f *fakeKRGood) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 7, PublicKey: "p", PrivateKey: "priv", IsActive: true}, nil
}
func (f *fakeKRGood) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return &model.SystemKey{Serial: serial, PublicKey: "p", PrivateKey: "priv", IsActive: true}, nil
}

// krNilSerial returns nil for GetSystemKeyBySerial to simulate missing serial key
type krNilSerial struct{}

func (k *krNilSerial) GetAllPublicKeys() ([]model.PublicKey, error) { return nil, nil }
func (k *krNilSerial) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "p", PrivateKey: "priv", IsActive: true}, nil
}
func (k *krNilSerial) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) { return nil, nil }

func TestRunDeploymentForAccount_NoKeyReader_ReturnsError(t *testing.T) {
	i18n.Init("en")
	// ensure no key reader is set
	SetDefaultKeyReader(nil)

	acct := model.Account{ID: 200, Username: "u", Hostname: "h", Serial: 0}
	if err := RunDeploymentForAccount(acct, false); err == nil {
		t.Fatalf("expected error when no DefaultKeyReader, got nil")
	}
}

func TestRunDeploymentForAccount_NoSystemKeyBySerial_TUI_NonTUI(t *testing.T) {
	i18n.Init("en")
	// fake reader that returns nil for system key by serial
	SetDefaultKeyReader(&krNilSerial{})

	acct := model.Account{ID: 201, Username: "u", Hostname: "h", Serial: 999}
	if err := RunDeploymentForAccount(acct, false); err == nil {
		t.Fatalf("expected error when no system key for serial (non-TUI), got nil")
	}
	if err := RunDeploymentForAccount(acct, true); err == nil {
		t.Fatalf("expected error when no system key for serial (TUI), got nil")
	}
}

func TestRunDeploymentForAccount_ActiveKeyNil_Error(t *testing.T) {
	i18n.Init("en")
	// connect key from serial present but active key is nil
	SetDefaultKeyReader(&fakeKRPartial{})

	// provide deployer factory that succeeds so function progresses to activeKey check
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteRun{deployErr: nil}, nil
	}

	acct := model.Account{ID: 202, Username: "u", Hostname: "h", Serial: 5}
	if err := RunDeploymentForAccount(acct, false); err == nil {
		t.Fatalf("expected error when active key is nil, got nil")
	}
}

func TestRunDeploymentForAccount_UpdaterNil_Error(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKRGood{})
	// ensure updater is nil
	origUpd := DefaultAccountSerialUpdater()
	SetDefaultAccountSerialUpdater(nil)
	defer SetDefaultAccountSerialUpdater(origUpd)

	// stub deployer factory to succeed
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteRun{deployErr: nil}, nil
	}

	acct := model.Account{ID: 203, Username: "u", Hostname: "h", Serial: 0}
	if err := RunDeploymentForAccount(acct, false); err == nil {
		t.Fatalf("expected error when AccountSerialUpdater is nil, got nil")
	}
}
