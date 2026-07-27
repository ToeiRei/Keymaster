// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bunrewrite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/client/bunrewrite/db"
)

// The mock connector authenticates on the succeed field and fingerprints the key
// field, so a swap between two working secrets is observable, and a swap to an
// unusable one fails where a real connector would: at the dial.
func mockSecret(succeed bool, key string) map[string]string {
	return map[string]string{"succeed": strconv.FormatBool(succeed), "key": key}
}

func newSecretTestClient(t *testing.T) (*Client, context.Context) {
	t.Helper()
	ctx := context.Background()
	c, err := NewDefaultBunClient(log.Default(), "test.keymaster.secret_update")
	if err != nil {
		t.Fatalf("NewDefaultBunClient: %v", err)
	}
	t.Cleanup(func() { c.Close(ctx) })
	return c, ctx
}

func newMockAccount(t *testing.T, c *Client, ctx context.Context) client.Account {
	t.Helper()
	account, err := c.CreateAccount(ctx, "root", "example.com", 22, "mock", mockSecret(true, "original"))
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	return account
}

// secretColumns reads the two columns the two-phase swap lives in.
func secretColumns(t *testing.T, c *Client, ctx context.Context, id client.AccountId) (string, sql.NullString) {
	t.Helper()
	var model db.AccountModel
	if err := c.bun.NewSelect().
		Model(&model).
		Column("connector_secret", "connector_secret_rollback").
		Where("id = ?", int(id)).
		Scan(ctx); err != nil {
		t.Fatalf("read secret columns: %v", err)
	}
	return model.ConnectorSecret, model.ConnectorSecretRollback
}

// interruptSecretUpdate leaves the account in the state a crash between the database
// swap and the remote write produces: the new secret recorded as current, the
// previous one held aside, and the target untouched. This is the only way to reach
// that state, which is the point -- every path the client itself takes either
// completes the secret update or unwinds it.
func interruptSecretUpdate(t *testing.T, c *Client, ctx context.Context, id client.AccountId, secret map[string]string) {
	t.Helper()
	serialized, err := serializeSecret("mock", secret)
	if err != nil {
		t.Fatalf("serializeSecret: %v", err)
	}
	if err := c.beginSecretUpdate(ctx, id, serialized); err != nil {
		t.Fatalf("beginSecretUpdate: %v", err)
	}
}

// runUpdateSecret drains the progress channel the way a caller does.
func runUpdateSecret(c *Client, ctx context.Context, id client.AccountId, secret map[string]string) error {
	progress := make(chan client.UpdateSecretProgressAccount)
	var err error
	go func() {
		defer close(progress)
		err = c.UpdateAccountSecret(ctx, nil, progress, id, secret)
	}()
	for range progress {
	}
	return err
}

