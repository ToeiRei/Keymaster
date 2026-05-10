// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tagsbun_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bobg/go-generics/v4/slices"
	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/tags/tagsbun"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type Tagged struct {
	bun.BaseModel `bun:"table:tagged"`

	Id   int64  `bun:"id,pk,autoincrement"`
	Name string `bun:"name,notnull"`
}

type Tag struct {
	bun.BaseModel `bun:"table:tag"`

	Id    int64  `bun:"id,pk,autoincrement"`
	Value string `bun:"value,notnull,unique"`
}

type TaggedToTag struct {
	bun.BaseModel `bun:"table:tagged_to_tag"`

	TagId    int64 `bun:"tag_id,pk"`
	TaggedId int64 `bun:"tagged_id,pk"`
}

var db *bun.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Open an in-memory SQLite connection
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	db = bun.NewDB(sqldb, sqlitedialect.New())

	// 2. Setup Tables
	models := []interface{}{
		(*Tagged)(nil),
		(*Tag)(nil),
		(*TaggedToTag)(nil),
	}

	for _, model := range models {
		_, err := db.NewCreateTable().Model(model).Exec(ctx)
		if err != nil {
			panic(fmt.Sprintf("failed to create table for %T: %v", model, err))
		}
	}

	// 3. Seed Data
	if err := seedTestData(ctx); err != nil {
		panic(fmt.Sprintf("failed to seed data: %v", err))
	}

	// 4. Run Tests
	code := m.Run()

	// 5. Teardown
	db.Close()
	os.Exit(code)
}

func addTagged(ctx context.Context, name string, tagValues tags.Tags) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// 1. Create the Tagged record
		tagged := &Tagged{Name: name}
		if _, err := tx.NewInsert().Model(tagged).Exec(ctx); err != nil {
			return err
		}

		if len(tagValues) == 0 {
			return nil
		}

		// 2. Prepare Tag structs
		tags := make([]Tag, len(tagValues))
		for i, v := range tagValues {
			tags[i] = Tag{Value: string(v)}
		}

		// 3. Upsert Tags: Insert new ones, ignore existing ones
		// We use "ON CONFLICT DO NOTHING" to avoid errors on duplicates
		if _, err := tx.NewInsert().
			Model(&tags).
			On("CONFLICT (value) DO NOTHING").
			Exec(ctx); err != nil {
			return err
		}

		// 4. Retrieve all Ids for these tags (both existing and newly created)
		var actualTags []Tag
		if err := tx.NewSelect().
			Model(&actualTags).
			Where("value IN (?)", bun.In(tagValues)).
			Scan(ctx); err != nil {
			return err
		}

		// 5. Link them in the Join Table
		taggedToTags := make([]TaggedToTag, len(actualTags))
		for i, t := range actualTags {
			taggedToTags[i] = TaggedToTag{
				TaggedId: tagged.Id,
				TagId:    t.Id,
			}
		}

		_, err := tx.NewInsert().Model(&taggedToTags).Exec(ctx)
		return err
	})
}

var testData = map[string]tags.Tags{
	"max":  {"prod"},
	"maxi": {"prod", "test"},
	"maxo": {"prod", "stage"},
	"leo":  {"stage"},
	"amy":  {"v1"},
	"liz":  {"v2"},
	"sam":  {"v12"},
	"ben":  {"version"},
	"eve":  {"api-user"},
	"joe":  {"api-admin"},
	"ray":  {"backend-api"},
	"meg":  {"apix"},
	"jay":  {"active"},
	"ann":  {"stable"},
	"eli":  {"deprecated"},
	"liv":  {"golang", "backend"},
	"tom":  {"golang"},
	"ada":  {"backend"},
	"nico": {"golang", "frontend"},
	"kai":  {"ios"},
	"zoe":  {"android"},
	"rick": {"windows"},
	"roy":  {"aws"},
	"gus":  {"gcp"},
	"ivy":  {"aws", "legacy"},
	"mia":  {"gcp", "legacy"},
	"luc":  {"x", "something else"},
	"samy": {"auth", "user-admin"},
	"evo":  {"auth", "super-user"},
	"wes":  {"auth"},
	"tia":  {"auth", "user"},

	// some unused names for future tests:
	// "jon": {},
	// "lyn": {},
	// "noa": {},
}

