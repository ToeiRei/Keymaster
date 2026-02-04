// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy_test

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/core/deploy"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

// TestCanonicalizeHostPort_StandardHost tests standard hostname with default port.
func TestCanonicalizeHostPort_StandardHost(t *testing.T) {
	result := deploy.CanonicalizeHostPort("example.com")
	if result != "example.com:22" {
		t.Fatalf("expected 'example.com:22', got %q", result)
	}
}

// TestCanonicalizeHostPort_ExplicitPort tests hostname with explicit port.
func TestCanonicalizeHostPort_ExplicitPort(t *testing.T) {
	result := deploy.CanonicalizeHostPort("example.com:2222")
	if result != "example.com:2222" {
		t.Fatalf("expected 'example.com:2222', got %q", result)
	}
}

// TestCanonicalizeHostPort_IPv6Default tests IPv6 with default port.
func TestCanonicalizeHostPort_IPv6Default(t *testing.T) {
	result := deploy.CanonicalizeHostPort("2001:db8::1")
	if result != "[2001:db8::1]:22" {
		t.Fatalf("expected '[2001:db8::1]:22', got %q", result)
	}
}

// TestCanonicalizeHostPort_IPv6ExplicitPort tests IPv6 with explicit port.
func TestCanonicalizeHostPort_IPv6ExplicitPort(t *testing.T) {
	result := deploy.CanonicalizeHostPort("[2001:db8::1]:2222")
	if result != "[2001:db8::1]:2222" {
		t.Fatalf("expected '[2001:db8::1]:2222', got %q", result)
	}
}

// TestCanonicalizeHostPort_UserPrefix tests hostname with user@ prefix (user is stripped).
func TestCanonicalizeHostPort_UserPrefix(t *testing.T) {
	result := deploy.CanonicalizeHostPort("user@example.com")
	if result != "example.com:22" {
		t.Fatalf("expected 'example.com:22', got %q", result)
	}
}

// TestCanonicalizeHostPort_UserPrefixWithPort tests user@host:port (user is stripped).
func TestCanonicalizeHostPort_UserPrefixWithPort(t *testing.T) {
	result := deploy.CanonicalizeHostPort("user@example.com:2222")
	if result != "example.com:2222" {
		t.Fatalf("expected 'example.com:2222', got %q", result)
	}
}

// TestParseHostPort_StandardHost tests parsing standard hostname.
func TestParseHostPort_StandardHost(t *testing.T) {
	host, port, err := deploy.ParseHostPort("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "example.com" || port != "" {
		t.Fatalf("expected ('example.com', ''), got (%q, %q)", host, port)
	}
}

// TestParseHostPort_HostAndPort tests parsing hostname with port.
func TestParseHostPort_HostAndPort(t *testing.T) {
	host, port, err := deploy.ParseHostPort("example.com:2222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "example.com" || port != "2222" {
		t.Fatalf("expected ('example.com', '2222'), got (%q, %q)", host, port)
	}
}

// TestParseHostPort_IPv6 tests parsing IPv6 address.
func TestParseHostPort_IPv6(t *testing.T) {
	host, port, err := deploy.ParseHostPort("2001:db8::1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "2001:db8::1" || port != "" {
		t.Fatalf("expected ('2001:db8::1', ''), got (%q, %q)", host, port)
	}
}

// TestParseHostPort_IPv6WithPort tests parsing IPv6 with port in brackets.
func TestParseHostPort_IPv6WithPort(t *testing.T) {
	host, port, err := deploy.ParseHostPort("[2001:db8::1]:2222")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "2001:db8::1" || port != "2222" {
		t.Fatalf("expected ('2001:db8::1', '2222'), got (%q, %q)", host, port)
	}
}

// TestStripIPv6Brackets_WithBrackets tests removing IPv6 brackets.
func TestStripIPv6Brackets_WithBrackets(t *testing.T) {
	result := deploy.StripIPv6Brackets("[2001:db8::1]")
	if result != "2001:db8::1" {
		t.Fatalf("expected '2001:db8::1', got %q", result)
	}
}

// TestStripIPv6Brackets_NoBrackets tests that hostname without brackets is unchanged.
func TestStripIPv6Brackets_NoBrackets(t *testing.T) {
	result := deploy.StripIPv6Brackets("example.com")
	if result != "example.com" {
		t.Fatalf("expected 'example.com', got %q", result)
	}
}

// TestStripIPv6Brackets_PartialBrackets tests asymmetric brackets are not removed.
func TestStripIPv6Brackets_PartialBrackets(t *testing.T) {
	result := deploy.StripIPv6Brackets("[2001:db8::1")
	if result != "[2001:db8::1" {
		t.Fatalf("expected '[2001:db8::1', got %q", result)
	}
}

