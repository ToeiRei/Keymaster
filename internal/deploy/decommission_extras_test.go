package deploy

import (
	"strings"
	"testing"
)

func TestExtractNonKeymasterContent_VariousCases(t *testing.T) {
	cases := []struct {
		name            string
		in              string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "no keymaster section",
			in:              "line1\nline2\n",
			wantContains:    []string{"line1", "line2"},
			wantNotContains: []string{"Keymaster Managed"},
		},
		{
			name:            "simple keymaster section removed",
			in:              "pre\n# Keymaster Managed Keys\nssh-ed25519 AAAA key-1\n# note\n\npost\n",
			wantContains:    []string{"pre", "post"},
			wantNotContains: []string{"ssh-ed25519", "Keymaster Managed"},
		},
		{
			name:            "keymaster section ends when non-keymaster line found",
			in:              "one\n# Keymaster Managed Keys\nssh-ed25519 AAAA key-1\nNON-KEYMASTER LINE\nafter\n",
			wantContains:    []string{"one", "NON-KEYMASTER LINE", "after"},
			wantNotContains: []string{"ssh-ed25519"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := extractNonKeymasterContent(tc.in)
			for _, want := range tc.wantContains {
				if !strings.Contains(out, want) {
					t.Fatalf("expected output to contain %q, got: %q", want, out)
				}
			}
			for _, not := range tc.wantNotContains {
				if strings.Contains(out, not) {
					t.Fatalf("expected output NOT to contain %q, got: %q", not, out)
				}
			}
		})
	}
}

func TestDecommissionResult_StringVariants(t *testing.T) {
	// Skipped
	r := DecommissionResult{AccountID: 1, AccountString: "u@h", Skipped: true, SkipReason: "dry run"}
	s := r.String()
	if !strings.Contains(s, "SKIPPED") || !strings.Contains(s, "dry run") {
		t.Fatalf("unexpected skipped string: %s", s)
	}

	// Partial with remote error
	r = DecommissionResult{AccountID: 2, AccountString: "u2@h2", RemoteCleanupError: errDummy("boom")}
	s = r.String()
	if !strings.Contains(s, "PARTIAL") || !strings.Contains(s, "remote cleanup failed") {
		t.Fatalf("unexpected partial string: %s", s)
	}

	// Failed with database error and backup
	r = DecommissionResult{AccountID: 3, AccountString: "u3@h3", DatabaseDeleteError: errDummy("dberr"), BackupPath: "/tmp/b"}
	s = r.String()
	if !strings.Contains(s, "FAILED") || !strings.Contains(s, "backup: /tmp/b") {
		t.Fatalf("unexpected failed string: %s", s)
	}
}

// simple error type for tests
type errDummy string

func (e errDummy) Error() string { return string(e) }
