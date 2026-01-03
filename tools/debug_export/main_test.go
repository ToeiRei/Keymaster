package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"
)

// TestMainRuns ensures that the debug export main() runs without panicking
// and prints expected summary lines. It captures stdout and verifies output.
func TestMainRuns(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		// Read output in background
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	// Run main (should not call os.Exit)
	main()

	// Restore stdout and close writer so reader finishes
	_ = w.Close()
	os.Stdout = old

	// Wait for reader goroutine with timeout
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for main output")
	}

	out := buf.String()
	if out == "" {
		t.Fatalf("expected main to print output, got empty string")
	}
	// Basic sanity checks
	if !bytes.Contains(buf.Bytes(), []byte("active accounts:")) {
		t.Fatalf("expected output to contain 'active accounts:', got %q", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("all accounts:")) {
		t.Fatalf("expected output to contain 'all accounts:', got %q", out)
	}
}
