// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package client

func NewOptional[T any](value T) Optional[T] {
	return Optional[T]{&value}
}

type Optional[T any] struct{ Value *T }

func (o Optional[T]) Get() (bool, T) { return o.Value != nil, *o.Value }
func (o Optional[T]) Set(value T)    { *o.Value = value }
func (o Optional[T]) Has() bool      { return o.Value != nil }
func (o Optional[T]) Some() bool     { return o.Value != nil }
func (o Optional[T]) None() bool     { return o.Value == nil }
func (o Optional[T]) Exec(some func(value T), none func()) {
	if o.Has() {
		some(*o.Value)
	} else {
		none()
	}
}
func (o Optional[T]) GetDefault(value T) T {
	if o.Has() {
		return *o.Value
	} else {
		return value
	}
}
func (o Optional[T]) GetFallback(fallback func() T) T {
	if o.Has() {
		return *o.Value
	} else {
		return fallback()
	}
}
