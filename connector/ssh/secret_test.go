// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"fmt"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/connector"
)

func TestNewSecret_RoundTrip(t *testing.T) {
	c := &Connector{}

	// SecretFields is the blank template for a new account; ParseSecret is for
	// stored JSON only and rejects an empty string.
	if _, err := c.ParseSecret(""); err == nil {
		t.Fatal("expected ParseSecret to reject an empty raw secret")
	}

	fields := c.SecretFields()
	if len(fields) != 3 {
		t.Fatalf("expected 3 blank fields, got %d", len(fields))
	}
	for _, field := range fields[:2] {
		if field.Value != "" {
			t.Fatalf("expected blank field %q to have no value, got %q", field.Key, field.Value)
		}
	}
	if !fields[0].Multiline || fields[0].Masked {
		t.Fatal("expected the private key field to be multiline and unmasked")
	}
	if fields[1].Multiline || !fields[1].Masked {
		t.Fatal("expected the passphrase field to be single-line and masked")
	}
	if fields[2].Key != "omit_passphrase" || fields[2].Value != "false" {
		t.Fatalf("expected a blank secret to default to storing the passphrase, got %+v", fields[2])
	}
	for _, field := range fields {
		if field.Key == "public_key" {
			t.Fatal("expected the derived public key to stay out of the editable fields")
		}
	}

	privateKey, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}

	fromValues, err := c.NewSecretFromValues(map[string]string{
		"private_key": privateKey,
		"passphrase":  "hunter2",
	})
	if err != nil {
		t.Fatalf("NewSecretFromValues: %v", err)
	}
	raw, err := fromValues.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	parsed, err := c.ParseSecret(raw)
	if err != nil {
		t.Fatalf("ParseSecret(%q): %v", raw, err)
	}
	secret := parsed.(*Secret)
	if secret.PrivateKey != privateKey {
		t.Fatalf("private key did not survive the round trip: %q", secret.PrivateKey)
	}
	if secret.Passphrase != "hunter2" {
		t.Fatalf("passphrase did not survive the round trip: %q", secret.Passphrase)
	}

	wantPublicKey, err := publicKeyFromPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}
	if secret.PublicKey != wantPublicKey {
		t.Fatalf("public key did not survive the round trip: got %q, want %q", secret.PublicKey, wantPublicKey)
	}
}

func TestNewSecretFromValues_Validates(t *testing.T) {
	c := &Connector{}
	privateKey, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}
	encryptedKey, err := generateEncryptedTestPrivateKeyPEM("hunter2")
	if err != nil {
		t.Fatalf("generate encrypted test private key: %v", err)
	}

	for name, values := range map[string]map[string]string{
		"no values":              {},
		"blank private key":      {"private_key": "  \n"},
		"unparsable private key": {"private_key": "pem"},
		"missing passphrase":     {"private_key": encryptedKey},
		"wrong passphrase":       {"private_key": encryptedKey, "passphrase": "hunter3"},
		"unparsable omit option": {"private_key": privateKey, "omit_passphrase": "maybe"},
	} {
		if _, err := c.NewSecretFromValues(values); err == nil {
			t.Fatalf("expected NewSecretFromValues to reject %s", name)
		}
	}
}

func TestNewSecretFromValues_DerivesPublicKeyFromEncryptedKey(t *testing.T) {
	c := &Connector{}
	encryptedKey, err := generateEncryptedTestPrivateKeyPEM("hunter2")
	if err != nil {
		t.Fatalf("generate encrypted test private key: %v", err)
	}
	wantPublicKey, err := publicKeyFromPrivateKey(encryptedKey, "hunter2")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}

	parsed, err := c.NewSecretFromValues(map[string]string{
		"private_key": encryptedKey,
		"passphrase":  "hunter2",
	})
	if err != nil {
		t.Fatalf("NewSecretFromValues: %v", err)
	}
	secret := parsed.(*Secret)
	if secret.Passphrase != "hunter2" {
		t.Fatalf("expected the passphrase to be kept, got %q", secret.Passphrase)
	}
	if secret.PublicKey != wantPublicKey {
		t.Fatalf("unexpected public key: got %q, want %q", secret.PublicKey, wantPublicKey)
	}
}

