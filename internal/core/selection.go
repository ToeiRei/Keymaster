// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

// EnsureCursorInView computes the Y offset for a viewport such that the
// given cursor index is visible. It implements edge-scrolling: only when
// the cursor moves above the top or below the bottom of the visible area
// will the offset change.
func EnsureCursorInView(cursor, yOffset, height int) int {
	top := yOffset
	bottom := top + height - 1

	if cursor < top {
		return cursor
	}
	if cursor > bottom {
		return cursor - height + 1
	}
	return yOffset
}
