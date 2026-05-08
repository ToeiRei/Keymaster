// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package bunhelpers_test

import (
	"testing"

	"github.com/toeirei/keymaster/bunhelpers"
	"github.com/toeirei/keymaster/tags"
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
	{"prod", "/* prod */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'prod')"},

	// --- Wildcards (* = single char, ** = any sequence) ---
	{"v*", "/* v* */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name LIKE 'v_' ESCAPE '!')"},
	{"api-**", "/* api-** */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name LIKE 'api!-%' ESCAPE '!')"},

	// --- NOT (!) ---
	{"!deprecated", "/* !deprecated */ SELECT \"id\" FROM \"public_key\" WHERE (id NOT IN (/* deprecated */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'deprecated')))"},

	// --- AND (&) ---
	{"golang&backend", "/* golang & backend */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* golang */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'golang'))) AND (id IN (/* backend */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'backend')))"},

	// --- OR (|) ---
	{"ios|android", "/* ios | android */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* ios */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'ios'))) OR (id IN (/* android */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'android')))"},

	// --- Complex Nesting ---
	{"(aws|gcp)&!legacy", "/* (aws | gcp) & !legacy */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* aws | gcp */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* aws */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'aws'))) OR (id IN (/* gcp */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'gcp'))))) AND (id IN (/* !legacy */ SELECT \"id\" FROM \"public_key\" WHERE (id NOT IN (/* legacy */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'legacy')))))"},
	{"auth&(**-admin|super-**)", "/* auth & (**-admin | super-**) */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* auth */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'auth'))) AND (id IN (/* **-admin | super-** */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* **-admin */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name LIKE '%!-admin' ESCAPE '!'))) OR (id IN (/* super-** */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name LIKE 'super!-%' ESCAPE '!')))))"},
	{"!(test|stage)&prod", "/* !(test | stage) & prod */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* !(test | stage) */ SELECT \"id\" FROM \"public_key\" WHERE (id NOT IN (/* test | stage */ SELECT \"id\" FROM \"public_key\" WHERE (id IN (/* test */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'test'))) OR (id IN (/* stage */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'stage'))))))) AND (id IN (/* prod */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name = 'prod')))"},

	// --- Edge Cases ---
	{"**", "/* ** */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name LIKE '%' ESCAPE '!')"},
	{"*", "/* * */ SELECT \"public_key_id\" FROM \"public_key_to_tag\" JOIN tag AS tag ON (tag.id = public_key_to_tag.tag_id) WHERE (tag.name LIKE '_' ESCAPE '!')"},
}

func TestTagsExprToSubquery(t *testing.T) {
	for _, bt := range bunTests {
		t.Run(bt.matcher, func(t *testing.T) {
			expr, err := tags.ParseMatcher(bt.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", bt.matcher, err)
			}

			// apply expr to new QueryBuilder
			sq := bunhelpers.TagsExprToSubquery(bunhelpers.TagsExprToSubqueryConfig{
				Db: bun.NewDB(nil, sqlitedialect.New()),

				// TaggedModel:    (*struct{})(nil),
				TaggedTable:    "public_key",
				TaggedColumnId: "id",

				TaggedToTagTable:          "public_key_to_tag",
				TaggedToTagColumnTagId:    "tag_id",
				TaggedToTagColumnTaggedId: "public_key_id",

				TagTable:       "tag",
				TagColumnId:    "id",
				TagColumnValue: "name",
			}, expr.Optimize())

			// render QueryBuilder to fresh []byte
			queryBytes, err := sq.AppendQuery(schema.NewQueryGen(sqlitedialect.New()), []byte{})
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
