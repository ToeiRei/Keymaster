package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/i18n"
)

type fakeRemoteDeployer struct {
	getContent []byte
	deployErr  error
}

func (f *fakeRemoteDeployer) DeployAuthorizedKeys(content string) error { return f.deployErr }
func (f *fakeRemoteDeployer) GetAuthorizedKeys() ([]byte, error)        { return f.getContent, nil }
func (f *fakeRemoteDeployer) Close()                                    {}

type fakeAccountMgr struct {
	deleted []int
	ferr    error
}

func (f *fakeAccountMgr) DeleteAccount(id int) error {
	f.deleted = append(f.deleted, id)
	return f.ferr
}

type spyAuditW struct {
	actions []string
}

func (s *spyAuditW) LogAction(action, details string) error {
	s.actions = append(s.actions, action+":"+details)
	return nil
}

func TestDecommissionAccount_DryRun(t *testing.T) {
	i18n.Init("en")
	aw := &spyAuditW{}
	SetDefaultAuditWriter(aw)
	defer SetDefaultAuditWriter(nil)

	acct := model.Account{ID: 10, Username: "u", Hostname: "h"}
	res := DecommissionAccount(acct, nil, DecommissionOptions{DryRun: true})
	if !res.Skipped {
		t.Fatalf("expected skipped for dry run")
	}
	found := false
	for _, a := range aw.actions {
		if strings.HasPrefix(a, "DECOMMISSION_DRYRUN") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DECOMMISSION_DRYRUN logged, got %v", aw.actions)
	}
}

func TestDecommissionAccount_RemoteCleanupFail_NoForce(t *testing.T) {
	i18n.Init("en")
	// stub deployer factory to return deployer whose DeployAuthorizedKeys fails
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteDeployer{deployErr: errors.New("remote err")}, nil
	}

	aw := &spyAuditW{}
	SetDefaultAuditWriter(aw)
	defer SetDefaultAuditWriter(nil)

	mgr := &fakeAccountMgr{}
	origMgr := DefaultAccountManager()
	SetDefaultAccountManager(mgr)
	defer SetDefaultAccountManager(origMgr)

	acct := model.Account{ID: 11, Username: "u", Hostname: "h"}
	res := DecommissionAccount(acct, nil, DecommissionOptions{Force: false})
	if !res.Skipped {
		t.Fatalf("expected skipped when remote cleanup fails and not forced")
	}
	if res.RemoteCleanupError == nil {
		t.Fatalf("expected remote cleanup error set")
	}
	found := false
	for _, a := range aw.actions {
		if strings.HasPrefix(a, "DECOMMISSION_FAILED") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DECOMMISSION_FAILED logged, got %v", aw.actions)
	}
	if len(mgr.deleted) != 0 {
		t.Fatalf("expected no DB delete when skipped, got %v", mgr.deleted)
	}
}

func TestDecommissionAccount_RemoteCleanupFail_ForceTrue(t *testing.T) {
	i18n.Init("en")
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteDeployer{deployErr: errors.New("remote err")}, nil
	}

	aw := &spyAuditW{}
	SetDefaultAuditWriter(aw)
	defer SetDefaultAuditWriter(nil)

	mgr := &fakeAccountMgr{}
	origMgr := DefaultAccountManager()
	SetDefaultAccountManager(mgr)
	defer SetDefaultAccountManager(origMgr)

	acct := model.Account{ID: 12, Username: "u", Hostname: "h"}
	res := DecommissionAccount(acct, nil, DecommissionOptions{Force: true})
	if res.Skipped {
		t.Fatalf("did not expect skipped when force=true")
	}
	if !res.DatabaseDeleteDone {
		t.Fatalf("expected DB delete done")
	}
	if res.RemoteCleanupError == nil {
		t.Fatalf("expected remote cleanup error recorded")
	}
	// success log expected
	found := false
	for _, a := range aw.actions {
		if strings.HasPrefix(a, "DECOMMISSION_SUCCESS") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DECOMMISSION_SUCCESS logged, got %v", aw.actions)
	}
}

func TestDecommissionAccount_NoAccountManager_Error(t *testing.T) {
	i18n.Init("en")
	// Make cleanup succeed
	origFactory := NewDeployerFactory
	defer func() { NewDeployerFactory = origFactory }()
	NewDeployerFactory = func(host, user string, privateKey security.Secret, passphrase []byte) (RemoteDeployer, error) {
		return &fakeRemoteDeployer{deployErr: nil}, nil
	}

	// ensure no account manager
	origMgr := DefaultAccountManager()
	SetDefaultAccountManager(nil)
	defer SetDefaultAccountManager(origMgr)

	acct := model.Account{ID: 13, Username: "u", Hostname: "h"}
	res := DecommissionAccount(acct, nil, DecommissionOptions{})
	if res.DatabaseDeleteError == nil {
		t.Fatalf("expected DatabaseDeleteError when no account manager configured")
	}
}
