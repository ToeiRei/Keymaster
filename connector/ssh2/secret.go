// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh2

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

// secret holds the SSH identity used to reach a target.
type secret struct {
	PrivateKey     string `json:"private_key"`
	Passphrase     string `json:"passphrase,omitempty"`
	OmitPassphrase bool   `json:"omit_passphrase,omitempty"`
	PublicKey      string `json:"public_key"` // derived from the private key once, when the secret is parsed from user input
}

// implements [connector.Secret]
var _ connector.Secret = (*secret)(nil)

// implements [fmt.Stringer] for redaction
var _ fmt.Stringer = (*secret)(nil)

// implements [fmt.Formatter] for redaction
var _ fmt.Formatter = (*secret)(nil)

func (s *secret) Serialize() (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", i18n.WrapError(err, "errors.connector.serialize_secret")
	}
	return string(raw), nil
}

func (s *secret) Fields() []connector.SecretField {
	return []connector.SecretField{
		{"private_key", s.PrivateKey, i18n.Text("connector.ssh.secret.private_key"), true, false},
		{"passphrase", s.Passphrase, i18n.Text("connector.ssh.secret.passphrase"), false, true},
		{"omit_passphrase", strconv.FormatBool(s.OmitPassphrase), i18n.Text("connector.ssh.secret.omit_passphrase"), false, false},
	}
}

func (s *secret) String() string { return "[SECRET]" }

func (s *secret) Format(f fmt.State, c rune) {
	if _, err := io.WriteString(f, "[SECRET]"); err != nil {
		_ = err // intentionally ignore write error when formatting secrets for logs
	}
}

// passphraseBytes returns nil rather than an empty slice for an unset
// passphrase, so the deployer keeps taking its no-passphrase branch.
func (s *secret) passphraseBytes() []byte {
	if s.Passphrase == "" {
		return nil
	}
	return []byte(s.Passphrase)
}

// publicKey returns the public key stored alongside the private one. It is
// never re-derived here: authorized_keys, and therefore the hash drift
// detection compares, has to stay byte-stable, so the key that was written
// when the secret was stored is the only one that may reach a target. A secret
// stored without one has to be saved again.
func (s *secret) publicKey() (string, error) {
	if s.PublicKey == "" {
		return "", i18n.NewError("errors.connector.missing_public_key")
	}
	return s.PublicKey, nil
}

// publicKeyFromPrivateKey parses the PEM-encoded private key, decrypting it with
// passphrase when it is encrypted, and returns its public key in
// authorized_keys wire format (without a trailing newline). It also reports
// whether the key was encrypted, so a caller can tell a passphrase that was
// needed from one that was beside the point.
func publicKeyFromPrivateKey(privateKey, passphrase string) (string, bool, error) {
	encrypted := false
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if _, ok := errors.AsType[*ssh.PassphraseMissingError](err); ok {
		encrypted = true
		if passphrase == "" {
			return "", encrypted, i18n.NewError("errors.connector.passphrase_required")
		}
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	}
	if err != nil {
		return "", encrypted, i18n.WrapError(err, "errors.connector.parse_private_key")
	}
	// MarshalAuthorizedKey appends a single trailing newline; strip just that
	// so the key can be embedded on a line of its own.
	pubKey := ssh.MarshalAuthorizedKey(signer.PublicKey())
	return strings.TrimSuffix(string(pubKey), "\n"), encrypted, nil
}

// validate checks a secret as far as it can be checked cheaply: the private key
// has to be a key, and any public key stored beside it has to be one too.
// Whether the passphrase really opens the private key is settled once, in
// ParseSecretFromFields -- ParseSecret runs for every account of every listing,
// so it must not pay for a key derivation.
func (s *secret) validate() error {
	if strings.TrimSpace(s.PrivateKey) == "" {
		return i18n.NewError("errors.connector.missing_private_key")
	}
	// An encrypted key reports itself before any key derivation runs, so a
	// structural check stays affordable even on the read path.
	if _, err := ssh.ParseRawPrivateKey([]byte(s.PrivateKey)); err != nil {
		if _, encrypted := errors.AsType[*ssh.PassphraseMissingError](err); !encrypted {
			return i18n.WrapError(err, "errors.connector.parse_private_key")
		}
	}
	if s.PublicKey != "" {
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(s.PublicKey)); err != nil {
			return i18n.WrapError(err, "errors.connector.parse_public_key")
		}
	}
	return nil
}

func (c *Connector) SecretFields() []connector.SecretField {
	return (&secret{}).Fields()
}

func (c *Connector) ParseSecret(raw string) (connector.Secret, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, i18n.NewError("errors.connector.missing_secret")
	}
	parsed := &secret{}
	if err := json.Unmarshal([]byte(raw), parsed); err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	if err := parsed.validate(); err != nil {
		return nil, err
	}
	return parsed, nil
}

// ParseSecretFromFields validates the submitted values, derives the public key
// from the private key, using the passphrase when the key is encrypted, and
// drops the passphrase again when the caller asked for it not to be stored.
// This is the only place the connector derives the public key; every later
// reader takes the stored one, so what reaches authorized_keys cannot drift.
func (c *Connector) ParseSecretFromFields(fieldValues map[string]string) (connector.Secret, error) {
	privateKey := fieldValues["private_key"]
	passphrase := fieldValues["passphrase"]

	omitPassphrase := false
	if raw := strings.TrimSpace(fieldValues["omit_passphrase"]); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, i18n.WrapError(err, "errors.connector.parse_secret")
		}
		omitPassphrase = parsed
	}

	if strings.TrimSpace(privateKey) == "" {
		return nil, i18n.NewError("errors.connector.missing_private_key")
	}

	publicKey, encrypted, err := publicKeyFromPrivateKey(privateKey, passphrase)
	if err != nil {
		return nil, err
	}
	// The deployer only ever reads the passphrase for an encrypted key, so
	// storing one here would persist material nothing goes on to use.
	if !encrypted && passphrase != "" {
		return nil, i18n.NewError("errors.connector.passphrase_unexpected")
	}

	if omitPassphrase {
		passphrase = ""
	}
	parsed := &secret{privateKey, passphrase, omitPassphrase, publicKey}
	if err := parsed.validate(); err != nil {
		return nil, err
	}
	return parsed, nil
}

// narrowSecret narrows a secret to this connector's type.
func narrowSecret(s connector.Secret) (*secret, error) {
	parsed, ok := s.(*secret)
	if !ok {
		return nil, i18n.NewError("errors.connector.secret_type", fmt.Sprintf("%T", s))
	}
	return parsed, nil
}
