// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags_test

import (
	"fmt"
	"testing"

	tags "github.com/toeirei/keymaster/tags"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/schema"
)

type bunTest struct {
	matcher  string
	sqlQuery string
}

var bunTests = []bunTest{
	// --- Basic Exact Matches ---
	{"prod", "SELECT * WHERE (tags LIKE '%|prod|%' ESCAPE '!')"},

	// --- Wildcards (* = single char, ** = any sequence) ---
	{"v*", "SELECT * WHERE (tags LIKE '%|v_|%' ESCAPE '!')"},
	{"api-**", "SELECT * WHERE (tags LIKE '%|api!-%|%' ESCAPE '!')"},

	// --- NOT (!) ---
	{"!deprecated", "SELECT * WHERE (tags NOT LIKE '%|deprecated|%' ESCAPE '!')"},

	// --- AND (&) ---
	{"golang&backend", "SELECT * WHERE (tags LIKE '%|golang|%' ESCAPE '!') AND (tags LIKE '%|backend|%' ESCAPE '!')"},

	// --- OR (|) ---
	{"ios|android", "SELECT * WHERE (tags LIKE '%|ios|%' ESCAPE '!') OR (tags LIKE '%|android|%' ESCAPE '!')"},

	// --- Complex Nesting ---
	{"(aws|gcp)&!legacy", "SELECT * WHERE ((tags LIKE '%|aws|%' ESCAPE '!') OR (tags LIKE '%|gcp|%' ESCAPE '!')) AND (tags NOT LIKE '%|legacy|%' ESCAPE '!')"},
	{"auth&(**-admin|super-**)", "SELECT * WHERE (tags LIKE '%|auth|%' ESCAPE '!') AND ((tags LIKE '%|%!-admin|%' ESCAPE '!') OR (tags LIKE '%|super!-%|%' ESCAPE '!'))"},
	{"!(test|stage)&prod", "SELECT * WHERE (tags NOT LIKE '%|test|%' ESCAPE '!') AND (tags NOT LIKE '%|stage|%' ESCAPE '!') AND (tags LIKE '%|prod|%' ESCAPE '!')"},

	// --- Edge Cases ---
	{"**", "SELECT * WHERE (tags LIKE '%|%|%' ESCAPE '!')"},
	{"*", "SELECT * WHERE (tags LIKE '%|_|%' ESCAPE '!')"},
}

func TestApplyToBunQuery(t *testing.T) {
	for _, bt := range bunTests {
		t.Run(fmt.Sprintf("Matcher(%q)", bt.matcher), func(t *testing.T) {
			expr, err := tags.ParseMatcher(bt.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", bt.matcher, err)
			}

			// apply expr to new QueryBuilder
			qb := tags.ApplyToBunQuery(expr, bun.NewSelectQuery(nil).QueryBuilder(), "tags")

			// render QueryBuilder to fresh []byte
			queryBytes, err := qb.AppendQuery(schema.NewQueryGen(sqlitedialect.New()), []byte{})
			if err != nil {
				t.Fatalf("failed to render QueryBuilder: %v", err)
			}
			got := string(queryBytes)

			if got != bt.sqlQuery {
				t.Errorf("got = %q; expected = %q", got, bt.sqlQuery)
			}

			t.Log(got)
		})
	}
}
