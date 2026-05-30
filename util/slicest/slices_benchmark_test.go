// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package slicest

import "testing"

func benchmarkFlattenInput(groups, groupSize int) [][]int {
	in := make([][]int, groups)
	for i := 0; i < groups; i++ {
		row := make([]int, groupSize)
		for j := 0; j < groupSize; j++ {
			row[j] = i*groupSize + j
		}
		in[i] = row
	}
	return in
}

func BenchmarkFlatten(b *testing.B) {
	cases := []struct {
		name      string
		groups    int
		groupSize int
	}{
		{name: "small", groups: 8, groupSize: 8},
		{name: "medium", groups: 64, groupSize: 16},
		{name: "large", groups: 256, groupSize: 32},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			input := benchmarkFlattenInput(tc.groups, tc.groupSize)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Flatten(input)
			}
		})
	}
}

func BenchmarkFlatten2(b *testing.B) {
	cases := []struct {
		name      string
		groups    int
		groupSize int
	}{
		{name: "small", groups: 8, groupSize: 8},
		{name: "medium", groups: 64, groupSize: 16},
		{name: "large", groups: 256, groupSize: 32},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			input := benchmarkFlattenInput(tc.groups, tc.groupSize)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Flatten2(input)
			}
		})
	}
}
