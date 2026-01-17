package core

import (
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

type fakeDeployerPerm struct {
	content  []byte
	deployed string
}

func (f *fakeDeployerPerm) DeployAuthorizedKeys(content string) error {
	f.deployed = content
	return nil
}
func (f *fakeDeployerPerm) GetAuthorizedKeys() ([]byte, error) { return f.content, nil }
func (f *fakeDeployerPerm) Close()                             {}

func TestRemoveSelectiveKeymasterContent_ExcludeIDsAndMergeNonKeymaster(t *testing.T) {
	// authorized_keys has keymaster section and non-keymaster lines
	auth := "preline\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA key1\nssh-ed25519 BBB key2\n# end\npostline\n"
	fd := &fakeDeployerPerm{content: []byte(auth)}

	// set reader/lister so GenerateSelectiveKeysContent will produce key1+key2 lines
	SetDefaultKeyReader(&krTest{})
	kl := &klTest{globals: []model.PublicKey{{ID: 50, Algorithm: "ssh-ed25519", KeyData: "G1", Comment: "g"}}, acc: map[int][]model.PublicKey{42: {{ID: 60, Algorithm: "ssh-ed25519", KeyData: "key1", Comment: "k1"}, {ID: 61, Algorithm: "ssh-ed25519", KeyData: "key2", Comment: "k2"}}}}
	SetDefaultKeyLister(kl)
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	res := &DecommissionResult{}
	// exclude key ID 61 (key2)
	if err := removeSelectiveKeymasterContent(fd, res, 42, []int{61}, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone")
	}
	// deployed content should contain non-keymaster lines and only key1, not key2
	if !strings.Contains(fd.deployed, "preline") || !strings.Contains(fd.deployed, "postline") {
		t.Fatalf("non-keymaster content lost: %q", fd.deployed)
	}
	if strings.Contains(fd.deployed, "key2") {
		t.Fatalf("excluded key2 still present: %q", fd.deployed)
	}
	if !strings.Contains(fd.deployed, "key1") {
		t.Fatalf("expected key1 present: %q", fd.deployed)
	}
}
