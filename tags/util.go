// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

func sliceOverlapping[T comparable, S ~[]T](s1, s2 S) []T {
	m2 := make(map[T]struct{}, len(s2))
	for _, v := range s2 {
		m2[v] = struct{}{}
	}

	var result []T
	for _, t1 := range s1 {
		if _, ok := m2[t1]; ok {
			result = append(result, t1)
			delete(m2, t1)
		}
	}
	return result
}

func sliceUnique[T comparable, S ~[]T](s1, s2 S) []T {
	m1 := make(map[T]struct{}, len(s1))
	m2 := make(map[T]struct{}, len(s2))

	for _, v := range s1 {
		m1[v] = struct{}{}
	}
	for _, v := range s2 {
		m2[v] = struct{}{}
	}

	var result []T
	for t1 := range m1 {
		if _, ok := m2[t1]; !ok {
			result = append(result, t1)
			m2[t1] = struct{}{}
		}
	}
	for t2 := range m2 {
		if _, ok := m1[t2]; !ok {
			result = append(result, t2)
		}
	}
	return result
}

func sliceDeduplicate[T comparable, S ~[]T](s S) []T {
	seen := make(map[T]struct{}, len(s))

	var result S
	for _, t := range s {
		if _, exists := seen[t]; !exists {
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	return result
}

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
