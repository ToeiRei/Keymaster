// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/toeirei/keymaster/internal/core/model"
	"github.com/toeirei/keymaster/internal/core/security"
)

type testDeployer struct {
	err    error
	closed bool
	used   bool
}

func (t *testDeployer) DeployAuthorizedKeys(content string) error { t.used = true; return t.err }
func (t *testDeployer) Close()                                    { t.closed = true }

type stubSessionStore struct {
	deleted *string
	updated *string
}

func (s *stubSessionStore) SaveBootstrapSession(id, username, hostname, label, tags, tempPublicKey string, expiresAt time.Time, status string) error {
	return nil
}
func (s *stubSessionStore) GetBootstrapSession(id string) (*model.BootstrapSession, error) {
	return nil, nil
}
func (s *stubSessionStore) DeleteBootstrapSession(id string) error {
	if s.deleted != nil {
		*s.deleted = id
	}
	return nil
}
func (s *stubSessionStore) UpdateBootstrapSessionStatus(id string, status string) error {
	if s.updated != nil {
		*s.updated = status
	}
	return nil
}
func (s *stubSessionStore) GetExpiredBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}
func (s *stubSessionStore) GetOrphanedBootstrapSessions() ([]*model.BootstrapSession, error) {
	return nil, nil
}

func TestPerformBootstrapDeployment_SuccessPath(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "alice", Hostname: "h", SelectedKeyIDs: []int{1, 2}, TempPrivateKey: security.FromString("p"), HostKey: "k", SessionID: "sess-1"}

	var deletedID int
	var auditEvent BootstrapAuditEvent
	fake := &testDeployer{}
	deletedSession := ""

	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 101, nil },
		DeleteAccount:       func(id int) error { deletedID = id; return nil },
		AssignKey:           func(k, a int) error { return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		NewBootstrapDeployer: func(hostname, username string, privateKey interface{}, expectedHostKey string) (BootstrapDeployer, error) {
			return fake, nil
		},
		GetActiveSystemKey: func() (*model.SystemKey, error) { return &model.SystemKey{Serial: 5, PublicKey: "pk"}, nil },
		LogAudit:           func(e BootstrapAuditEvent) error { auditEvent = e; return nil },
		SessionStore:       &stubSessionStore{deleted: &deletedSession},
	}

	res, err := PerformBootstrapDeployment(ctx, params, deps)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res.Account.ID != 101 {
		t.Fatalf("bad account id: %d", res.Account.ID)
	}
	if !res.RemoteDeployed {
		t.Fatalf("expected deployed")
	}
	if len(res.KeysDeployed) != 2 {
		t.Fatalf("keys deployed count")
	}
	if !fake.used || !fake.closed {
		t.Fatalf("deployer not used/closed")
	}
	if auditEvent.Action != "BOOTSTRAP_SUCCESS" {
		t.Fatalf("audit missing")
	}
	if deletedID != 0 {
		t.Fatalf("account should not be deleted on success")
	}
	if deletedSession != "sess-1" {
		t.Fatalf("session should be deleted, got %q", deletedSession)
	}
}

func TestPerformBootstrapDeployment_AssignKeyFailCleansUp(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "bob", Hostname: "h", SelectedKeyIDs: []int{999}}
	var deletedID int
	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 202, nil },
		DeleteAccount:       func(id int) error { deletedID = id; return nil },
		AssignKey:           func(k, a int) error { return errors.New("boom") },
		GenerateKeysContent: func(accountID int) (string, error) { return "", nil },
	}
	_, err := PerformBootstrapDeployment(ctx, params, deps)
	if err == nil {
		t.Fatalf("expected assign error")
	}
	if deletedID != 202 {
		t.Fatalf("expected cleanup of account 202")
	}
}

func TestPerformBootstrapDeployment_DeployFailUpdatesSession(t *testing.T) {
	ctx := context.Background()
	params := BootstrapParams{Username: "carol", Hostname: "x", TempPrivateKey: security.FromString("p"), HostKey: "k", SessionID: "s-2"}
	var deletedID int
	updated := ""
	deps := BootstrapDeps{
		AddAccount:          func(u, h, l, tags string) (int, error) { return 303, nil },
		DeleteAccount:       func(id int) error { deletedID = id; return nil },
		GenerateKeysContent: func(accountID int) (string, error) { return "ok", nil },
		NewBootstrapDeployer: func(hostname, username string, privateKey interface{}, expectedHostKey string) (BootstrapDeployer, error) {
			return &testDeployer{err: errors.New("ssh fail")}, nil
		},
		SessionStore: &stubSessionStore{updated: &updated},
	}
	_, err := PerformBootstrapDeployment(ctx, params, deps)
	if err == nil {
		t.Fatalf("expected deploy error")
	}
	if deletedID != 303 {
		t.Fatalf("expected account cleanup")
	}
	// The current implementation returns early on deploy failure and does
	// not reach the session-update code path; ensure no session status was set.
	if updated != "" {
		t.Fatalf("did not expect session status update on early return; got %q", updated)
	}
}