func seedTestData(ctx context.Context) error {
	for name, tags := range testData {
		err := addTagged(ctx, name, tags)
		if err != nil {
			return err
		}
	}

	return nil
}

func queryToString(t *testing.T, query bun.Query) string {
	queryBytes, err := query.AppendQuery(db.QueryGen(), []byte{})
	if err != nil {
		t.Fatalf("failed to render Query: %v", err)
	}

	return string(queryBytes)
}

type testCase struct {
	matcher string
	results []string
}

var testCases = []testCase{
	{"prod", []string{"max", "maxi", "maxo"}},
	{"v*", []string{"amy", "liz"}},
	{"api-**", []string{"eve", "joe"}},
	{"!deprecated", []string{"max", "maxi", "maxo", "leo", "amy", "liz", "sam", "ben", "eve", "joe", "ray", "meg", "jay", "ann", "liv", "tom", "ada", "nico", "kai", "zoe", "rick", "roy", "gus", "ivy", "mia", "luc", "samy", "evo", "wes", "tia"}},
	{"golang&backend", []string{"liv"}},
	{"ios|android", []string{"kai", "zoe"}},
	{"(aws|gcp)&!legacy", []string{"roy", "gus"}},
	{"auth&(**-admin|super-**)", []string{"samy", "evo"}},
	{"!(test|stage)&prod", []string{"max"}},
	{"**", []string{"max", "maxi", "maxo", "leo", "amy", "liz", "sam", "ben", "eve", "joe", "ray", "meg", "jay", "ann", "eli", "liv", "tom", "ada", "nico", "kai", "zoe", "rick", "roy", "gus", "ivy", "mia", "luc", "samy", "evo", "wes", "tia"}},
	{"*", []string{"luc"}},
}

func TestTagsExprToWhere(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.matcher, func(t *testing.T) {
			expr, err := tags.ParseMatcher(tc.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", tc.matcher, err)
			}

			// create select statement
			var taggeds []Tagged
			sq := db.NewSelect().
				Model(&taggeds).
				Column("id", "name").
				Apply(tagsbun.TagsExprToWhere(expr, tagsbun.TagsExprToSubqueryConfig{
					TaggedTable:    "tagged",
					TaggedColumnId: "id",

					TaggedToTagTable:          "tagged_to_tag",
					TaggedToTagColumnTagId:    "tag_id",
					TaggedToTagColumnTaggedId: "tagged_id",

					TagTable:       "tag",
					TagColumnId:    "id",
					TagColumnValue: "value",
				})).
				Comment(tc.matcher)

			// log rendered query for debugging
			t.Logf("query: %s", queryToString(t, sq))

			// run query
			err = sq.Scan(t.Context())
			if err != nil {
				t.Fatalf("failed query tagged: %v", err)
			}

			// run expr.Eval against the same test data
			evalResults := slicest.Filter(slicest.MapKeys(testData), func(name string) bool {
				return expr.Eval(testData[name])
			})

			// concatinate expected
			expectedResult := slices.Clone(tc.results)
			slices.Sort(expectedResult)
			expected := strings.Join(expectedResult, ";")

			// concatinate got
			gotResult := slicest.Map(taggeds, func(tagges Tagged) string { return tagges.Name })
			slices.Sort(gotResult)
			got := strings.Join(gotResult, ";")

			// check got
			t.Logf("got: %s", got)
			if got != expected {
				t.Fatalf("expected: %s", expected)
			}

			// concatinate eval
			slices.Sort(evalResults)
			eval := strings.Join(evalResults, ";")

			// check eval
			t.Logf("eval: %s", eval)
			if eval != expected {
				t.Fatalf("expected: %s", expected)
			}
		})
	}
}
