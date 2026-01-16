package logging

import (
	"bytes"
	"strings"
	"testing"

	clog "github.com/charmbracelet/log"
)

// TestLoggingHelpers_WriteToBuffer verifies the package helper functions write
// formatted messages to the package-level logger `L`. The test swaps `L` with
// a buffer-backed logger and restores it afterwards.
func TestLoggingHelpers_WriteToBuffer(t *testing.T) {
	var buf bytes.Buffer
	prev := L
	L = clog.New(&buf)
	L.SetLevel(clog.DebugLevel)
	defer func() { L = prev }()

	Debugf("hello %s", "dbg")
	Infof("info %d", 1)
	Warnf("warn")
	Errorf("err %v", "E")

	out := buf.String()
	if !strings.Contains(out, "hello dbg") {
		t.Fatalf("missing debug output; got: %s", out)
	}
	if !strings.Contains(out, "info 1") {
		t.Fatalf("missing info output; got: %s", out)
	}
	if !strings.Contains(out, "warn") {
		t.Fatalf("missing warn output; got: %s", out)
	}
	if !strings.Contains(out, "err E") {
		t.Fatalf("missing error output; got: %s", out)
	}
}
