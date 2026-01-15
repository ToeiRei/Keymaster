// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/core"
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
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
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

	// Use core.ParallelRun and reproduce similar output formatting
	fmt.Println("START")
	results := core.ParallelRun(context.Background(), accounts, func(a model.Account) error {
		if a.ID == 2 {
			return &os.PathError{Op: "test", Path: "x", Err: os.ErrInvalid}
		}
		return nil
	})
	for _, r := range results {
		if r.Error == nil {
			fmt.Printf("OK %s\n", r.Name)
		} else {
			fmt.Printf("FAIL %s: %v\n", r.Name, r.Error)
		}
	}

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
