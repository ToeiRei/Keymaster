// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package mock

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/toeirei/keymaster/connector"
	"github.com/toeirei/keymaster/ui/i18n"
)

// Secret decides whether the simulated connection succeeds.
type Secret struct {
	Succeed bool `json:"succeed"`
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
		{"succeed", i18n.Text("connector.mock.secret.succeed"), strconv.FormatBool(s.Succeed), false, false},
	}
}

func (s *Secret) String() string { return "[SECRET]" }

func (c *Connector) SecretFields() []connector.SecretField {
	return (&Secret{}).Fields()
}

func (c *Connector) ParseSecret(raw string) (connector.Secret, error) {
	secret := &Secret{}
	if strings.TrimSpace(raw) == "" {
		return secret, nil
	}
	if err := json.Unmarshal([]byte(raw), secret); err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	return secret, nil
}

func (c *Connector) ParseSecretFromValues(values map[string]string) (connector.Secret, error) {
	// An empty field means "not configured", which for the mock is a failure.
	raw := strings.TrimSpace(values["succeed"])
	if raw == "" {
		return &Secret{}, nil
	}
	succeed, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	return &Secret{succeed}, nil
}

// secretOf narrows the deploy data's secret to this connector's type.
func secretOf(deployData connector.DeployData) (*Secret, error) {
	secret, ok := deployData.Secret.(*Secret)
	if !ok {
		return nil, i18n.NewError("errors.connector.secret_type", fmt.Sprintf("%T", deployData.Secret))
	}
	return secret, nil
}