func TestNewSecretFromValues_OmitsPassphraseButKeepsPublicKey(t *testing.T) {
	c := &Connector{}
	encryptedKey, err := generateEncryptedTestPrivateKeyPEM("hunter2")
	if err != nil {
		t.Fatalf("generate encrypted test private key: %v", err)
	}
	wantPublicKey, err := publicKeyFromPrivateKey(encryptedKey, "hunter2")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}

	parsed, err := c.NewSecretFromValues(map[string]string{
		"private_key":     encryptedKey,
		"passphrase":      "hunter2",
		"omit_passphrase": "true",
	})
	if err != nil {
		t.Fatalf("NewSecretFromValues: %v", err)
	}
	secret := parsed.(*Secret)
	if secret.Passphrase != "" {
		t.Fatalf("expected the passphrase to be dropped, got %q", secret.Passphrase)
	}
	if secret.PublicKey != wantPublicKey {
		t.Fatalf("unexpected public key: got %q, want %q", secret.PublicKey, wantPublicKey)
	}

	raw, err := secret.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if strings.Contains(raw, "hunter2") {
		t.Fatalf("expected the omitted passphrase to stay out of the serialized secret: %q", raw)
	}

	// The stored public key spares the deploy path from decrypting the key it can
	// no longer unlock.
	if publicKey, err := secret.publicKey(); err != nil || publicKey != wantPublicKey {
		t.Fatalf("expected the stored public key to be reused, got %q err=%v", publicKey, err)
	}

	// An edit form re-offers the choice the secret was stored with.
	fields := secret.Fields()
	if got := fields[len(fields)-1]; got.Key != "omit_passphrase" || got.Value != "true" {
		t.Fatalf("expected the omit option to stay on, got %+v", got)
	}
}

func TestSecretPublicKey_FallsBackToPrivateKey(t *testing.T) {
	privateKey, err := generateTestPrivateKeyPEM()
	if err != nil {
		t.Fatalf("generate test private key: %v", err)
	}
	want, err := publicKeyFromPrivateKey(privateKey, "")
	if err != nil {
		t.Fatalf("publicKeyFromPrivateKey: %v", err)
	}

	// Secrets stored before the public key field existed still render.
	got, err := (&Secret{PrivateKey: privateKey}).publicKey()
	if err != nil {
		t.Fatalf("publicKey: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected public key: got %q, want %q", got, want)
	}
}

func TestSecret_Redacts(t *testing.T) {
	secret := &Secret{"-----BEGIN RSA PRIVATE KEY-----\nsupersecret\n", "hunter2", false, "ssh-rsa AAAApublic"}

	for _, formatted := range []string{
		secret.String(),
		fmt.Sprintf("%v", secret),
		fmt.Sprintf("%#v", secret),
		fmt.Sprintf("%s", secret),
	} {
		if strings.Contains(formatted, "supersecret") || strings.Contains(formatted, "hunter2") {
			t.Fatalf("secret leaked through formatting: %q", formatted)
		}
	}
}

func TestSecret_PassphraseBytesNilWhenUnset(t *testing.T) {
	if got := (&Secret{PrivateKey: "pem"}).passphraseBytes(); got != nil {
		t.Fatalf("expected nil passphrase bytes when unset, got %v", got)
	}
	if got := (&Secret{"pem", "hunter2", false, ""}).passphraseBytes(); string(got) != "hunter2" {
		t.Fatalf("unexpected passphrase bytes: %q", got)
	}
}

func TestSecretOf_RejectsForeignSecret(t *testing.T) {
	if _, err := secretOf(connector.DeployData{Secret: foreignSecret{}}); err == nil {
		t.Fatal("expected secretOf to reject a secret from another connector")
	}
}

type foreignSecret struct{}

func (foreignSecret) Serialize() (string, error)      { return "", nil }
func (foreignSecret) Fields() []connector.SecretField { return nil }
func (foreignSecret) String() string                  { return "[SECRET]" }
