// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import "cmp"

func Clamp[T cmp.Ordered](_min, _wanted, _max T) T {
	return min(max(_min, _wanted), _max)
}
