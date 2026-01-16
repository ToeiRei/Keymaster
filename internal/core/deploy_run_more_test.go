package core

import (
	"errors"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/security"
)

// retryUpdater simulates transient DB "database is locked" errors a few times
type retryUpdater struct {
	calls      int
	failCount  int
	lastID     int
	lastSerial int
}

func (r *retryUpdater) UpdateAccountSerial(accountID int, serial int) error {
	r.calls++
	r.lastID = accountID
	r.lastSerial = serial
	if r.calls <= r.failCount {
		return errors.New("database is locked")
	}
	return nil
}

func TestRunDeploymentForAccount_UpdaterRetriesEventuallySucceeds(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteRun{deployErr: nil}, nil
	}

	upd := &retryUpdater{failCount: 2}
	origUpd := DefaultAccountSerialUpdater()
	SetDefaultAccountSerialUpdater(upd)
	defer SetDefaultAccountSerialUpdater(origUpd)

	acct := model.Account{ID: 400, Username: "u", Hostname: "h", Serial: 1}
	start := time.Now()
	if err := RunDeploymentForAccount(acct, false); err != nil {
		t.Fatalf("expected deployment to succeed after retries, got err: %v", err)
	}
	if upd.lastID != acct.ID {
		t.Fatalf("updater called with wrong id: got %d expected %d", upd.lastID, acct.ID)
	}
	if upd.calls <= upd.failCount {
		t.Fatalf("expected retries, got calls=%d failCount=%d", upd.calls, upd.failCount)
	}
	// ensure some retry backoff happened (not strict, just sanity)
	if time.Since(start) < 50*time.Millisecond {
		t.Fatalf("expected some retry delay, elapsed too short")
	}
}

func TestRunDeploymentForAccount_DeployerFactoryFails_ReturnsError(t *testing.T) {
	i18n.Init("en")
	SetDefaultKeyReader(&fakeKR{})
	SetDefaultKeyLister(&fakeKL{})

	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return nil, errors.New("connect failure")
	}

	acct := model.Account{ID: 401, Username: "u2", Hostname: "h2", Serial: 1}
	if err := RunDeploymentForAccount(acct, true); err == nil {
		t.Fatalf("expected error when deployer factory fails, got nil")
	}
}
