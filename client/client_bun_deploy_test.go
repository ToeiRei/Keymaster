package client

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/toeirei/keymaster/config"
	"github.com/toeirei/keymaster/core"
	"github.com/toeirei/keymaster/core/model"
	"github.com/toeirei/keymaster/core/security"
	"github.com/toeirei/keymaster/ui/tui/util"
)

// fake deployer manager used to avoid network operations in tests.
type fakeDM struct{}

func (f *fakeDM) DeployForAccount(account model.Account, keepFile bool) error { return nil }
func (f *fakeDM) AuditSerial(account model.Account) error                     { return nil }
func (f *fakeDM) AuditStrict(account model.Account) error                     { return nil }
func (f *fakeDM) DecommissionAccount(account model.Account, systemPrivateKey security.Secret, options interface{}) (core.DecommissionResult, error) {
	return core.DecommissionResult{Account: account, AccountID: account.ID}, nil
}
func (f *fakeDM) BulkDecommissionAccounts(accounts []model.Account, systemPrivateKey security.Secret, options interface{}) ([]core.DecommissionResult, error) {
	res := make([]core.DecommissionResult, 0, len(accounts))
	for _, a := range accounts {
		res = append(res, core.DecommissionResult{Account: a, AccountID: a.ID})
	}
	return res, nil
}
func (f *fakeDM) CanonicalizeHostPort(host string) string                   { return host }
func (f *fakeDM) ParseHostPort(host string) (string, string, error)         { return host, "", nil }
func (f *fakeDM) GetRemoteHostKey(host string) (string, error)              { return "", nil }
func (f *fakeDM) FetchAuthorizedKeys(account model.Account) ([]byte, error) { return []byte(""), nil }
func (f *fakeDM) ImportRemoteKeys(account model.Account) ([]model.PublicKey, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeDM) IsPassphraseRequired(err error) bool { return false }

func TestBunClient_Onboard_Decom_Deploy(t *testing.T) {
	cfg := config.Config{Database: config.ConfigDatabase{Type: "sqlite", Dsn: ":memory:"}}
	logger := log.New(io.Discard, "", 0)

	c, err := NewBunClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewBunClient failed: %v", err)
	}
	defer func() { _ = c.Close(context.Background()) }()

	// inject fake deployer manager and bootstrap deployer
	origDM := core.DefaultDeployerManager
	core.DefaultDeployerManager = &fakeDM{}
	defer func() { core.DefaultDeployerManager = origDM }()

	origBootstrap := core.NewBootstrapDeployerFunc
	core.NewBootstrapDeployerFunc = func(hostname, username string, privateKey interface{}, expectedHostKey string) (core.BootstrapDeployer, error) {
		return nil, nil
	}
	defer func() { core.NewBootstrapDeployerFunc = origBootstrap }()

	// Onboard a host
	ch, err := c.OnboardHost(context.Background(), "example.local", 22, "alice", "")
	if err != nil {
		t.Fatalf("OnboardHost failed: %v", err)
	}
	var lastPct float32
	for v := range ch {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("expected final percent 100 got %v", lastPct)
	}

	// Create a target and account
	tgt, err := c.CreateTarget(context.Background(), "example.local", 22)
	if err != nil {
		t.Fatalf("CreateTarget failed: %v", err)
	}
	acct, err := c.CreateAccount(context.Background(), tgt.Id, "bob", "")
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}

	// Create a public key and assign to account
	pk, err := c.CreatePublicKey(context.Background(), "ci-key", util.NewPointer("some comment"), nil)
	if err != nil {
		t.Fatalf("CreatePublicKey failed: %v", err)
	}
	// assign via core KeyManager
	km := core.DefaultKeyManager()
	if km == nil {
		t.Fatalf("no key manager available")
	}
	if err := km.AssignKeyToAccount(int(pk.Id), int(acct.Id)); err != nil {
		t.Fatalf("AssignKeyToAccount failed: %v", err)
	}

	// DeployPublicKeys
	dch, err := c.DeployPublicKeys(context.Background(), pk.Id)
	if err != nil {
		t.Fatalf("DeployPublicKeys failed: %v", err)
	}
	lastPct = 0
	for v := range dch {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("DeployPublicKeys final percent != 100: %v", lastPct)
	}

	// DeployTargets
	dch2, err := c.DeployTargets(context.Background(), tgt.Id)
	if err != nil {
		t.Fatalf("DeployTargets failed: %v", err)
	}
	lastPct = 0
	for v := range dch2 {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("DeployTargets final percent != 100: %v", lastPct)
	}

	// DeployAccounts
	dch3, err := c.DeployAccounts(context.Background(), acct.Id)
	if err != nil {
		t.Fatalf("DeployAccounts failed: %v", err)
	}
	lastPct = 0
	for v := range dch3 {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("DeployAccounts final percent != 100: %v", lastPct)
	}

	// DeployAll
	dch4, err := c.DeployAll(context.Background())
	if err != nil {
		t.Fatalf("DeployAll failed: %v", err)
	}
	lastPct = 0
	for v := range dch4 {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("DeployAll final percent != 100: %v", lastPct)
	}

	// DecommissionTarget
	dech, err := c.DecommisionTarget(context.Background(), tgt.Id)
	if err != nil {
		t.Fatalf("DecommisionTarget failed: %v", err)
	}
	lastPct = 0
	for v := range dech {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("DecommisionTarget final percent != 100: %v", lastPct)
	}

	// DecommissionAccount
	dech2, err := c.DecommisionAccount(context.Background(), acct.Id)
	if err != nil {
		t.Fatalf("DecommisionAccount failed: %v", err)
	}
	lastPct = 0
	for v := range dech2 {
		lastPct = v.Percent
	}
	if lastPct != 100 {
		t.Fatalf("DecommisionAccount final percent != 100: %v", lastPct)
	}
}
