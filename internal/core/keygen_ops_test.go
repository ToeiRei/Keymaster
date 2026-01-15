// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"testing"
)

type fakeSystemKeyStore struct {
	createSerial int
	rotateSerial int
	createErr    error
	rotateErr    error
}

func (f *fakeSystemKeyStore) CreateSystemKey(publicKey, privateKey string) (int, error) {
	return f.createSerial, f.createErr
}
func (f *fakeSystemKeyStore) RotateSystemKey(publicKey, privateKey string) (int, error) {
	return f.rotateSerial, f.rotateErr
}

func TestCreateInitialSystemKey_NilStore(t *testing.T) {
	pub, serial, err := CreateInitialSystemKey(nil, "passphrase")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub == "" {
		t.Fatalf("expected public key string, got empty")
	}
	if serial != 0 {
		t.Fatalf("expected serial 0 when store nil, got %d", serial)
	}
}

func TestCreateInitialSystemKey_WithStore(t *testing.T) {
	f := &fakeSystemKeyStore{createSerial: 123}
	pub, serial, err := CreateInitialSystemKey(f, "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub == "" {
		t.Fatalf("expected public key")
	}
	if serial != 123 {
		t.Fatalf("expected serial 123, got %d", serial)
	}
}

func TestRotateSystemKey_WithStore(t *testing.T) {
	f := &fakeSystemKeyStore{rotateSerial: 555}
	serial, err := RotateSystemKey(f, "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if serial != 555 {
		t.Fatalf("expected rotated serial 555, got %d", serial)
	}
}

