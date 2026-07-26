// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package ssh

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// Secret holds the SSH identity used to reach a target.
type Secret struct {
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase,omitempty"`
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

func (c *Connector) NewSecretFromValues(values map[string]string) (connector.Secret, error) {
	return &Secret{values["private_key"], values["passphrase"]}, nil
}

// secretOf narrows the deploy data's secret to this connector's type.
func secretOf(deployData connector.DeployData) (*Secret, error) {
	secret, ok := deployData.Secret.(*Secret)
	if !ok {
		return nil, i18n.NewError("errors.connector.secret_type", fmt.Sprintf("%T", deployData.Secret))
	}
	return secret, nil
}
