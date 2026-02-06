package db

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/core/db/tags"
)

func TestDebugStoredTags(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// create accounts with tags
		_, err := AddAccountBun(bdb, "u1", "h1", "label1", tags.JoinTags([]string{"prod", "web"}))
		if err != nil {
			t.Fatalf("AddAccountBun a1 failed: %v", err)
		}
		_, err = AddAccountBun(bdb, "u2", "h2", "label2", tags.JoinTags([]string{"staging"}))
		if err != nil {
			t.Fatalf("AddAccountBun a2 failed: %v", err)
		}

		// fetch tags directly
		type row struct{ Tags string }
		var rows []row
		if err := QueryRawInto(context.Background(), bdb, &rows, "SELECT tags FROM accounts"); err != nil {
			t.Fatalf("select tags failed: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		// sanity check content
		if rows[0].Tags == "" && rows[1].Tags == "" {
			t.Fatalf("stored tags appear empty: %+v", rows)
		}
		t.Logf("stored tags rows: %+v", rows)

		// Render the select used by GetAccountsByTagBun and inspect SQL
		tagQb, err := tags.QueryBuilderFromTagMatcherColumn("tags", "prod")
		if err != nil {
			t.Fatalf("QueryBuilderFromTagMatcherColumn failed: %v", err)
		}
		sel := bdb.NewSelect().Model((*AccountModel)(nil))
		sel = sel.ApplyQueryBuilder(tagQb)
		t.Logf("Rendered SQL: %s", sel.String())

		// Try executing the select into account models to reproduce behavior
		var am []AccountModel
		if err := sel.OrderExpr("label, hostname, username").Scan(context.Background(), &am); err != nil {
			t.Fatalf("executing select failed: %v", err)
		}
		if len(am) == 0 {
			t.Fatalf("executed select returned 0 rows; expected matches; SQL: %s", sel.String())
		}
	})
}
