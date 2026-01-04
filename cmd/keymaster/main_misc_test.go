package main

import (
	"os"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/toeirei/keymaster/internal/model"
)

func TestFindAccountByIdentifier_ID_UserAtHost_Label_And_NotFound(t *testing.T) {
	accounts := []model.Account{
		{ID: 10, Username: "alice", Hostname: "example.com", Label: "web-1"},
		{ID: 20, Username: "bob", Hostname: "host.local", Label: "db-1"},
	}

	// By ID
	a, err := findAccountByIdentifier("10", accounts)
	if err != nil || a == nil || a.ID != 10 {
		t.Fatalf("expected to find account ID 10, got %v err=%v", a, err)
	}

	// By user@host (case-insensitive)
	a, err = findAccountByIdentifier("Bob@Host.Local", accounts)
	if err != nil || a == nil || a.ID != 20 {
		t.Fatalf("expected to find bob by user@host, got %v err=%v", a, err)
	}

	// By label (case-insensitive)
	a, err = findAccountByIdentifier("WEB-1", accounts)
	if err != nil || a == nil || a.ID != 10 {
		t.Fatalf("expected to find web-1 by label, got %v err=%v", a, err)
	}

	// Not found
	a, err = findAccountByIdentifier("not-there", accounts)
	if err == nil || a != nil {
		t.Fatalf("expected not found error, got %v err=%v", a, err)
	}
}

func TestWriteAndReadCompressedBackup_RoundTrip(t *testing.T) {
	// Prepare backup data
	data := &model.BackupData{
		SchemaVersion: 1,
		Accounts:      []model.Account{{ID: 1, Username: "u", Hostname: "h"}},
	}

	tmp, err := os.CreateTemp("", "km-backup-*.json.zst")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)

	if err := writeCompressedBackup(name, data); err != nil {
		t.Fatalf("writeCompressedBackup failed: %v", err)
	}

	// Validate file exists and looks like zstd (has magic bytes)
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("open backup failed: %v", err)
	}
	defer f.Close()
	zr, err := zstd.NewReader(f)
	if err != nil {
		t.Fatalf("zstd.NewReader failed: %v", err)
	}
	zr.Close()

	// Read via helper
	got, err := readCompressedBackup(name)
	if err != nil {
		t.Fatalf("readCompressedBackup failed: %v", err)
	}
	if got == nil || got.SchemaVersion != data.SchemaVersion || len(got.Accounts) != 1 || got.Accounts[0].Username != "u" {
		t.Fatalf("unexpected backup roundtrip result: %+v", got)
	}
}
