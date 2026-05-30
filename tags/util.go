// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

func sliceDeduplicateFunc[T any, K comparable, S ~[]T](s S, fn func(t T) K) []T {
	seen := make(map[K]struct{}, len(s))

	var result S
	for _, t := range s {
		key := fn(t)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, t)
		}
	}
	return result
}
