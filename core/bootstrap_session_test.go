// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"errors"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
)

// failingStore simulates a storage backend that fails on Save.
type failingStore struct{ err error }

func (f *failingStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return f.err
}
func (f *failingStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return nil, nil
}
func (f *failingStore) DeleteBootstrapSession(id string) error                      { return nil }
func (f *failingStore) UpdateBootstrapSessionStatus(id string, status string) error { return nil }
func (f *failingStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}
func (f *failingStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}

// delFailStore simulates a backend that fails on Delete.
type delFailStore struct{ err error }

func (d *delFailStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return nil
}
func (d *delFailStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return nil, nil
}
func (d *delFailStore) DeleteBootstrapSession(id string) error                      { return d.err }
func (d *delFailStore) UpdateBootstrapSessionStatus(id string, status string) error { return nil }
func (d *delFailStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}
func (d *delFailStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}

func TestNewSession_NilStore(t *testing.T) {
	s, err := NewSession(nil, "alice", "host.local", "lbl", "tags")
	if err != nil {
		t.Fatalf("NewSession(nil) returned error: %v", err)
	}
	if s == nil {
		t.Fatalf("expected session, got nil")
	}
	if s.TempKeyPair == nil || len(s.TempKeyPair.GetPrivateKeyPEM()) == 0 {
		t.Fatalf("expected temporary private key to be present")
	}
}

func TestNewSession_WithStore_Success(t *testing.T) {
	// use existing spySessionStore which returns nil for SaveBootstrapSession
	store := &spySessionStore{}
	s, err := NewSession(store, "bob", "example", "lab", "t")
	if err != nil {
		t.Fatalf("NewSession returned error: %v", err)
	}
	if s == nil {
		t.Fatalf("expected session, got nil")
	}
	if s.TempKeyPair == nil || len(s.TempKeyPair.GetPrivateKeyPEM()) == 0 {
		t.Fatalf("expected temporary private key to be present")
	}
}

func TestNewSession_WithStore_SaveFails(t *testing.T) {
	store := &failingStore{err: errors.New("fail")}
	s, err := NewSession(store, "c", "h", "l", "")
	if err == nil {
		t.Fatalf("expected error when SaveBootstrapSession fails")
	}
	if s != nil {
		t.Fatalf("expected nil session on save failure")
	}
}

func TestCancelBootstrapSession_WithStore(t *testing.T) {
	deleted := ""
	store := &spySessionStore{deleted: &deleted}
	if err := CancelBootstrapSession(store, "sid123"); err != nil {
		t.Fatalf("CancelBootstrapSession returned error: %v", err)
	}
	if deleted != "sid123" {
		t.Fatalf("expected DeleteBootstrapSession called with sid123, got %q", deleted)
	}
}

func TestCancelBootstrapSession_DeleteFails(t *testing.T) {
	store := &delFailStore{err: errors.New("nope")}
	if err := CancelBootstrapSession(store, "x"); err == nil {
		t.Fatalf("expected error when DeleteBootstrapSession fails")
	}
}
