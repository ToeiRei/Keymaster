// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func TestFilterAccounts_LocalAndSearcher(t *testing.T) {
	accounts := []model.Account{
		{ID: 1, Username: "alice", Hostname: "host1", Label: "dev", Tags: "team-a"},
		{ID: 2, Username: "bob", Hostname: "host2", Label: "ops", Tags: "team-b"},
		{ID: 3, Username: "carol", Hostname: "host3", Label: "qa", Tags: "team-a"},
	}

	// Empty query returns all
	res := FilterAccounts(accounts, "", nil)
	if len(res) != 3 {
		t.Fatalf("expected 3 accounts for empty query, got %d", len(res))
	}

	// Local match: by username
	res = FilterAccounts(accounts, "bob", nil)
	if len(res) != 1 || res[0].Username != "bob" {
		t.Fatalf("expected bob in local results, got %+v", res)
	}

	// Searcher provided and returns results -> should prefer searcher
	searcher := func(q string) ([]model.Account, error) {
		return []model.Account{{ID: 99, Username: "x"}}, nil
	}
	res = FilterAccounts(accounts, "nonexistent", searcher)
	if len(res) != 1 || res[0].ID != 99 {
		t.Fatalf("expected searcher result to be returned, got %+v", res)
	}

	// Searcher error -> fallback to local
	searcherErr := func(q string) ([]model.Account, error) { return nil, errors.New("boom") }
	res = FilterAccounts(accounts, "carol", searcherErr)
	if len(res) != 1 || res[0].Username != "carol" {
		t.Fatalf("expected local fallback on searcher error, got %+v", res)
	}

	// Searcher returns empty -> fallback to local
	searcherEmpty := func(q string) ([]model.Account, error) { return []model.Account{}, nil }
	res = FilterAccounts(accounts, "team-a", searcherEmpty)
	if len(res) != 2 {
		t.Fatalf("expected local fallback when searcher empty, got %d", len(res))
	}
}

// Note: ContainsIgnoreCase and FilterKeys already have dedicated tests
// in other files; keep this file focused on searcher/selection behaviors.

func TestEnsureCursorInView(t *testing.T) {
	// viewport height 5
	h := 5

	// cursor above top
	if got := EnsureCursorInView(0, 2, h); got != 0 {
		t.Fatalf("expected top when cursor above, got %d", got)
	}

	// cursor below bottom
	// top=0 bottom=4, cursor=7 => expect 7-5+1 = 3
	if got := EnsureCursorInView(7, 0, h); got != 3 {
		t.Fatalf("expected 3 when cursor below, got %d", got)
	}

	// cursor within view -> unchanged
	if got := EnsureCursorInView(3, 0, h); got != 0 {
		t.Fatalf("expected unchanged yOffset 0, got %d", got)
	}
}