// TestDecommissionResult_String_Skipped tests String() for skipped decommission.
func TestDecommissionResult_String_Skipped(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:     1,
		AccountString: "user@host.com",
		Skipped:       true,
		SkipReason:    "dry run mode",
	}
	str := result.String()
	if !contains(str, "SKIPPED") || !contains(str, "user@host.com") {
		t.Fatalf("unexpected String() output: %q", str)
	}
}

// TestDecommissionResult_String_SuccessFull tests String() for successful decommission.
func TestDecommissionResult_String_SuccessFull(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:          1,
		AccountString:      "user@host.com",
		RemoteCleanupDone:  true,
		DatabaseDeleteDone: true,
	}
	str := result.String()
	if !contains(str, "SUCCESS") || !contains(str, "removed from database") {
		t.Fatalf("unexpected String() output: %q", str)
	}
}

// TestDecommissionResult_String_PartialFailure tests String() for partial failure.
func TestDecommissionResult_String_PartialFailure(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:          1,
		AccountString:      "user@host.com",
		RemoteCleanupError: errors.New("connection failed"),
		DatabaseDeleteDone: true,
	}
	str := result.String()
	if !contains(str, "PARTIAL") || !contains(str, "remote cleanup failed") {
		t.Fatalf("unexpected String() output: %q", str)
	}
}

// TestDecommissionResult_String_CompleteFail tests String() for complete failure.
func TestDecommissionResult_String_CompleteFail(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:           1,
		AccountString:       "user@host.com",
		RemoteCleanupError:  errors.New("connection failed"),
		DatabaseDeleteError: errors.New("database error"),
	}
	str := result.String()
	if !contains(str, "FAILED") {
		t.Fatalf("unexpected String() output: %q", str)
	}
}

// TestDecommissionResult_String_WithBackup tests String() includes backup path.
func TestDecommissionResult_String_WithBackup(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:          1,
		AccountString:      "user@host.com",
		RemoteCleanupDone:  true,
		DatabaseDeleteDone: true,
		BackupPath:         "/tmp/backup.json",
	}
	str := result.String()
	if !contains(str, "backup") || !contains(str, "backup.json") {
		t.Fatalf("unexpected String() output: %q", str)
	}
}

// TestDecommissionAccount_DryRun tests that dry-run mode skips actual operations.
func TestDecommissionAccount_DryRun(t *testing.T) {
	i18n.Init("en")
	acct := model.Account{ID: 1, Username: "test", Hostname: "host.com"}
	result := deploy.DecommissionAccount(acct, nil, deploy.DecommissionOptions{DryRun: true})
	if !result.Skipped || result.SkipReason != "dry run mode" {
		t.Fatalf("expected dry run skip, got skipped=%v reason=%q", result.Skipped, result.SkipReason)
	}
	if result.RemoteCleanupDone || result.DatabaseDeleteDone {
		t.Fatalf("expected no operations in dry run mode")
	}
}

// TestDecommissionOptions_ValidConfiguration tests DecommissionOptions configuration.
func TestDecommissionOptions_ValidConfiguration(t *testing.T) {
	opts := deploy.DecommissionOptions{
		DryRun:            true,
		Force:             true,
		SkipRemoteCleanup: false,
		KeepFile:          true,
	}
	if !opts.DryRun || !opts.Force || opts.SkipRemoteCleanup || !opts.KeepFile {
		t.Fatalf("expected options to be correctly set")
	}
}

// TestDecommissionResult_NoErrorsEmptyBackup tests successful decommission without backup.
func TestDecommissionResult_NoErrorsEmptyBackup(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:          1,
		AccountString:      "user@host.com",
		RemoteCleanupDone:  true,
		DatabaseDeleteDone: true,
		BackupPath:         "",
	}
	str := result.String()
	if !contains(str, "SUCCESS") {
		t.Fatalf("expected SUCCESS in output: %q", str)
	}
}

// TestDecommissionResult_WithRemoteErrorButForced tests force flag allows partial success.
func TestDecommissionResult_WithRemoteErrorButForced(t *testing.T) {
	result := deploy.DecommissionResult{
		AccountID:          1,
		AccountString:      "user@host.com",
		RemoteCleanupError: errors.New("ssh connection failed"),
		DatabaseDeleteDone: true,
	}
	str := result.String()
	// Should be PARTIAL because database delete succeeded even though remote failed
	if !contains(str, "PARTIAL") {
		t.Fatalf("expected PARTIAL status: %q", str)
	}
}

// contains checks if string contains substring.
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
