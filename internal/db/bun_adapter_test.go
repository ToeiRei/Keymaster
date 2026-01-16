// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"
	"time"
)

func TestAccountHelpers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		// Add an account
		id, err := AddAccountBun(bdb, "deploy", "example.com", "web-1", "env:prod")
		if err != nil {
			t.Fatalf("AddAccountBun failed: %v", err)
		}

		acc, err := GetAccountByIDBun(bdb, id)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if acc == nil || acc.Username != "deploy" || acc.Hostname != "example.com" {
			t.Fatalf("unexpected account: %+v", acc)
		}

		// Toggle status
		orig := acc.IsActive
		newStatus, err := ToggleAccountStatusBun(bdb, id)
		if err != nil {
			t.Fatalf("ToggleAccountStatusBun failed: %v", err)
		}
		if newStatus == orig {
			t.Fatalf("status did not change: before=%v after=%v", orig, newStatus)
		}

		// Update serial
		if err := UpdateAccountSerialBun(bdb, id, 42); err != nil {
			t.Fatalf("UpdateAccountSerialBun failed: %v", err)
		}
		acc2, err := GetAccountByIDBun(bdb, id)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if acc2.Serial != 42 {
			t.Fatalf("expected serial 42, got %d", acc2.Serial)
		}

		// Mark dirty and verify
		if err := UpdateAccountIsDirtyBun(bdb, id, true); err != nil {
			t.Fatalf("UpdateAccountIsDirtyBun failed: %v", err)
		}
		acc3, err := GetAccountByIDBun(bdb, id)
		if err != nil {
			t.Fatalf("GetAccountByIDBun failed: %v", err)
		}
		if !acc3.IsDirty {
			t.Fatalf("expected account to be dirty")
		}
	})
}

func TestBootstrapSessionHelpers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		id := "sess-1"
		expires := time.Now().Add(1 * time.Hour)
		if err := SaveBootstrapSessionBun(bdb, id, "user", "host.local", "lab", "tag", "ssh-rsa AAA", expires, "active"); err != nil {
			t.Fatalf("SaveBootstrapSessionBun failed: %v", err)
		}

		bs, err := GetBootstrapSessionBun(bdb, id)
		if err != nil {
			t.Fatalf("GetBootstrapSessionBun failed: %v", err)
		}
		if bs == nil || bs.ID != id || bs.Username != "user" {
			t.Fatalf("unexpected bootstrap session: %+v", bs)
		}

		// Update status
		if err := UpdateBootstrapSessionStatusBun(bdb, id, "orphaned"); err != nil {
			t.Fatalf("UpdateBootstrapSessionStatusBun failed: %v", err)
		}
		bs2, err := GetBootstrapSessionBun(bdb, id)
		if err != nil {
			t.Fatalf("GetBootstrapSessionBun failed: %v", err)
		}
		if bs2.Status != "orphaned" {
			t.Fatalf("expected status orphaned, got %s", bs2.Status)
		}

		if err := DeleteBootstrapSessionBun(bdb, id); err != nil {
			t.Fatalf("DeleteBootstrapSessionBun failed: %v", err)
		}
		bs3, err := GetBootstrapSessionBun(bdb, id)
		if err != nil {
			t.Fatalf("GetBootstrapSessionBun failed after delete: %v", err)
		}
		if bs3 != nil {
			t.Fatalf("expected session deleted, still present: %+v", bs3)
		}
	})
}

func TestKnownHostHelpers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		host := "git.example"
		key := "ssh-ed25519 AAAA"
		if err := AddKnownHostKeyBun(bdb, host, key); err != nil {
			t.Fatalf("AddKnownHostKeyBun failed: %v", err)
		}
		got, err := GetKnownHostKeyBun(bdb, host)
		if err != nil {
			t.Fatalf("GetKnownHostKeyBun failed: %v", err)
		}
		if got != key {
			t.Fatalf("expected key '%s', got '%s'", key, got)
		}
	})
}

func TestSystemKeyHelpers(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.bun

		have, err := HasSystemKeysBun(bdb)
		if err != nil {
			t.Fatalf("HasSystemKeysBun failed: %v", err)
		}
		if have {
			t.Fatalf("expected no system keys initially")
		}

		serial, err := CreateSystemKeyBun(bdb, "pub", "priv")
		if err != nil {
			t.Fatalf("CreateSystemKeyBun failed: %v", err)
		}
		if serial <= 0 {
			t.Fatalf("unexpected serial: %d", serial)
		}

		have2, err := HasSystemKeysBun(bdb)
		if err != nil {
			t.Fatalf("HasSystemKeysBun failed: %v", err)
		}
		if !have2 {
			t.Fatalf("expected system keys present after create")
		}

		sk, err := GetSystemKeyBySerialBun(bdb, serial)
		if err != nil {
			t.Fatalf("GetSystemKeyBySerialBun failed: %v", err)
		}
		if sk == nil || sk.Serial != serial {
			t.Fatalf("unexpected system key: %+v", sk)
		}
	})
}
