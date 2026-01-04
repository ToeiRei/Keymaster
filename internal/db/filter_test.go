package db

import (
	"reflect"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func mkAcc(id int, user, host, label string) model.Account {
	return model.Account{ID: id, Username: user, Hostname: host, Label: label}
}

func TestFilterAccountsByTokens_NoTokensReturnsAll(t *testing.T) {
	in := []model.Account{mkAcc(1, "u1", "h1", "l1"), mkAcc(2, "u2", "h2", "l2")}
	out := FilterAccountsByTokens(in, nil)
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("expected same slice when no tokens, got %#v", out)
	}
}

func TestFilterAccountsByTokens_MatchUsername(t *testing.T) {
	in := []model.Account{mkAcc(1, "alice", "host", "lab"), mkAcc(2, "bob", "host", "lab")}
	out := FilterAccountsByTokens(in, []string{"ali"})
	if len(out) != 1 || out[0].Username != "alice" {
		t.Fatalf("unexpected result: %#v", out)
	}
}

func TestFilterAccountsByTokens_MatchMultipleTokens(t *testing.T) {
	in := []model.Account{
		mkAcc(1, "alice", "web-prod", "frontend"),
		mkAcc(2, "alice", "db-prod", "backend"),
	}
	out := FilterAccountsByTokens(in, []string{"alice", "prod"})
	if len(out) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(out))
	}
	out = FilterAccountsByTokens(in, []string{"alice", "db"})
	if len(out) != 1 || out[0].Hostname != "db-prod" {
		t.Fatalf("expected single db match, got %#v", out)
	}
}

func TestFilterAccountsByTokens_NoMatch(t *testing.T) {
	in := []model.Account{mkAcc(1, "u1", "h1", "l1")}
	out := FilterAccountsByTokens(in, []string{"nomatch"})
	if len(out) != 0 {
		t.Fatalf("expected no matches, got %#v", out)
	}
}
