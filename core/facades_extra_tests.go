package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/core/model"
)

func TestExtractNonKeymasterContent_Simple(t *testing.T) {
	content := "before\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA foo@a\n# comment\nafter\n"
	got := extractNonKeymasterContent(content)
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Fatalf("unexpected output: %q", got)
	}
}

// fakes local to this file to avoid name collisions with other tests
type fd3 struct {
	content           []byte
	deployed          string
	getErr, deployErr error
}

func (f *fd3) DeployAuthorizedKeys(content string) error { f.deployed = content; return f.deployErr }
func (f *fd3) GetAuthorizedKeys() ([]byte, error)        { return f.content, f.getErr }
func (f *fd3) Close()                                    {}

type kr3 struct {
	active *model.SystemKey
	by     map[int]*model.SystemKey
}

func (k *kr3) GetAllPublicKeys() ([]model.PublicKey, error)  { return nil, nil }
func (k *kr3) GetActiveSystemKey() (*model.SystemKey, error) { return k.active, nil }
func (k *kr3) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	if v, ok := k.by[serial]; ok {
		return v, nil
	}
	return nil, nil
}

type kl3 struct {
	globals []model.PublicKey
	acc     map[int][]model.PublicKey
}

func (k *kl3) GetGlobalPublicKeys() ([]model.PublicKey, error) { return k.globals, nil }
func (k *kl3) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return k.acc[accountID], nil
}
func (k *kl3) GetAllPublicKeys() ([]model.PublicKey, error) {
	var out []model.PublicKey
	out = append(out, k.globals...)
	for _, v := range k.acc {
		out = append(out, v...)
	}
	return out, nil
}

func TestRemoveSelectiveKeymasterContent_EndToEnd(t *testing.T) {
	auth := "pre\n# Keymaster Managed Keys (Serial: 1)\nssh-ed25519 AAA key1\n# end\npost\n"
	deployer := &fd3{content: []byte(auth)}

	sk := &model.SystemKey{Serial: 1, PublicKey: "ssh-ed25519 AAA key1"}
	kr := &kr3{active: sk, by: map[int]*model.SystemKey{1: sk}}
	kl := &kl3{globals: []model.PublicKey{{ID: 1, Algorithm: "ssh-ed25519", KeyData: "AAA", Comment: "g"}}, acc: map[int][]model.PublicKey{42: {{ID: 2, Algorithm: "ssh-ed25519", KeyData: "BBB", Comment: "a"}}}}

	SetDefaultKeyReader(kr)
	SetDefaultKeyLister(kl)
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	res := &DecommissionResult{}
	if err := removeSelectiveKeymasterContent(deployer, res, 42, nil, true); err != nil {
		t.Fatalf("remove returned err: %v", err)
	}
	if deployer.deployed == "" {
		t.Fatalf("expected deploy to be called")
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone true")
	}

	// test no such file path
	d2 := &fd3{getErr: errors.New("no such file")}
	res2 := &DecommissionResult{}
	if err := removeSelectiveKeymasterContent(d2, res2, 42, nil, true); err != nil {
		t.Fatalf("expected nil on no such file, got %v", err)
	}
}

func TestGenerateSelectiveKeysContent_Basic(t *testing.T) {
	sk := &model.SystemKey{Serial: 5, PublicKey: "ssh-ed25519 AAA pub"}
	kr := &kr3{active: sk, by: map[int]*model.SystemKey{5: sk}}
	kl := &kl3{globals: []model.PublicKey{{ID: 10, Algorithm: "ssh-ed25519", KeyData: "G", Comment: "g"}}, acc: map[int][]model.PublicKey{7: {{ID: 11, Algorithm: "ssh-ed25519", KeyData: "A", Comment: "a"}}}}
	SetDefaultKeyReader(kr)
	SetDefaultKeyLister(kl)
	defer func() { SetDefaultKeyReader(nil); SetDefaultKeyLister(nil) }()

	s, err := GenerateSelectiveKeysContent(7, 0, nil, false)
	if err != nil {
		t.Fatalf("generate selective failed: %v", err)
	}
	if !strings.Contains(s, "Keymaster Managed Keys") || !strings.Contains(s, "ssh-ed25519") {
		t.Fatalf("unexpected generated content: %q", s)
	}
}
