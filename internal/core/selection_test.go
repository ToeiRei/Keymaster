// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import "testing"

func TestEnsureCursorInView_TopBottomAndNoop(t *testing.T) {
	// viewport height 5
	h := 5

	// when cursor is within view, offset unchanged
	if got := EnsureCursorInView(2, 0, h); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}

	// cursor above top
	if got := EnsureCursorInView(0, 2, h); got != 0 {
		t.Fatalf("expected 0 for cursor above top, got %d", got)
	}

	// cursor below bottom
	if got := EnsureCursorInView(9, 3, h); got != 5 {
		t.Fatalf("expected 5 for cursor below bottom, got %d", got)
	}
}

