package tags

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

// This test asserts the SQL produced by tag query-builder callbacks without
// executing the query. It uses an in-memory SQLite *only* as a formatter/driver
// provider so bun can render SQL; the test does not execute any statements or
// depend on SQLite-specific behaviour. This keeps the test unit-focused on
// SQL generation and avoids running GetAccountsByTagBun against SQLite.
func TestQueryBuilderRendering_TagMatchers(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer sqldb.Close()

	bdb := bun.NewDB(sqldb, sqlitedialect.New())
	defer bdb.Close()

	cases := []struct {
		matcher       string
		wantFragments []string
	}{
		{"prod", []string{"tag", "LIKE", "|prod|"}},
		{"!prod", []string{"tag", "NOT LIKE", "|prod|"}},
	}

	for _, c := range cases {
		sel := bdb.NewSelect()
		// Directly parse into the select's QueryBuilder to avoid the extra
		// validation path that uses a mock QueryBuilder (which can panic
		// when used for rendering). This keeps the test focused on SQL
		// generation without executing any queries.
		qb := sel.QueryBuilder()
		if _, err := parseTagMatcher(c.matcher, qb, true, false); err != nil {
			t.Fatalf("parseTagMatcher(%q) returned error: %v", c.matcher, err)
		}
		sqlStr := sel.String()

		for _, want := range c.wantFragments {
			if !strings.Contains(sqlStr, want) {
				t.Errorf("matcher %q: rendered SQL missing %q; got: %s", c.matcher, want, sqlStr)
			}
		}
	}

	if _, err := QueryBuilderFromTagMatcher("bad tag"); err == nil {
		t.Fatalf("expected QueryBuilderFromTagMatcher to reject invalid matcher")
	}
}
