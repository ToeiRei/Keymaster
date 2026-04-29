// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

func NewPointer[T any](v T) *T { return &v }

func NewZero[T any]() (v T) { return }

func DerefOrZeroValue[T any](p *T) T {
	if p != nil {
		return *p
	}
	return NewZero[T]()
}
