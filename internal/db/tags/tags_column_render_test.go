package tags

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

// Ensure the column-aware parser generates SQL that references a provided
// column expression (for join/alias compatibility) without executing queries.
func TestQueryBuilderRendering_ColumnAware(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer func() {
		if err := sqldb.Close(); err != nil {
			t.Fatalf("failed to close sqlite: %v", err)
		}
	}()

	bdb := bun.NewDB(sqldb, sqlitedialect.New())
	defer func() {
		if err := bdb.Close(); err != nil {
			t.Fatalf("failed to close bun DB: %v", err)
		}
	}()

	sel := bdb.NewSelect().TableExpr("accounts AS a")
	qb := sel.QueryBuilder()

	if _, err := parseTagMatcherColumn("prod", qb, true, false, "a.tags"); err != nil {
		t.Fatalf("parseTagMatcherColumn returned error: %v", err)
	}
	sqlStr := sel.String()
	if !strings.Contains(sqlStr, "a.tags") || !strings.Contains(sqlStr, "|prod|") {
		t.Fatalf("rendered SQL missing expected fragments; got: %s", sqlStr)
	}
}
