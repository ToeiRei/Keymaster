package tui

import (
    "strings"
    "testing"

    "github.com/toeirei/keymaster/internal/i18n"
    "github.com/toeirei/keymaster/internal/model"
)

func TestRebuildDisplayedAccounts_Filtering(t *testing.T) {
    i18n.Init("en")
    m := &accountsModel{}
    m.accounts = []model.Account{
        {ID: 1, Username: "alice", Hostname: "host1", Label: "ops", Tags: "dev"},
        {ID: 2, Username: "bob", Hostname: "db", Label: "db", Tags: "prod"},
        {ID: 3, Username: "carol", Hostname: "host2", Label: "web", Tags: "vault"},
    }

    m.filter = "host"
    m.rebuildDisplayedAccounts()
    if len(m.displayedAccounts) != 2 {
        t.Fatalf("expected 2 accounts matching 'host', got %d", len(m.displayedAccounts))
    }

    // filter by tag
    m.filter = "vault"
    m.rebuildDisplayedAccounts()
    if len(m.displayedAccounts) != 1 || m.displayedAccounts[0].Username != "carol" {
        t.Fatalf("expected only carol for 'vault', got: %v", m.displayedAccounts)
    }
}

func TestListContentView_CursorIndicatorAndEntries(t *testing.T) {
    i18n.Init("en")
    m := &accountsModel{}
    m.displayedAccounts = []model.Account{{ID: 1, Username: "alice", Hostname: "h1"}, {ID: 2, Username: "bob", Hostname: "h2"}}
    m.cursor = 1
    out := m.listContentView()
    // Expect selected marker on the second item
    if !strings.Contains(out, "▸ ") {
        t.Fatalf("expected cursor marker in list content, got: %q", out)
    }
    if !strings.Contains(out, "bob") || !strings.Contains(out, "alice") {
        t.Fatalf("expected both alice and bob in output, got: %q", out)
    }
}
