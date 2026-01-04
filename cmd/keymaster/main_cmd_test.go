package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

func TestNewRootCmd_RegistersSubcommandsAndVersion(t *testing.T) {
	// Preserve globals
	oldV := version
	oldC := gitCommit
	oldD := buildDate
	version = "v9.9.9"
	gitCommit = "deadbeef"
	buildDate = "2025-01-02T15:04:05Z"
	defer func() {
		version = oldV
		gitCommit = oldC
		buildDate = oldD
	}()

	cmd := NewRootCmd()
	if cmd == nil {
		t.Fatalf("NewRootCmd returned nil")
	}

	// Version should include our values
	if !strings.Contains(cmd.Version, "v9.9.9") {
		t.Fatalf("expected version to contain v9.9.9, got %s", cmd.Version)
	}

	// Ensure common subcommands exist
	names := []string{"deploy", "rotate-key", "audit", "version"}
	for _, n := range names {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == n {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected subcommand %s to be registered", n)
		}
	}
}

func TestRunParallelTasks_PrintsResultsAndLogs(t *testing.T) {
	// Initialize i18n and in-memory DB so logging doesn't fail
	i18n.Init("en")
	if err := db.InitDB("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.InitDB failed: %v", err)
	}

	// Prepare accounts
	accounts := []model.Account{
		{ID: 1, Username: "u1", Hostname: "h1"},
		{ID: 2, Username: "u2", Hostname: "h2"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := parallelTask{
		name:       "test",
		startMsg:   "START",
		successMsg: "OK %s",
		failMsg:    "FAIL %s: %s",
		successLog: "SLOG",
		failLog:    "FLOG",
		taskFunc: func(a model.Account) error {
			if a.ID == 2 {
				return &os.PathError{Op: "test", Path: "x", Err: os.ErrInvalid}
			}
			return nil
		},
	}

	runParallelTasks(accounts, task)

	// Close writer and restore stdout
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = old

	out := buf.String()
	if !strings.Contains(out, "START") {
		t.Fatalf("expected START in output, got: %s", out)
	}
	if !strings.Contains(out, "OK u1@h1") && !strings.Contains(out, "OK u1") {
		t.Fatalf("expected success message for first account, got: %s", out)
	}
	if !strings.Contains(out, "FAIL") {
		t.Fatalf("expected a failure message for second account, got: %s", out)
	}
}
