package db

import (
	"context"
	"testing"

	"github.com/toeirei/keymaster/core/db/tags"
)

// Test QueryBuilderFromTagMatcher end-to-end by inserting accounts and
// retrieving them via the tag matcher helper.
func TestGetAccountsByTagMatcher_DB(t *testing.T) {
	WithTestStore(t, func(s *BunStore) {
		bdb := s.BunDB()

		// create accounts with tags
		a1, err := AddAccountBun(bdb, "u1", "h1", "label1", tags.JoinTags([]string{"prod", "web"}))
		if err != nil {
			t.Fatalf("AddAccountBun a1 failed: %v", err)
		}
		_, err = AddAccountBun(bdb, "u2", "h2", "label2", tags.JoinTags([]string{"staging"}))
		if err != nil {
			t.Fatalf("AddAccountBun a2 failed: %v", err)
		}

		// simple tag
		res, err := GetAccountsByTagBun(context.Background(), bdb, "prod")
		if err != nil {
			t.Fatalf("GetAccountsByTagBun prod failed: %v", err)
		}
		if len(res) != 1 || res[0].ID != a1 {
			t.Fatalf("expected single prod account, got: %+v", res)
		}

		// AND matcher prod&web -> should match a1
		res, err = GetAccountsByTagBun(context.Background(), bdb, "prod&web")
		if err != nil {
			t.Fatalf("GetAccountsByTagBun prod&web failed: %v", err)
		}
		if len(res) != 1 || res[0].ID != a1 {
			t.Fatalf("expected single prod&web account, got: %+v", res)
		}

		// OR matcher prod|staging -> should match both
		res, err = GetAccountsByTagBun(context.Background(), bdb, "prod|staging")
		if err != nil {
			t.Fatalf("GetAccountsByTagBun prod|staging failed: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("expected two accounts for prod|staging, got: %+v", res)
		}

		// invalid matcher should return error via QueryBuilderFromTagMatcher path
		if _, err := tags.QueryBuilderFromTagMatcher("bad tag"); err == nil {
			t.Fatalf("expected QueryBuilderFromTagMatcher to reject invalid matcher")
		}
	})
}
