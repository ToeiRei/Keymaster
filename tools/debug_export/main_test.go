// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

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
	// Capture stderr (charm log writes to stderr)
	oldOut := os.Stdout
	oldErr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	os.Stdout = w
	os.Stderr = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		// Read output in background
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	// Run main (should not call os.Exit)
	main()

	// Restore stdout/stderr and close writer so reader finishes
	_ = w.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	// Wait for reader goroutine with timeout (longer in CI)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
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
