// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/bootstrap"
	"github.com/toeirei/keymaster/internal/model"
)

type spySessionStore struct {
	deleted *string
	updated *string
}

func (s *spySessionStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return nil
}
func (s *spySessionStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return nil, nil
}
func (s *spySessionStore) DeleteBootstrapSession(id string) error {
	if s.deleted != nil {
		*s.deleted = id
	}
	return nil
}
func (s *spySessionStore) UpdateBootstrapSessionStatus(id string, status string) error {
	if s.updated != nil {
		*s.updated = status
	}
	return nil
}
func (s *spySessionStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}
func (s *spySessionStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}

type spyDeployer struct {
	err          error
	used, closed bool
}

func (d *spyDeployer) DeployAuthorizedKeys(content string) error { d.used = true; return d.err }
func (d *spyDeployer) Close()                                    { d.closed = true }

type spyAuditor struct {
	called          bool
	action, details string
}

func (s *spyAuditor) LogAction(action, details string) error {
	s.called = true
	s.action = action
	s.details = details
	return nil
}

func TestGenerateKeysContentFailure_CleansUp(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "gk", Hostname: "h"}
	var deleted int
	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 501, nil },
		DeleteAccount:       func(id int) error { deleted = id; return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "", errors.New("gen fail") },
	}
	_, err := PerformBootstrapDeployment(ctx, params, deps)
	if err == nil {
		t.Fatalf("expected generate keys error")
	}
	if deleted != 501 {
		t.Fatalf("expected account cleanup, got %d", deleted)
	}
}

func TestNewBootstrapDeployerCreationFailure_CleansUp(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "nd", Hostname: "h"}
	var deleted int
	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 601, nil },
		DeleteAccount:       func(id int) error { deleted = id; return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		NewBootstrapDeployer: func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
			return nil, errors.New("create fail")
		},
	}
	_, err := PerformBootstrapDeployment(ctx, params, deps)
	if err == nil {
		t.Fatalf("expected deployer creation error")
	}
	if deleted != 601 {
		t.Fatalf("expected account cleanup, got %d", deleted)
	}
}

func TestNoDeployerPath_UpdatesSessionToFailed(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "ndp", Hostname: "h", SelectedKeyIDs: []int{1}, SessionID: "sess-no"}
	updated := ""
	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 701, nil },
		DeleteAccount:       func(id int) error { return nil },
		AssignKey:           func(k, a int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		SessionStore:        &spySessionStore{updated: &updated},
	}
	res, err := PerformBootstrapDeployment(ctx, params, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RemoteDeployed {
		t.Fatalf("expected remote not deployed")
	}
	if updated != string(bootstrap.StatusFailed) {
		t.Fatalf("expected session status failed, got %q", updated)
	}
}

func TestAuditorVsLogAudit_Priority(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "au", Hostname: "h"}

	// Auditor provided should be used
	sa := &spyAuditor{}
	depsA := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 801, nil },
		DeleteAccount:       func(id int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		NewBootstrapDeployer: func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
			return &spyDeployer{}, nil
		},
		Auditor: sa,
	}
	_, err := PerformBootstrapDeployment(ctx, params, depsA)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !sa.called {
		t.Fatalf("expected Auditor.LogAction to be called")
	}

	// If Auditor nil but LogAudit provided, LogAudit should be used
	called := false
	depsL := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 802, nil },
		DeleteAccount:       func(id int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		NewBootstrapDeployer: func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
			return &spyDeployer{}, nil
		},
		LogAudit: func(e BootstrapAuditEvent) error { called = true; return nil },
	}
	_, err = PerformBootstrapDeployment(ctx, params, depsL)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !called {
		t.Fatalf("expected LogAudit to be called")
	}
}

func TestSystemKeyWarningRecorded(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "sk", Hostname: "h"}
	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 901, nil },
		DeleteAccount:       func(id int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		NewBootstrapDeployer: func(hostname, username, privateKey, expectedHostKey string) (BootstrapDeployer, error) {
			return &spyDeployer{}, nil
		},
		GetActiveSystemKey: func() (*model.SystemKey, error) { return &model.SystemKey{Serial: 9, PublicKey: "pk"}, nil },
	}
	res, err := PerformBootstrapDeployment(ctx, params, deps)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "system key serial available: 9") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected system key warning in result")
	}
}
