// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
	"golang.org/x/crypto/ssh"
)

// Secret holds the SSH identity used to reach a target.
type Secret struct {
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase,omitempty"`
	PublicKey  string `json:"public_key"`
}

// *[Secret] implements [connector.Secret]
var _ connector.Secret = (*Secret)(nil)

func (s *Secret) Serialize() (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", i18n.WrapError(err, "errors.connector.serialize_secret")
	}
	return string(raw), nil
}

func (s *Secret) Fields() []connector.SecretField {
	return []connector.SecretField{
		{"private_key", i18n.Text("connector.ssh.secret.private_key"), s.PrivateKey, true, false},
		{"passphrase", i18n.Text("connector.ssh.secret.passphrase"), s.Passphrase, false, true},
		{"omit_passphrase", i18n.Text("connector.ssh.secret.omit_passphrase"), strconv.FormatBool(s.missingPassphrase()), false, false},
	}
}

// String redacts the secret so it cannot leak through fmt.Print*.
func (s *Secret) String() string { return "[SECRET]" }

// Format implements fmt.Formatter so `%v`, `%#v` and friends stay redacted.
func (s *Secret) Format(f fmt.State, c rune) {
	if _, err := io.WriteString(f, "[SECRET]"); err != nil {
		_ = err // intentionally ignore write error when formatting secrets for logs
	}
}

// passphraseBytes returns nil rather than an empty slice for an unset
// passphrase, so the deployer keeps taking its no-passphrase branch.
func (s *Secret) passphraseBytes() []byte {
	if s.Passphrase == "" {
		return nil
	}
	return []byte(s.Passphrase)
}

// missingPassphrase reports whether the secret withholds a passphrase its private
// key needs, so an edit form re-offers the choice it was stored with.
func (s *Secret) missingPassphrase() bool {
	if s.PrivateKey == "" || s.Passphrase != "" {
		return false
	}
	_, err := ssh.ParsePrivateKey([]byte(s.PrivateKey))
	var passphraseMissing *ssh.PassphraseMissingError
	return errors.As(err, &passphraseMissing)
}

// publicKey returns the stored public key, deriving it from the private key for
// secrets persisted before the field existed.
func (s *Secret) publicKey() (string, error) {
	if s.PublicKey != "" {
		return s.PublicKey, nil
	}
	return publicKeyFromPrivateKey(s.PrivateKey, s.Passphrase)
}

// publicKeyFromPrivateKey parses the PEM-encoded private key, decrypting it with
// passphrase when it is encrypted, and returns its public key in
// authorized_keys wire format (without a trailing newline).
func publicKeyFromPrivateKey(privateKey, passphrase string) (string, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if _, ok := errors.AsType[*ssh.PassphraseMissingError](err); ok {
		if passphrase == "" {
			return "", i18n.NewError("errors.connector.passphrase_required")
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	}
	if err != nil {
		return "", i18n.WrapError(err, "errors.connector.parse_private_key")
	}
	// MarshalAuthorizedKey appends a single trailing newline; strip just that
	// so the key can be embedded on a line of its own.
	pubKey := ssh.MarshalAuthorizedKey(signer.PublicKey())
	return strings.TrimSuffix(string(pubKey), "\n"), nil
}

func (c *Connector) NewSecret(raw string) (connector.Secret, error) {
	secret := &Secret{}
	if strings.TrimSpace(raw) == "" {
		return secret, nil
	}
	if err := json.Unmarshal([]byte(raw), secret); err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	return secret, nil
}

// NewSecretFromValues validates the submitted values, derives the public key
// from the private key — using the passphrase when the key is encrypted — and
// drops the passphrase again when the caller asked for it not to be stored.
func (c *Connector) NewSecretFromValues(values map[string]string) (connector.Secret, error) {
	privateKey := values["private_key"]
	passphrase := values["passphrase"]

	omitPassphrase := false
	if raw := strings.TrimSpace(values["omit_passphrase"]); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, i18n.WrapError(err, "errors.connector.parse_secret")
		}
		omitPassphrase = parsed
	}

	if strings.TrimSpace(privateKey) == "" {
		return nil, i18n.NewError("errors.connector.missing_private_key")
	}

	publicKey, err := publicKeyFromPrivateKey(privateKey, passphrase)
	if err != nil {
		return nil, err
	}

	if omitPassphrase {
		passphrase = ""
	}
	return &Secret{privateKey, passphrase, publicKey}, nil
}

// secretOf narrows the deploy data's secret to this connector's type.
func secretOf(deployData connector.DeployData) (*Secret, error) {
	secret, ok := deployData.Secret.(*Secret)
	if !ok {
		return nil, i18n.NewError("errors.connector.secret_type", fmt.Sprintf("%T", deployData.Secret))
	}
	return secret, nil
}
