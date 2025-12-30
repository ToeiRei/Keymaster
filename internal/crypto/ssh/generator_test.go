// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	xssh "golang.org/x/crypto/ssh"
)

func TestGenerateAndMarshalEd25519Key(t *testing.T) {
	pub, priv, err := GenerateAndMarshalEd25519Key("test-comment", "")
	if err != nil {
		t.Fatalf("GenerateAndMarshalEd25519Key failed: %v", err)
	}
	if pub == "" {
		t.Fatal("expected non-empty public key string")
	}
	if priv == "" {
		t.Fatal("expected non-empty private key string")
	}

	// Parse public key
	pk, comment, _, _, err := xssh.ParseAuthorizedKey([]byte(pub))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey failed: %v", err)
	}
	if comment != "test-comment" {
		t.Errorf("unexpected comment: got %q want %q", comment, "test-comment")
	}
	if pk == nil {
		t.Fatal("parsed public key is nil")
	}

	// Parse private key
	_, err = xssh.ParseRawPrivateKey([]byte(priv))
	if err != nil {
		t.Fatalf("ParseRawPrivateKey failed: %v", err)
	}
}

func TestMarshalEd25519PrivateKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey failed: %v", err)
	}

	block, err := MarshalEd25519PrivateKey(priv, "unit-test")
	if err != nil {
		t.Fatalf("MarshalEd25519PrivateKey failed: %v", err)
	}
	if block == nil {
		t.Fatal("expected non-nil PEM block")
	}

	pemBytes := pem.EncodeToMemory(block)
	if len(pemBytes) == 0 {
		t.Fatal("expected PEM bytes")
	}

	if _, err := xssh.ParseRawPrivateKey(pemBytes); err != nil {
		t.Fatalf("ParseRawPrivateKey failed on marshaled PEM: %v", err)
	}
}

func TestFingerprintSHA256(t *testing.T) {
	pub, _, err := GenerateAndMarshalEd25519Key("fp-test", "")
	if err != nil {
		t.Fatalf("GenerateAndMarshalEd25519Key failed: %v", err)
	}
	pk, _, _, _, err := xssh.ParseAuthorizedKey([]byte(pub))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey failed: %v", err)
	}

	fp := FingerprintSHA256(pk)
	if fp == "" {
		t.Fatal("expected non-empty fingerprint")
	}
}
