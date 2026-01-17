package tags

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

func TestSplitOnTopLevelChar_NestedAndSimple(t *testing.T) {
	// nested should not split at top-level
	in := "a|(b|c)"
	parts := splitOnTopLevelChar(in, '|')
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != "a" || strings.TrimSpace(parts[1]) != "(b|c)" {
		t.Fatalf("unexpected split result for %q: %#v", in, parts)
	}

	// simple no-op
	single := splitOnTopLevelChar("abc", '&')
	if len(single) != 1 || single[0] != "abc" {
		t.Fatalf("expected single element for simple input, got: %#v", single)
	}

	// nested with deeper parentheses — expect three top-level parts here
	deep := splitOnTopLevelChar("(a&(b|c))|d|e", '|')
	if len(deep) != 3 || strings.TrimSpace(deep[0]) != "(a&(b|c))" || strings.TrimSpace(deep[1]) != "d" || strings.TrimSpace(deep[2]) != "e" {
		t.Fatalf("unexpected deep split: %#v", deep)
	}
}

func TestParseTagMatcher_ValidationOnly_DoesNotPanic(t *testing.T) {
	// validation-only path uses a nil QueryBuilder — ensure complex expressions validate
	if _, err := parseTagMatcherColumn("(prod|staging)&!dev", nil, true, false, "tags"); err != nil {
		t.Fatalf("expected validation to succeed, got: %v", err)
	}
}

func TestWildcardsAndEscaping_RendersExpectedSQL(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	defer sqldb.Close()

	bdb := bun.NewDB(sqldb, sqlitedialect.New())
	defer bdb.Close()

	sel := bdb.NewSelect()
	qb := sel.QueryBuilder()

	// '*' should become '_' and be embedded in the joined tags pattern
	if _, err := parseTagMatcherColumn("pro*", qb, true, false, "tagcol"); err != nil {
		t.Fatalf("parseTagMatcherColumn returned error: %v", err)
	}
	sqlStr := sel.String()
	if !strings.Contains(sqlStr, "LIKE") || !strings.Contains(sqlStr, "pro_") {
		t.Fatalf("rendered SQL did not contain expected wildcard fragment: %s", sqlStr)
	}
}

func TestSplitOnTopLevelChar_NoSplitInsideParens(t *testing.T) {
	in := "(a|b)|c"
	parts := splitOnTopLevelChar(in, '|')
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != "(a|b)" || strings.TrimSpace(parts[1]) != "c" {
		t.Fatalf("unexpected split for %q: %#v", in, parts)
	}
}
