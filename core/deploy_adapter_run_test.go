package core

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/i18n"
)

// fake bootstrap deployer used to test NewBootstrapDeployer wrapper
type fakeBootstrap struct {
	received string
	ferr     error
}

func (f *fakeBootstrap) DeployAuthorizedKeys(content string) error {
	f.received = content
	return f.ferr
}
func (f *fakeBootstrap) Close() {}

// fake remote deployer for RunDeploymentForAccount
type fakeRemoteRun struct {
	deployErr error
}

func (f *fakeRemoteRun) DeployAuthorizedKeys(content string) error { return f.deployErr }
func (f *fakeRemoteRun) GetAuthorizedKeys() ([]byte, error)        { return nil, nil }
func (f *fakeRemoteRun) Close()                                    {}

type recordingUpdater struct {
	lastID     int
	lastSerial int
	ferr       error
}

func (f *recordingUpdater) UpdateAccountSerial(accountID int, serial int) error {
	f.lastID = accountID
	f.lastSerial = serial
	return f.ferr
}

func TestNewBootstrapDeployer_DelegatesAndErrors(t *testing.T) {
	i18n.Init("en")
	orig := NewBootstrapDeployerFunc
	defer func() { NewBootstrapDeployerFunc = orig }()

	fb := &fakeBootstrap{ferr: errors.New("boom")}
	NewBootstrapDeployerFunc = func(hostname, username string, privateKey interface{}, expectedHostKey string) (BootstrapDeployer, error) {
		return fb, nil
	}

	d, err := NewBootstrapDeployer("h", "u", nil, "")
	if err != nil {
		t.Fatalf("NewBootstrapDeployer returned err: %v", err)
	}
	if d == nil {
		t.Fatalf("expected deployer wrapper, got nil")
	}

	// call DeployAuthorizedKeys through wrapper - expect underlying error
	if err := d.DeployAuthorizedKeys("content"); err == nil {
		t.Fatalf("expected error from underlying DeployAuthorizedKeys, got nil")
	}
}

func TestRunDeploymentForAccount_SuccessAndUpdaterCalled(t *testing.T) {
	i18n.Init("en")
	// set key reader/lister to fakes
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	// inject deployer factory that returns a deployer which succeeds
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteRun{deployErr: nil}, nil
	}

	// inject updater to record UpdateAccountSerial call
	upd := &recordingUpdater{}
	origUpd := DefaultAccountSerialUpdater()
	SetDefaultAccountSerialUpdater(upd)
	defer SetDefaultAccountSerialUpdater(origUpd)

	acct := model.Account{ID: 300, Username: "u", Hostname: "h", Serial: 1}
	if err := RunDeploymentForAccount(acct, false); err != nil {
		t.Fatalf("RunDeploymentForAccount failed: %v", err)
	}
	if upd.lastID != acct.ID || upd.lastSerial == 0 {
		t.Fatalf("expected updater called with account id %d and non-zero serial, got id=%d serial=%d", acct.ID, upd.lastID, upd.lastSerial)
	}
}

func TestRunDeploymentForAccount_DeployFails_ReturnsError(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteRun{deployErr: errors.New("remote failure")}, nil
	}

	// ensure updater is set so code reaches deploy step
	upd := &recordingUpdater{}
	origUpd := DefaultAccountSerialUpdater()
	SetDefaultAccountSerialUpdater(upd)
	defer SetDefaultAccountSerialUpdater(origUpd)

	acct := model.Account{ID: 301, Username: "u2", Hostname: "h2", Serial: 1}
	if err := RunDeploymentForAccount(acct, false); err == nil {
		t.Fatalf("expected error when deploy fails, got nil")
	}
}
