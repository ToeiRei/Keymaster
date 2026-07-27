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

// secret decides whether the simulated connection succeeds. Key is arbitrary
// identity material with no effect on reachability: it stands in for the public
// half a real connector installs on the target, so two working secrets can still
// be told apart and a rotation between them changes what [Connector.hash] sees.
type secret struct {
	Succeed bool   `json:"succeed"`
	Key     string `json:"key,omitempty"`
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
		{"succeed", strconv.FormatBool(s.Succeed), i18n.Text("connector.mock.secret.succeed"), false, false},
		{"key", s.Key, i18n.Text("connector.mock.secret.key"), false, false},
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

func (c *Connector) ParseSecretFromFields(fieldValues map[string]string) (connector.Secret, error) {
	key := strings.TrimSpace(fieldValues["key"])

	// An empty field means "not configured", which for the mock is a failure.
	raw := strings.TrimSpace(fieldValues["succeed"])
	if raw == "" {
		return &secret{false, key}, nil
	}
	succeed, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, i18n.WrapError(err, "errors.connector.parse_secret")
	}
	return &secret{succeed, key}, nil
}

// narrowSecret narrows a secret to this connector's type.
func narrowSecret(s connector.Secret) (*secret, error) {
	parsed, ok := s.(*secret)
	if !ok {
		return nil, i18n.NewError("errors.connector.secret_type", fmt.Sprintf("%T", s))
	}
	return parsed, nil
}
