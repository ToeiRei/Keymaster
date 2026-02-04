// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"testing"

	dbtags "github.com/toeirei/keymaster/internal/core/db/tags"
	"github.com/uptrace/bun"
)

func TestSetDebugAndDbLogf(t *testing.T) {
	// ensure no panic when toggling debug and calling dbLogf
	SetDebug(true)
	dbLogf("test debug %s", "on")
	SetDebug(false)
	dbLogf("test debug %s", "off")
}

func TestRunMigrationsBun_NilAndReal(t *testing.T) {
	// nil bun.DB
	if err := RunMigrationsBun(nil, "sqlite"); err != nil {
		t.Fatalf("expected nil error for nil bun.DB: %v", err)
	}

	// WithTestStore provides a real DB
	WithTestStore(t, func(s *BunStore) {
		if err := RunMigrationsBun(s.BunDB(), "sqlite"); err != nil {
			t.Fatalf("RunMigrationsBun failed on real DB: %v", err)
		}
	})
}

func TestQueryBuilderFromTagMatcherColumn_And_SplitTagsSafe(t *testing.T) {
	// Basic valid matcher
	qbFunc, err := dbtags.QueryBuilderFromTagMatcherColumn("tags", "env:prod")
	if err != nil {
		t.Fatalf("QueryBuilderFromTagMatcherColumn returned error: %v", err)
	}
	// Call returned function with a SelectQuery query builder to ensure no panic
	qb := (&bun.SelectQuery{}).QueryBuilder()
	_ = qbFunc(qb)

	// SplitTagsSafe: valid and invalid
	out := dbtags.SplitTagsSafe("|a|b|")
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("SplitTagsSafe valid returned unexpected: %#v", out)
	}
	out2 := dbtags.SplitTagsSafe("no-delims")
	if len(out2) != 0 {
		t.Fatalf("SplitTagsSafe invalid should return empty slice, got: %#v", out2)
	}
}
