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

type bunTest2 struct {
	matcher  string
	sqlQuery string
}

var bunTests2 = []bunTest2{
	// --- Basic Exact Matches ---
	{"prod", "/* prod */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'prod') WHERE (tag_1.id IS NOT NULL)"},

	// --- Wildcards (* = single char, ** = any sequence) ---
	{"v*", "/* v* */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name LIKE 'v_' ESCAPE '!') WHERE (tag_1.id IS NOT NULL)"},
	{"api-**", "/* api-** */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name LIKE 'api!-%' ESCAPE '!') WHERE (tag_1.id IS NOT NULL)"},

	// --- NOT (!) ---
	{"!deprecated", "/* !deprecated */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'deprecated') WHERE (tag_1.id IS NULL)"},

	// --- AND (&) ---
	{"golang&backend", "/* golang&backend */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'golang') JOIN public_key_to_tag AS tagged_to_tag_2 ON (tagged_to_tag_2.public_key_id = public_key.id) JOIN tag AS tag_2 ON (tag_2.id = tagged_to_tag_2.tag_id AND tag_2.name = 'backend') WHERE ((tag_1.id IS NOT NULL)) AND ((tag_2.id IS NOT NULL))"},

	// --- OR (|) ---
	{"ios|android", "/* ios|android */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'ios') JOIN public_key_to_tag AS tagged_to_tag_2 ON (tagged_to_tag_2.public_key_id = public_key.id) JOIN tag AS tag_2 ON (tag_2.id = tagged_to_tag_2.tag_id AND tag_2.name = 'android') WHERE ((tag_1.id IS NOT NULL)) OR ((tag_2.id IS NOT NULL))"},

	// --- Complex Nesting ---
	{"(aws|gcp)&!legacy", "/* (aws|gcp)&!legacy */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_3 ON (tagged_to_tag_3.public_key_id = public_key.id) JOIN tag AS tag_3 ON (tag_3.id = tagged_to_tag_3.tag_id AND tag_3.name = 'legacy') JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'aws') JOIN public_key_to_tag AS tagged_to_tag_2 ON (tagged_to_tag_2.public_key_id = public_key.id) JOIN tag AS tag_2 ON (tag_2.id = tagged_to_tag_2.tag_id AND tag_2.name = 'gcp') WHERE (((tag_1.id IS NOT NULL)) OR ((tag_2.id IS NOT NULL))) AND ((tag_3.id IS NULL))"},
	{"auth&(**-admin|super-**)", "/* auth&(**-admin|super-**) */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_3 ON (tagged_to_tag_3.public_key_id = public_key.id) JOIN tag AS tag_3 ON (tag_3.id = tagged_to_tag_3.tag_id AND tag_3.name LIKE 'super!-%' ESCAPE '!') JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'auth') JOIN public_key_to_tag AS tagged_to_tag_2 ON (tagged_to_tag_2.public_key_id = public_key.id) JOIN tag AS tag_2 ON (tag_2.id = tagged_to_tag_2.tag_id AND tag_2.name LIKE '%!-admin' ESCAPE '!') WHERE ((tag_1.id IS NOT NULL)) AND (((tag_2.id IS NOT NULL)) OR ((tag_3.id IS NOT NULL)))"},
	{"!(test|stage)&prod", "/* !(test|stage)&prod */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_3 ON (tagged_to_tag_3.public_key_id = public_key.id) JOIN tag AS tag_3 ON (tag_3.id = tagged_to_tag_3.tag_id AND tag_3.name = 'prod') JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name = 'test') JOIN public_key_to_tag AS tagged_to_tag_2 ON (tagged_to_tag_2.public_key_id = public_key.id) JOIN tag AS tag_2 ON (tag_2.id = tagged_to_tag_2.tag_id AND tag_2.name = 'stage') WHERE (((tag_1.id IS NULL)) AND ((tag_2.id IS NULL))) AND ((tag_3.id IS NOT NULL))"},

	// --- Edge Cases ---
	{"**", "/* ** */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name LIKE '%' ESCAPE '!') WHERE (tag_1.id IS NOT NULL)"},
	{"*", "/* * */ SELECT \"id\" FROM \"public_key\" JOIN public_key_to_tag AS tagged_to_tag_1 ON (tagged_to_tag_1.public_key_id = public_key.id) JOIN tag AS tag_1 ON (tag_1.id = tagged_to_tag_1.tag_id AND tag_1.name LIKE '_' ESCAPE '!') WHERE (tag_1.id IS NOT NULL)"},
}

func TestTagsExprToSubquery2(t *testing.T) {
	for _, bt := range bunTests2 {
		t.Run(bt.matcher, func(t *testing.T) {
			expr, err := tags.ParseMatcher(bt.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", bt.matcher, err)
			}

			// apply expr to new QueryBuilder
			sq := bunhelpers.TagsExprToSubquery2(
				bun.NewDB(nil, sqlitedialect.New()).NewSelect().Table("public_key").Column("id").Comment(bt.matcher),
				bunhelpers.TagsExprToSubqueryConfig2{
					TaggedTable:    "public_key",
					TaggedColumnId: "id",

					TaggedToTagTable:          "public_key_to_tag",
					TaggedToTagColumnTagId:    "tag_id",
					TaggedToTagColumnTaggedId: "public_key_id",

					TagTable:       "tag",
					TagColumnId:    "id",
					TagColumnValue: "name",
				},
				expr.Optimize(),
			)

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
