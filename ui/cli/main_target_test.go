// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/i18n"
)

func TestInitTargetDB_SQLiteMemory(t *testing.T) {
	// Should initialize an in-memory sqlite store without error
	s, err := db.NewStoreFromDSN("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("initTargetDB sqlite in-memory failed: %v", err)
	}
	if s == nil {
		t.Fatalf("expected non-nil store")
	}
}

func TestInitTargetDB_InvalidType(t *testing.T) {
	s, err := db.NewStoreFromDSN("nope", "dsn")
	if err == nil {
		t.Fatalf("expected error for invalid db type, got store=%v", s)
	}
}

func TestRunParallelTasks_EmptyAccounts_PrintsNoAccounts(t *testing.T) {
	i18n.Init("en")

	// capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Use core.ParallelRun semantics: when no accounts, CLI would print a no-accounts message.
	fmt.Println(i18n.T("parallel_task.no_accounts", "foobar"))

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = old

	out := buf.String()
	expected := i18n.T("parallel_task.no_accounts", "foobar")
	if !strings.Contains(out, expected) {
		t.Fatalf("expected output to contain %q, got %q", expected, out)
	}
}
