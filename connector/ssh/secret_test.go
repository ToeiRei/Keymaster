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

	blank, err := c.NewSecret("")
	if err != nil {
		t.Fatalf("NewSecret(\"\"): %v", err)
	}
	fields := blank.Fields()
	if len(fields) != 2 {
		t.Fatalf("expected 2 blank fields, got %d", len(fields))
	}
	for _, field := range fields {
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

	fromValues, err := c.NewSecretFromValues(map[string]string{
		"private_key": "-----BEGIN KEY-----\nabc\n-----END KEY-----\n",
		"passphrase":  "hunter2",
	})
	if err != nil {
		t.Fatalf("NewSecretFromValues: %v", err)
	}
	raw, err := fromValues.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	parsed, err := c.NewSecret(raw)
	if err != nil {
		t.Fatalf("NewSecret(%q): %v", raw, err)
	}
	secret := parsed.(*Secret)
	if secret.PrivateKey != "-----BEGIN KEY-----\nabc\n-----END KEY-----\n" {
		t.Fatalf("private key did not survive the round trip: %q", secret.PrivateKey)
	}
	if secret.Passphrase != "hunter2" {
		t.Fatalf("passphrase did not survive the round trip: %q", secret.Passphrase)
	}
}

func TestNewSecretFromValues_MissingKeysAreEmpty(t *testing.T) {
	c := &Connector{}

	parsed, err := c.NewSecretFromValues(map[string]string{"private_key": "pem"})
	if err != nil {
		t.Fatalf("NewSecretFromValues: %v", err)
	}
	if secret := parsed.(*Secret); secret.PrivateKey != "pem" || secret.Passphrase != "" {
		t.Fatalf("unexpected secret: %+v", *secret)
	}
}

func TestSecret_Redacts(t *testing.T) {
	secret := &Secret{"-----BEGIN RSA PRIVATE KEY-----\nsupersecret\n", "hunter2"}

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
	if got := (&Secret{"pem", "hunter2"}).passphraseBytes(); string(got) != "hunter2" {
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
