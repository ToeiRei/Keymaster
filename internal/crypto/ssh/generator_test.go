// Copyright (c) 2026 Keymaster Team
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

func TestGenerateAndMarshalEd25519Key_WithPassphrase(t *testing.T) {
	passphrase := "test-passphrase"
	pub, priv, err := GenerateAndMarshalEd25519Key("test-comment-encrypted", passphrase)
	if err != nil {
		t.Fatalf("GenerateAndMarshalEd25519Key with passphrase failed: %v", err)
	}
	if pub == "" {
		t.Fatal("expected non-empty public key string")
	}
	if priv == "" {
		t.Fatal("expected non-empty private key string")
	}

	// 1. Check that parsing without a passphrase fails
	_, err = xssh.ParseRawPrivateKey([]byte(priv))
	if err == nil {
		t.Fatal("expected error when parsing encrypted key without passphrase, but got nil")
	}
	if _, ok := err.(*xssh.PassphraseMissingError); !ok {
		t.Fatalf("expected PassphraseMissingError, got %T", err)
	}

	// 2. Check that parsing with the wrong passphrase fails
	_, err = xssh.ParseRawPrivateKeyWithPassphrase([]byte(priv), []byte("wrong-passphrase"))
	if err == nil {
		t.Fatal("expected error when parsing with wrong passphrase, but got nil")
	}

	// 3. Check that parsing with the correct passphrase succeeds
	pk, err := xssh.ParseRawPrivateKeyWithPassphrase([]byte(priv), []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to parse private key with correct passphrase: %v", err)
	}
	if pk == nil {
		t.Fatal("parsed private key is nil")
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

