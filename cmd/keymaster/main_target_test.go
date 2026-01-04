package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestInitTargetDB_SQLiteMemory(t *testing.T) {
	// Should initialize an in-memory sqlite store without error
	s, err := initTargetDB("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("initTargetDB sqlite in-memory failed: %v", err)
	}
	if s == nil {
		t.Fatalf("expected non-nil store")
	}
}

func TestInitTargetDB_InvalidType(t *testing.T) {
	s, err := initTargetDB("nope", "dsn")
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

	task := parallelTask{name: "foobar", startMsg: "START", successMsg: "OK", failMsg: "FAIL", taskFunc: func(a model.Account) error { return nil }}
	runParallelTasks([]model.Account{}, task)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = old

	out := buf.String()
	expected := i18n.T("parallel_task.no_accounts", task.name)
	if !strings.Contains(out, expected) {
		t.Fatalf("expected output to contain %q, got %q", expected, out)
	}
}
