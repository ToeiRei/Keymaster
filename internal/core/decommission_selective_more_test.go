package core

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

type fakeRemoteDeployer4 struct {
	getContent []byte
	deployed   string
	deployErr  error
}

func (f *fakeRemoteDeployer4) DeployAuthorizedKeys(content string) error {
	f.deployed = content
	return f.deployErr
}
func (f *fakeRemoteDeployer4) GetAuthorizedKeys() ([]byte, error) { return f.getContent, nil }
func (f *fakeRemoteDeployer4) Close()                             {}

type krNil struct{}

func (k *krNil) GetAllPublicKeys() ([]model.PublicKey, error)              { return nil, nil }
func (k *krNil) GetActiveSystemKey() (*model.SystemKey, error)             { return nil, nil }
func (k *krNil) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) { return nil, nil }

func TestRemoveSelectiveKeymasterContent_RemoveSystemKeyTrue_MergesNonKeymaster(t *testing.T) {
	// authorized_keys contains keymaster section and non-keymaster lines
	auth := "preline\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA key1\n# end\npostline\n"
	fd := &fakeRemoteDeployer4{getContent: []byte(auth)}

	// set key reader/lister so GenerateSelectiveKeysContent will produce a keymaster section
	SetDefaultKeyReader(&krTest{})
	kl := &klTest{globals: []model.PublicKey{{ID: 10, Algorithm: "ssh-ed25519", KeyData: "G1", Comment: "g"}}, acc: map[int][]model.PublicKey{5: {{ID: 11, Algorithm: "ssh-ed25519", KeyData: "A1", Comment: "a1"}}}}
	SetDefaultKeyLister(kl)
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	res := &DecommissionResult{}
	if err := removeSelectiveKeymasterContent(fd, res, 5, nil, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone true")
	}
	if !strings.Contains(fd.deployed, "preline") || !strings.Contains(fd.deployed, "postline") {
		t.Fatalf("non-keymaster content lost: %q", fd.deployed)
	}
	// ensure generated key lines are present
	if !strings.Contains(fd.deployed, "A1") || !strings.Contains(fd.deployed, "G1") {
		t.Fatalf("expected generated keys present in deployed content: %q", fd.deployed)
	}
}

func TestGenerateSelectiveKeysContent_NoActiveSystemKey_Error(t *testing.T) {
	// KeyReader that returns nil active key
	SetDefaultKeyReader(&krNil{})
	defer SetDefaultKeyReader(nil)
	SetDefaultKeyLister(&klTest{globals: nil, acc: map[int][]model.PublicKey{}})
	defer SetDefaultKeyLister(nil)

	if _, err := GenerateSelectiveKeysContent(1, 0, nil, false); err == nil {
		t.Fatalf("expected error when no active system key present")
	}
}
