// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

import (
	"context"
	"fmt"
	"time"
)

type Connector interface {
	// OpenConnection tries to open a connector specific connection for remote interactions.
	OpenConnection(
		ctx context.Context,
		secret Secret,
		cache Cache,
		user string,
		host string,
		port int,
		userRequester UserRequester,
	) (Connection, error)

	// SecretFields provides fields from [Secret.Fields] for client implementations,
	// to prompt the user for a secret's configuration.
	SecretFields() []SecretField

	// ParseSecretFromFields builds a secret from fieldValues keyed by [SecretField.Key].
	// The input is also validated to ensure the user provided a working configuration.
	ParseSecretFromFields(fieldValues map[string]string) (Secret, error)

	// ParseSecret parses a serialized secret from [Secret.Serialize] back into [Secret].
	ParseSecret(raw string) (Secret, error)

	// ParseCache parses a serialized cache from [Cache.Serialize] back into [Cache].
	ParseCache(raw string) (Cache, error)

	// VerifyOffline verifies the provided deployment matches the last cached deployment,
	// without requiring a [Connection].
	VerifyOffline(
		ctx context.Context,
		cache Cache,
		deployment Deployment,
	) (ok bool, err error)
}

// Deployment is the state a target should be in: the records to install and the
// secret whose public material identifies Keymaster on that target. A deployment
// is normally keyed to the secret its connection authenticated with; keying it to
// a different one is what makes a secret rotation an ordinary deploy.
type Deployment struct {
	Secret  Secret
	Records []DeployRecord
}

type DeployRecord struct {
	Algorithm string
	Data      string
	ExpiresAt time.Time

	// non critical metadata:

	Comment  string
	IsGlobal bool
}

type UserRequester interface {
	RequestText(prompt fmt.Stringer) string
	RequestChoice(prompts []fmt.Stringer) int
}
