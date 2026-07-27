// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/toeirei/keymaster/client"
	"github.com/toeirei/keymaster/ui/i18n"
)

const testAccountId = client.AccountId(7)

// collect drains the aggregate channel the way runInteractive does. Keeping every
// snapshot is only sound because the adapter hands out a fresh one per update; a
// shared entry would have been overwritten by the time it was read.
func collect(op accountOperation, ids ...client.AccountId) ([]client.ProgressAccountWithError, error) {
	aggregate := make(chan client.ProgressAccounts)
	var opErr error
	go func() {
		defer close(aggregate)
		opErr = op(context.Background(), nil, aggregate, ids...)
	}()

	var seen []client.ProgressAccountWithError
	for snapshot := range aggregate {
		seen = append(seen, *snapshot.Accounts[testAccountId])
	}
	return seen, opErr
}

// runInteractive builds its final message from the last snapshot it saw and reads
// a missing one as "never started", so an operation that fails before reporting
// anything would be shown as a cancellation. The adapter has to report once up
// front to keep that from happening.
func TestSingleAccountOperation_ReportsBeforeTheOperationCanFail(t *testing.T) {
	op := singleAccountOperation(func(ctx context.Context, _ client.UserRequester, progress chan<- client.UpdateSecretProgressAccount, id client.AccountId) error {
		return errors.New("failed before reporting anything")
	})

	seen, opErr := collect(op, testAccountId)
	if opErr != nil {
		t.Fatalf("expected the failure to be carried on the account, not returned: %v", opErr)
	}
	if len(seen) < 2 {
		t.Fatalf("expected an opening snapshot and a failure snapshot, got %d", len(seen))
	}
	if seen[0].Err != nil || seen[0].Progress != 0 {
		t.Fatalf("unexpected opening snapshot: %+v", seen[0])
	}
	last := seen[len(seen)-1]
	if last.Err == nil {
		t.Fatal("expected the failure to be carried on the account entry")
	}
	if last.Progress != 1 {
		t.Fatalf("a finished account should report full progress, got %v", last.Progress)
	}
}

func TestSingleAccountOperation_ForwardsProgress(t *testing.T) {
	op := singleAccountOperation(func(ctx context.Context, _ client.UserRequester, progress chan<- client.UpdateSecretProgressAccount, id client.AccountId) error {
		if id != testAccountId {
			t.Errorf("operation saw account %v, want %v", id, testAccountId)
		}
		progress <- client.UpdateSecretProgressAccount{0.5, i18n.RawText("halfway")}
		progress <- client.UpdateSecretProgressAccount{1, i18n.RawText("done")}
		return nil
	})

	seen, opErr := collect(op, testAccountId)
	if opErr != nil {
		t.Fatalf("unexpected error: %v", opErr)
	}
	// opening + the two forwarded updates
	if len(seen) != 3 {
		t.Fatalf("expected 3 snapshots, got %d: %+v", len(seen), seen)
	}
	if seen[1].Status.String() != "halfway" || seen[1].Progress != 0.5 {
		t.Fatalf("unexpected forwarded snapshot: %+v", seen[1])
	}
	last := seen[len(seen)-1]
	if last.Err != nil {
		t.Fatalf("a successful operation should leave no error: %v", last.Err)
	}
	if last.Status.String() != "done" || last.Progress != 1 {
		t.Fatalf("unexpected final snapshot: %+v", last)
	}
}

// The aggregate shape allows a batch, but this adapter only has one account's
// progress to report, so anything else is a programming error rather than
// something to silently run on the first id.
func TestSingleAccountOperation_RejectsAnythingButOneAccount(t *testing.T) {
	called := false
	op := singleAccountOperation(func(context.Context, client.UserRequester, chan<- client.UpdateSecretProgressAccount, client.AccountId) error {
		called = true
		return nil
	})

	for name, ids := range map[string][]client.AccountId{
		"none": {},
		"two":  {testAccountId, 8},
	} {
		t.Run(name, func(t *testing.T) {
			aggregate := make(chan client.ProgressAccounts)
			var opErr error
			go func() {
				defer close(aggregate)
				opErr = op(context.Background(), nil, aggregate, ids...)
			}()
			for range aggregate {
			}
			if opErr == nil {
				t.Fatal("expected an operation-level error")
			}
			if called {
				t.Fatal("the operation should not have run")
			}
		})
	}
}
