package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/core/model"
)

// Additional permutations for removeSelectiveKeymasterContent

type fakeRemoteDeployerForRemove struct {
	content    []byte
	deployed   string
	closed     bool
	failDeploy bool
}

func (f *fakeRemoteDeployerForRemove) DeployAuthorizedKeys(content string) error {
	if f.failDeploy {
		return errors.New("deploy fail")
	}
	f.deployed = content
	return nil
}
func (f *fakeRemoteDeployerForRemove) GetAuthorizedKeys() ([]byte, error) {
	return append([]byte(nil), f.content...), nil
}
func (f *fakeRemoteDeployerForRemove) Close() { f.closed = true }

func TestRemoveSelective_ExcludeIDs_UsesNonKeymasterContent(t *testing.T) {
	// arrange: authorized_keys has keymaster section and extra lines
	fd := &fakeRemoteDeployerForRemove{content: []byte("# Keymaster Managed Keys\nssh-ed25519 A\n\npreline\npostline\n")}
	res := &DecommissionResult{}

	// ensure GenerateSelectiveKeysContent returns empty (no keys) by making default lister return empty lists
	origKL := DefaultKeyLister()
	defer func() { SetDefaultKeyLister(origKL) }()
	SetDefaultKeyLister(&localFakeKeyLister2{gkeys: nil, akeys: nil})

	if err := removeSelectiveKeymasterContent(fd, res, 5, []int{42}, true); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if fd.deployed == "" {
		t.Fatalf("expected deployed content to be set")
	}
	if fd.deployed != "preline\npostline\n" && fd.deployed != "preline\npostline" {
		t.Fatalf("unexpected deployed content: %q", fd.deployed)
	}
}

func TestRemoveSelective_GenerateError_Propagates(t *testing.T) {
	fd := &fakeRemoteDeployerForRemove{content: []byte("# Keymaster Managed Keys\nssh-ed25519 A\n")}
	res := &DecommissionResult{}

	// cause GenerateSelectiveKeysContent to fail by removing KeyLister
	origKL := DefaultKeyLister()
	origKR := DefaultKeyReader()
	defer func() { SetDefaultKeyLister(origKL); SetDefaultKeyReader(origKR) }()
	SetDefaultKeyLister(nil)

	err := removeSelectiveKeymasterContent(fd, res, 7, nil, true)
	if err == nil || !strings.Contains(err.Error(), "failed to generate keys content") {
		t.Fatalf("expected wrapped generate error, got %v", err)
	}
}

func TestRemoveSelective_FinalEmpty_DeploysEmpty(t *testing.T) {
	fd := &fakeRemoteDeployerForRemove{content: []byte("# Keymaster Managed Keys\nssh-ed25519 A\n")}
	res := &DecommissionResult{}

	// make GenerateSelectiveKeysContent return empty by providing empty lister and reader returning nil system key
	origKL := DefaultKeyLister()
	origKR := DefaultKeyReader()
	defer func() { SetDefaultKeyLister(origKL); SetDefaultKeyReader(origKR) }()
	SetDefaultKeyLister(&localFakeKeyLister2{gkeys: nil, akeys: nil})
	SetDefaultKeyReader(&localFakeKeyReader2{sys: nil, ferr: nil})

	if err := removeSelectiveKeymasterContent(fd, res, 99, nil, true); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if fd.deployed != "" {
		t.Fatalf("expected empty deploy, got %q", fd.deployed)
	}
}

// lightweight fakes for KeyLister/KeyReader used in tests
type localFakeKeyLister2 struct {
	gkeys []model.PublicKey
	akeys []model.PublicKey
}

func (f *localFakeKeyLister2) GetGlobalPublicKeys() ([]model.PublicKey, error) { return f.gkeys, nil }
func (f *localFakeKeyLister2) GetKeysForAccount(id int) ([]model.PublicKey, error) {
	return f.akeys, nil
}
func (f *localFakeKeyLister2) GetAllPublicKeys() ([]model.PublicKey, error) {
	return append([]model.PublicKey(nil), f.gkeys...), nil
}

type localFakeKeyReader2 struct {
	sys  *model.SystemKey
	ferr error
}

func (f *localFakeKeyReader2) GetActiveSystemKey() (*model.SystemKey, error) { return f.sys, f.ferr }
func (f *localFakeKeyReader2) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return f.sys, f.ferr
}
func (f *localFakeKeyReader2) GetAllPublicKeys() ([]model.PublicKey, error) { return nil, nil }
