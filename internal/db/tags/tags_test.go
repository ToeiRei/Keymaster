package tags

import (
	"testing"
	// "github.com/toeirei/keymaster/internal/db/tags"
)

// Test Split/Join roundtrip and validation behavior.
func TestSplitJoinValidate(t *testing.T) {
	// Join
	joined := JoinTags([]string{"prod", "web"})
	if joined != "|prod|web|" {
		t.Fatalf("unexpected join: %s", joined)
	}

	// Split good
	parts, err := SplitTags(joined)
	if err != nil {
		t.Fatalf("SplitTags failed: %v", err)
	}
	if len(parts) != 2 || parts[0] != "prod" || parts[1] != "web" {
		t.Fatalf("unexpected split parts: %+v", parts)
	}

	// Split bad (missing delimiters)
	if _, err := SplitTags("prod|web"); err == nil {
		t.Fatalf("expected error for missing prefix/suffix")
	}

	// ValidateTagMatcher good
	if err := ValidateTagMatcher("prod-01"); err != nil {
		t.Fatalf("expected prod-01 to validate: %v", err)
	}

	// ValidateTagMatcher bad (space)
	if err := ValidateTagMatcher("bad tag"); err == nil {
		t.Fatalf("expected validation failure for 'bad tag'")
	}
}
