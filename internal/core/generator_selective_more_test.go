package core

import (
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/model"
)

// genKR3 implements KeyReader for tests
type genKR3 struct {
	sys  *model.SystemKey
	ferr error
}

func (g *genKR3) GetActiveSystemKey() (*model.SystemKey, error)             { return g.sys, g.ferr }
func (g *genKR3) GetSystemKeyBySerial(serial int) (*model.SystemKey, error) { return g.sys, g.ferr }
func (g *genKR3) GetAllPublicKeys() ([]model.PublicKey, error)              { return nil, nil }

// genKL3 implements KeyLister for tests
type genKL3 struct {
	global  []model.PublicKey
	account []model.PublicKey
}

func (g *genKL3) GetGlobalPublicKeys() ([]model.PublicKey, error)     { return g.global, nil }
func (g *genKL3) GetKeysForAccount(id int) ([]model.PublicKey, error) { return g.account, nil }
func (g *genKL3) GetAllPublicKeys() ([]model.PublicKey, error)        { return g.global, nil }

func TestGenerateSelectiveKeysContent_SystemKeyBySerialMissing_Error(t *testing.T) {
	origKR := DefaultKeyReader()
	defer SetDefaultKeyReader(origKR)
	SetDefaultKeyReader(&genKR3{sys: nil, ferr: nil})

	if _, err := GenerateSelectiveKeysContent(1, 12345, nil, false); err == nil || !strings.Contains(err.Error(), "no system key found for serial") {
		t.Fatalf("expected system key by serial error, got %v", err)
	}
}

func TestGenerateSelectiveKeysContent_ExcludeAndExpire_More(t *testing.T) {
	sys := &model.SystemKey{Serial: 1, PublicKey: "PUBKEY"}
	origKR := DefaultKeyReader()
	origKL := DefaultKeyLister()
	defer func() { SetDefaultKeyReader(origKR); SetDefaultKeyLister(origKL) }()
	SetDefaultKeyReader(&genKR3{sys: sys, ferr: nil})

	now := time.Now().UTC()
	gkeys := []model.PublicKey{
		{ID: 10, Algorithm: "ssh-rsa", KeyData: "A", Comment: "b", IsGlobal: true, ExpiresAt: time.Time{}},
		{ID: 11, Algorithm: "ssh-ed25519", KeyData: "B", Comment: "a", IsGlobal: true, ExpiresAt: now.Add(-1 * time.Hour)},
	}
	akeys := []model.PublicKey{
		{ID: 12, Algorithm: "ssh-rsa", KeyData: "C", Comment: "c", IsGlobal: false, ExpiresAt: time.Time{}},
	}
	SetDefaultKeyLister(&genKL3{global: gkeys, account: akeys})

	out, err := GenerateSelectiveKeysContent(7, 1, []int{10}, false)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !strings.Contains(out, "# Keymaster Managed Keys (Serial: 1)") {
		t.Fatalf("expected header in output, got: %q", out)
	}
	if !strings.Contains(out, "PUBKEY") {
		t.Fatalf("expected system public key present")
	}
	if !strings.Contains(out, "ssh-rsa C") {
		t.Fatalf("expected account key C present")
	}
	if strings.Contains(out, "ssh-rsa A") {
		t.Fatalf("excluded key A found")
	}
	if strings.Contains(out, "ssh-ed25519 B") {
		t.Fatalf("expired key B should be filtered")
	}
}

func TestGenerateSelectiveKeysContent_RemoveSystemKeyTrue_OmitsSystemKey(t *testing.T) {
	origKL := DefaultKeyLister()
	defer SetDefaultKeyLister(origKL)
	SetDefaultKeyLister(&genKL3{global: []model.PublicKey{{ID: 21, Algorithm: "ssh-ed25519", KeyData: "D", Comment: "z"}}, account: nil})

	out, err := GenerateSelectiveKeysContent(7, 0, nil, true)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if strings.Contains(out, "# Keymaster Managed Keys") || strings.Contains(out, SystemKeyRestrictions) {
		t.Fatalf("did not expect system key header or restrictions when removeSystemKey=true: %q", out)
	}
	if !strings.Contains(out, "ssh-ed25519 D") {
		t.Fatalf("expected user key present, got %q", out)
	}
}
