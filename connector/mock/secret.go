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

// secret decides whether the simulated connection succeeds.
type secret struct {
	Succeed bool `json:"succeed"`
}

// implements [connector.Secret]
var _ connector.Secret = (*secret)(nil)

func (s *secret) Serialize() (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", i18n.WrapError(err, "errors.connector.serialize_secret")
	}
	return string(raw), nil
}

func (s *secret) Fields() []connector.SecretField {
	return []connector.SecretField{
		{"succeed", i18n.Text("connector.mock.secret.succeed"), strconv.FormatBool(s.Succeed), false, false},
	}
}

func (s *secret) String() string { return "[SECRET]" }

func (c *Connector) SecretFields() []connector.SecretField {
	return (&secret{}).Fields()
}

func (c *Connector) ParseSecret(raw string) (connector.Secret, error) {
	parsed := &secret{}
	if strings.TrimSpace(raw) == "" {
		return parsed, nil
	}
	if err := json.Unmarshal([]byte(raw), parsed); err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	return parsed, nil
}

func (c *Connector) ParseSecretFromValues(values map[string]string) (connector.Secret, error) {
	// An empty field means "not configured", which for the mock is a failure.
	raw := strings.TrimSpace(values["succeed"])
	if raw == "" {
		return &secret{}, nil
	}
	succeed, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	return &secret{succeed}, nil
}

// secretOf narrows the deploy data's secret to this connector's type.
func secretOf(deployData connector.DeployData) (*secret, error) {
	parsed, ok := deployData.Secret.(*secret)
	if !ok {
		return nil, i18n.NewError("errors.connector.secret_type", fmt.Sprintf("%T", deployData.Secret))
	}
	return parsed, nil
}
