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
	SecretFields() []SecretField
	// ParseSecret parses a stored secret and validates what it can. Connectors
	// whose secret needs credential material reject an empty raw; SecretFields is
	// the blank template a UI renders for a new account.
	ParseSecret(raw string) (Secret, error)
	// ParseSecretFromValues builds a secret from values keyed by SecretField.Key.
	// Missing keys are treated as empty; unknown keys are ignored.
	ParseSecretFromValues(values map[string]string) (Secret, error)
	// ParseCache parses a stored cache and validates what it can. An empty raw
	// yields a zero cache.
	ParseCache(raw string) (Cache, error)

	Deploy(ctx context.Context, deployData DeployData, connectionData ConnectionData, userRequester UserRequester, progress chan<- Progress) (newCache Cache, err error)
	Verify(ctx context.Context, deployData DeployData, connectionData ConnectionData, userRequester UserRequester, progress chan<- Progress) (ok bool, newCache Cache, err error)
	VerifyOffline(ctx context.Context, deployData DeployData) (bool, error)
}

type ConnectionData struct {
	User string
	Host string
	Port int
}

type DeployData struct {
	Records         []DeployRecord
	Secret          Secret
	Cache           Cache
	SystemKeySerial int
}

type DeployRecord struct {
	Algorithm string
	Data      string
	Comment   string
	IsGlobal  bool
	ExpiresAt time.Time
}

type Progress struct {
	Progress float64
	Status   fmt.Stringer
}

type UserRequester interface {
	RequestText(promt fmt.Stringer) string
	RequestChoice(promts []fmt.Stringer) int
}
