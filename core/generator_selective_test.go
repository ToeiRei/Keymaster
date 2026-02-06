package core

import (
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/core/model"
)

type krTest struct{}

func (kr *krTest) GetActiveSystemKey() (*model.SystemKey, error) {
	return &model.SystemKey{Serial: 1, PublicKey: "ssh-ed25519 AAA sys"}, nil
}
func (kr *krTest) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) {
	return &model.SystemKey{Serial: serial, PublicKey: "ssh-ed25519 AAA sys"}, nil
}
func (kr *krTest) GetAllPublicKeys() ([]model.PublicKey, error) {
	return nil, nil
}

type klTest struct {
	globals []model.PublicKey
	acc     map[int][]model.PublicKey
}

func (k *klTest) GetGlobalPublicKeys() ([]model.PublicKey, error) { return k.globals, nil }
func (k *klTest) GetKeysForAccount(accountID int) ([]model.PublicKey, error) {
	return k.acc[accountID], nil
}
func (k *klTest) GetAllPublicKeys() ([]model.PublicKey, error) {
	var out []model.PublicKey
	out = append(out, k.globals...)
	for _, v := range k.acc {
		out = append(out, v...)
	}
	return out, nil
}

func TestGenerateSelectiveKeysContent_ExcludeAndExpire(t *testing.T) {
	// set up key reader/lister
	SetDefaultKeyReader(&krTest{})
	defer SetDefaultKeyReader(nil)
	now := time.Now().UTC()
	expired := now.Add(-24 * time.Hour)

	kl := &klTest{
		globals: []model.PublicKey{{ID: 10, Algorithm: "ssh-ed25519", KeyData: "G1", Comment: "ga"}},
		acc: map[int][]model.PublicKey{
			7: {
				{ID: 20, Algorithm: "ssh-ed25519", KeyData: "A1", Comment: "a1"},
				{ID: 21, Algorithm: "ssh-ed25519", KeyData: "A2", Comment: "a2", ExpiresAt: expired},
			},
		},
	}
	SetDefaultKeyLister(kl)
	defer SetDefaultKeyLister(nil)

	// exclude key ID 10 (global) and expect only ID 20 included (21 expired)
	out, err := GenerateSelectiveKeysContent(7, 1, []int{10}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "G1") {
		t.Fatalf("excluded global key present in output")
	}
	if !strings.Contains(out, "A1") {
		t.Fatalf("expected account key A1 present")
	}
	if strings.Contains(out, "A2") {
		t.Fatalf("expired key A2 should be filtered")
	}
}

func TestGenerateSelectiveKeysContent_NoKeyLister_Error(t *testing.T) {
	SetDefaultKeyReader(&krTest{})
	defer SetDefaultKeyReader(nil)
	SetDefaultKeyLister(nil)

	if _, err := GenerateSelectiveKeysContent(1, 1, nil, false); err == nil {
		t.Fatalf("expected error when no KeyLister available")
	}
}
