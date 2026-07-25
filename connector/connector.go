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
	Deploy(ctx context.Context, deployData DeployData, connectionData ConnectionData, userRequester UserRequester, progress chan<- Progress) (newCache string, err error)
	Verify(ctx context.Context, deployData DeployData, connectionData ConnectionData, userRequester UserRequester, progress chan<- Progress) (ok bool, newCache string, err error)
	VerifyOffline(ctx context.Context, deployData DeployData) (bool, error)
}

type ConnectionData struct {
	Username string
	Host     string
	Port     int
}

type DeployData struct {
	Records         []DeployRecord
	Secret          string
	Cache           string
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
	Status   string
}

type UserRequester interface {
	RequestText(promt fmt.Stringer) string
	RequestChoice(promts []fmt.Stringer) int
}