func hasAuditAction(t *testing.T, c *Client, ctx context.Context, action string) bool {
	t.Helper()
	logs, err := c.ListAuditLogs(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	for _, l := range logs {
		if l.Action == action {
			return true
		}
	}
	return false
}

func TestUpdateAccountSecret_ConfirmedSwapClearsTheRollback(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(true, "rotated")); err != nil {
		t.Fatalf("UpdateAccountSecret: %v", err)
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if !strings.Contains(secret, `"key":"rotated"`) {
		t.Fatalf("connector_secret was not replaced: %s", secret)
	}
	if rollback.Valid {
		t.Fatalf("expected the rollback to be cleared, got %q", rollback.String)
	}
	// Intent first, then outcome: a secret update can strand a target, so the request is
	// on record before anything is attempted.
	if !hasAuditAction(t, c, ctx, "account.update.secret.requested") {
		t.Fatal("expected an account.update.secret.requested audit entry")
	}
	if !hasAuditAction(t, c, ctx, "account.update.secret") {
		t.Fatal("expected an account.update.secret audit entry")
	}

	// The secret update ends with the target confirmed in the state it asked for, so the
	// account is clean rather than needing a follow-up deploy.
	reloaded, err := c.GetAccount(ctx, account.Id)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	dirty, err := c.IsAccountDirty(ctx, reloaded)
	if err != nil {
		t.Fatalf("IsAccountDirty: %v", err)
	}
	if dirty {
		t.Fatal("expected a confirmed secret update to leave the account clean")
	}
}

// A secret that cannot authenticate is installed, fails to confirm, and is undone
// over the still-open old connection. Because the restore is confirmed, the swap
// is unwound completely: no secret update is left outstanding.
func TestUpdateAccountSecret_UnconfirmedSwapIsUnwound(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	err := runUpdateSecret(c, ctx, account.Id, mockSecret(false, "unreachable"))
	if err == nil {
		t.Fatal("expected UpdateAccountSecret to fail when the new secret cannot authenticate")
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if rollback.Valid {
		t.Fatalf("expected a restored target to leave no secret update outstanding, got %q", rollback.String)
	}
	if !strings.Contains(secret, `"key":"original"`) {
		t.Fatalf("expected the previous secret to be current again, got %s", secret)
	}

	// And because nothing is outstanding, a corrected secret goes straight through.
	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(true, "corrected")); err != nil {
		t.Fatalf("retry after an unwound secret update: %v", err)
	}
	if secret, _ := secretColumns(t, c, ctx, account.Id); !strings.Contains(secret, `"key":"corrected"`) {
		t.Fatalf("expected the corrected secret to be current, got %s", secret)
	}
}

// Nothing authenticated, so nothing was written and the swap is unwound the same
// way.
func TestUpdateAccountSecret_UnreachableTargetIsUnwound(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account, err := c.CreateAccount(ctx, "root", "example.com", 22, "mock", mockSecret(false, "dead"))
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(false, "alsodead")); err == nil {
		t.Fatal("expected UpdateAccountSecret to fail when neither secret authenticates")
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if rollback.Valid {
		t.Fatalf("expected an untouched target to leave no secret update outstanding, got %q", rollback.String)
	}
	if !strings.Contains(secret, `"key":"dead"`) {
		t.Fatalf("expected the previous secret to be current again, got %s", secret)
	}
}

// Resuming after a crash: the target still holds the previous secret, the dial
// finds that out, and the secret update runs to completion.
func TestUpdateAccountSecret_ResumesAfterAnInterruption(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	target := mockSecret(true, "resumed")
	interruptSecretUpdate(t, c, ctx, account.Id, target)

	if err := runUpdateSecret(c, ctx, account.Id, target); err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if rollback.Valid {
		t.Fatalf("expected the resume to clear the rollback, got %q", rollback.String)
	}
	if !strings.Contains(secret, `"key":"resumed"`) {
		t.Fatalf("expected the resumed target to be current, got %s", secret)
	}
}

// Retargeting mid-secret update would discard the secret the target might be holding.
func TestUpdateAccountSecret_RefusesADifferentTargetWhilePending(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(true, "first"))

	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(true, "second")); err == nil {
		t.Fatal("expected a different target to be refused while a secret update is outstanding")
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if !strings.Contains(secret, `"key":"first"`) {
		t.Fatalf("expected the pending target to be left alone, got %s", secret)
	}
	if !strings.Contains(rollback.String, `"key":"original"`) {
		t.Fatalf("expected the original secret to be left alone, got %s", rollback.String)
	}
}

// An interrupted secret update whose target is unreachable stays outstanding: unlike a
// fresh attempt, it was already unclear which secret the target holds.
func TestUpdateAccountSecret_InterruptedAndUnreachableStaysPending(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account, err := c.CreateAccount(ctx, "root", "example.com", 22, "mock", mockSecret(false, "dead"))
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	target := mockSecret(false, "alsodead")
	interruptSecretUpdate(t, c, ctx, account.Id, target)

	if err := runUpdateSecret(c, ctx, account.Id, target); err == nil {
		t.Fatal("expected the resume to fail when neither secret authenticates")
	}

	_, rollback := secretColumns(t, c, ctx, account.Id)
	if !rollback.Valid {
		t.Fatal("expected an interrupted secret update to stay outstanding while unreachable")
	}
}

// choiceRequester answers RequestChoice with a fixed index, the way a UI would.
type choiceRequester struct {
	choice int
	asked  int
}

func (r *choiceRequester) RequestText(fmt.Stringer) string { return "" }

func (r *choiceRequester) RequestChoice(prompts []fmt.Stringer) int {
	r.asked++
	return r.choice
}

func runDeploy(c *Client, ctx context.Context, requester client.UserRequester, id client.AccountId) error {
	progress := make(chan client.ProgressAccounts)
	var opErr error
	go func() {
		defer close(progress)
		opErr = c.DeployAccounts(ctx, requester, progress, id)
	}()
	var accountErr error
	for p := range progress {
		if pa := p.Accounts[id]; pa != nil && pa.Err != nil {
			accountErr = pa.Err
		}
	}
	return errors.Join(opErr, accountErr)
}

