package db

import (
	"testing"
)

func TestStoresBun_UpdateHelpers(t *testing.T) {
	s, err := New("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create an account
	id, err := s.AddAccount("dave", "host-x", "label-x", "t1")
	if err != nil {
		t.Fatalf("AddAccount failed: %v", err)
	}

	// Update serial
	if err := s.UpdateAccountSerial(id, 99); err != nil {
		t.Fatalf("UpdateAccountSerial failed: %v", err)
	}
	acc, err := GetAccountByIDBun(s.(*BunStore).BunDB(), id)
	if err != nil {
		t.Fatalf("GetAccountByIDBun failed: %v", err)
	}
	if acc == nil || acc.Serial != 99 {
		t.Fatalf("expected serial 99, got %+v", acc)
	}

	// Set status inactive, then active
	if err := s.ToggleAccountStatus(id, false); err != nil {
		t.Fatalf("SetAccountActive first failed: %v", err)
	}
	a2, err := GetAccountByIDBun(s.(*BunStore).BunDB(), id)
	if err != nil {
		t.Fatalf("GetAccountByIDBun failed: %v", err)
	}
	if a2 == nil || a2.IsActive {
		t.Fatalf("expected account to be inactive after toggle, got %+v", a2)
	}
	if err := s.ToggleAccountStatus(id, true); err != nil {
		t.Fatalf("SetAccountActive second failed: %v", err)
	}
	a3, err := GetAccountByIDBun(s.(*BunStore).BunDB(), id)
	if err != nil {
		t.Fatalf("GetAccountByIDBun failed: %v", err)
	}
	if a3 == nil || !a3.IsActive {
		t.Fatalf("expected account to be active after second toggle, got %+v", a3)
	}

	// Update label, hostname, tags
	if err := s.UpdateAccountLabel(id, "new-label"); err != nil {
		t.Fatalf("UpdateAccountLabel failed: %v", err)
	}
	if err := s.UpdateAccountHostname(id, "new-host"); err != nil {
		t.Fatalf("UpdateAccountHostname failed: %v", err)
	}
	if err := s.UpdateAccountTags(id, "a=b"); err != nil {
		t.Fatalf("UpdateAccountTags failed: %v", err)
	}
	a4, err := GetAccountByIDBun(s.(*BunStore).BunDB(), id)
	if err != nil {
		t.Fatalf("GetAccountByIDBun failed: %v", err)
	}
	if a4 == nil || a4.Label != "new-label" || a4.Hostname != "new-host" || a4.Tags != "a=b" {
		t.Fatalf("expected updated fields, got %+v", a4)
	}

	// Update is_dirty flag
	if err := s.UpdateAccountIsDirty(id, true); err != nil {
		t.Fatalf("UpdateAccountIsDirty failed: %v", err)
	}
	a5, err := GetAccountByIDBun(s.(*BunStore).BunDB(), id)
	if err != nil {
		t.Fatalf("GetAccountByIDBun failed: %v", err)
	}
	if a5 == nil || !a5.IsDirty {
		t.Fatalf("expected IsDirty true, got %+v", a5)
	}
}
