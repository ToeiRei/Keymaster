// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

// polyfill: won't be needed as of go 1.26
func new[T any](v T) *T { return &v }

func NewPointer[T any](v T) *T { return &v }

func DerefOrNullValue[T any](p *T) T {
	var null T
	if p != nil {
		return *p
	}
	return null
}