// The recorded secret authenticating is positive proof the target holds it, so the
// rollback is stale bookkeeping and clearing it needs no confirmation. Left in
// place it would block every later secret update.
func TestDeploy_PendingSecretUpdateWithAWorkingSecretClearsSilently(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(true, "reachable"))

	requester := &choiceRequester{}
	if err := runDeploy(c, ctx, requester, account.Id); err != nil {
		t.Fatalf("DeployAccounts: %v", err)
	}
	if requester.asked != 0 {
		t.Fatalf("expected no question when the recorded secret works, asked %d times", requester.asked)
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if rollback.Valid {
		t.Fatalf("expected the stale rollback to be cleared, got %q", rollback.String)
	}
	if !strings.Contains(secret, `"key":"reachable"`) {
		t.Fatalf("expected the recorded secret to stay current, got %s", secret)
	}

	// And a later secret update is no longer blocked by orphaned bookkeeping.
	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(true, "later")); err != nil {
		t.Fatalf("secret update after a cleared rollback: %v", err)
	}
}

// The target predating the secret update is a real fork, so the operator decides.
// Discarding moves the previous secret back to current.
func TestDeploy_PendingSecretUpdateDiscardRestoresThePreviousSecret(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	// The recorded secret cannot authenticate, the previous one can, so the deploy
	// has to ask.
	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(false, "unconfirmed"))

	requester := &choiceRequester{choice: pendingSecretUpdateDiscard}
	if err := runDeploy(c, ctx, requester, account.Id); err != nil {
		t.Fatalf("DeployAccounts: %v", err)
	}
	if requester.asked == 0 {
		t.Fatal("expected the operator to be asked how to proceed")
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if rollback.Valid {
		t.Fatalf("expected discarding to leave no secret update outstanding, got %q", rollback.String)
	}
	if !strings.Contains(secret, `"key":"original"`) {
		t.Fatalf("expected the previous secret to be current again, got %s", secret)
	}
	if !hasAuditAction(t, c, ctx, "account.update.secret.discarded") {
		t.Fatal("expected an account.update.secret.discarded audit entry")
	}
}

// Choosing to finish routes into the secret update resume. The mock cannot model a
// target that holds a secret the first probe could not use, so the resume it runs
// here is the failing one -- which still shows the routing, and shows that a
// failed finish unwinds instead of stranding the account.
func TestDeploy_PendingSecretUpdateCompleteRoutesIntoTheResume(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(false, "unconfirmed"))

	requester := &choiceRequester{choice: pendingSecretUpdateComplete}
	if err := runDeploy(c, ctx, requester, account.Id); err == nil {
		t.Fatal("expected finishing an unusable secret update to report the failure")
	}
	if requester.asked == 0 {
		t.Fatal("expected the operator to be asked how to proceed")
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if rollback.Valid {
		t.Fatalf("expected the failed resume to unwind, got %q", rollback.String)
	}
	if !strings.Contains(secret, `"key":"original"`) {
		t.Fatalf("expected the previous secret to be current again, got %s", secret)
	}
}

// Declining either choice leaves the account exactly as it was, and says why.
func TestDeploy_PendingSecretUpdateRefusedLeavesEverythingAlone(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(false, "unconfirmed"))

	if err := runDeploy(c, ctx, &choiceRequester{choice: -1}, account.Id); err == nil {
		t.Fatal("expected the deploy to report the unresolved secret update")
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if !rollback.Valid {
		t.Fatal("expected the secret update to still be outstanding")
	}
	if !strings.Contains(secret, `"key":"unconfirmed"`) {
		t.Fatalf("expected the pending target to be untouched, got %s", secret)
	}
}

// With no one to ask, refusing is the only safe answer: both choices lose
// something, and guessing which loss is acceptable is not the client's call.
func TestDeploy_PendingSecretUpdateWithoutARequesterRefuses(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(false, "unconfirmed"))

	if err := runDeploy(c, ctx, nil, account.Id); err == nil {
		t.Fatal("expected the deploy to refuse with no requester to ask")
	}
	if _, rollback := secretColumns(t, c, ctx, account.Id); !rollback.Valid {
		t.Fatal("expected the secret update to still be outstanding")
	}
}

// Verify writes nothing, so it cannot settle the question -- it reports what it
// finds and leaves the bookkeeping alone.
func TestVerify_PendingSecretUpdateIsReadOnly(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(true, "unconfirmed"))
	before, rollbackBefore := secretColumns(t, c, ctx, account.Id)

	progress := make(chan client.ProgressAccounts)
	go func() {
		defer close(progress)
		_ = c.VerifyAccounts(ctx, &choiceRequester{}, progress, account.Id)
	}()
	for range progress {
	}

	after, rollbackAfter := secretColumns(t, c, ctx, account.Id)
	if after != before {
		t.Fatalf("verify changed connector_secret: %s -> %s", before, after)
	}
	if rollbackAfter != rollbackBefore {
		t.Fatalf("verify changed the rollback: %+v -> %+v", rollbackBefore, rollbackAfter)
	}
}

