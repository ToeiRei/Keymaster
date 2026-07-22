// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package connector

import (
	"context"
	"time"
)

type Connector interface {
	Deploy(ctx context.Context, deployData DeployData, connectionData ConnectionData) (chan Progress, error)
	Verify(ctx context.Context, deployData DeployData, connectionData ConnectionData) (chan Progress, error)
	VerifyOffline(ctx context.Context, deployData DeployData) (bool, error)
}

type ConnectionData struct {
	Username string
	Host     string
	Port     int
}

type DeployData struct {
	Records []DeployRecord
	Secret  string
	Cache   string
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
	Err      error
}