// UpdateAccount can repoint an account but must not be able to touch its
// credential; the signature enforces that, this pins the column.
func TestUpdateAccount_LeavesTheSecretAlone(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)
	before, _ := secretColumns(t, c, ctx, account.Id)

	updated, err := c.UpdateAccount(ctx, account.Id, "admin", "other.example", 2222)
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.Username != "admin" || updated.Host != "other.example" || updated.Port != 2222 {
		t.Fatalf("target not updated: %+v", updated)
	}

	after, _ := secretColumns(t, c, ctx, account.Id)
	if after != before {
		t.Fatalf("connector_secret changed: %s -> %s", before, after)
	}
}

func TestUpdateAccountConnectorForce_AbandonsAPendingSecretUpdateAndKeepsTheCache(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	// Give the account a cache, then strand it mid-secret update.
	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(true, "deployed")); err != nil {
		t.Fatalf("seed secret update: %v", err)
	}
	cacheBefore, err := c.accountConnectorCache(ctx, account.Id)
	if err != nil {
		t.Fatalf("accountConnectorCache: %v", err)
	}
	if cacheBefore == "" {
		t.Fatal("expected the seed secret update to leave a cache")
	}
	interruptSecretUpdate(t, c, ctx, account.Id, mockSecret(true, "stranded"))

	forced, err := c.UpdateAccountConnectorForce(ctx, account.Id, "mock", mockSecret(true, "manual"))
	if err != nil {
		t.Fatalf("UpdateAccountConnectorForce: %v", err)
	}
	if forced.Connector != "mock" {
		t.Fatalf("unexpected connector: %+v", forced)
	}

	secret, rollback := secretColumns(t, c, ctx, account.Id)
	if !strings.Contains(secret, `"key":"manual"`) {
		t.Fatalf("expected the forced secret to be current, got %s", secret)
	}
	if rollback.Valid {
		t.Fatalf("expected the force to abandon the outstanding secret update, got %q", rollback.String)
	}
	if !hasAuditAction(t, c, ctx, "account.update.connector_force") {
		t.Fatal("expected an account.update.connector_force audit entry")
	}

	// Same connector: the cache still parses and its known host is worth keeping,
	// so it is left in place.
	cacheAfter, err := c.accountConnectorCache(ctx, account.Id)
	if err != nil {
		t.Fatalf("accountConnectorCache: %v", err)
	}
	if cacheAfter != cacheBefore {
		t.Fatalf("expected the cache to be kept, got %q want %q", cacheAfter, cacheBefore)
	}

	// Nothing confirmed what the target holds, so the account reads dirty.
	dirty, err := c.IsAccountDirty(ctx, forced)
	if err != nil {
		t.Fatalf("IsAccountDirty: %v", err)
	}
	if !dirty {
		t.Fatal("expected a forced secret change to leave the account dirty")
	}
}

// A cache written by another connector is a foreign type its ParseCache rejects,
// which would break every later read of the account, so it has to be reset.
func TestUpdateAccountConnectorForce_ChangedConnectorResetsTheCache(t *testing.T) {
	c, ctx := newSecretTestClient(t)
	account := newMockAccount(t, c, ctx)

	if err := runUpdateSecret(c, ctx, account.Id, mockSecret(true, "deployed")); err != nil {
		t.Fatalf("seed secret update: %v", err)
	}

	if _, err := c.UpdateAccountConnectorForce(ctx, account.Id, "ssh", map[string]string{
		"private_key": testPrivateKeyPEM(t),
	}); err != nil {
		t.Fatalf("UpdateAccountConnectorForce: %v", err)
	}

	cache, err := c.accountConnectorCache(ctx, account.Id)
	if err != nil {
		t.Fatalf("accountConnectorCache: %v", err)
	}
	if cache != "" {
		t.Fatalf("expected the cache to be reset on a connector change, got %q", cache)
	}

	// The account still has to load: a stale foreign cache would fail ParseCache
	// and take the whole listing down with it.
	accounts, err := c.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts after a connector change: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Connector != "ssh" {
		t.Fatalf("unexpected accounts after a connector change: %+v", accounts)
	}
}
