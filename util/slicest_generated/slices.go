package slicestgenerated

import "slices"

// Get
func Get[T any, S ~[]T](s S, idx int) T {
	if idx < 0 {
		idx += len(s)
	}
	return s[idx]
}

// Put
func Put[T any, S ~[]T](s S, idx int, val T) {
	if idx < 0 {
		idx += len(s)
	}
	s[idx] = val
}

// Append
func Append[T any, S ~[]T](s S, vals ...T) S {
	return append(s, vals...)
}

// Insert
func Insert[S ~[]E, E any](s S, idx int, vals ...E) S {
	if idx < 0 {
		idx += len(s)
	}
	return slices.Insert(s, idx, vals...)
}

// Delete
func Delete[S ~[]E, E any](s S, i, j int) S {
	return RemoveTo(s, i, j)
}

// Replace
func Replace[S ~[]E, E any](s S, i, j int, v ...E) S {
	return ReplaceTo(s, i, j, v...)
}

// ReplaceN
func ReplaceN[T any, S ~[]T](s S, idx, n int, vals ...T) S {
	if idx < 0 {
		idx += len(s)
	}
	return slices.Replace(s, idx, idx+n, vals...)
}

// ReplaceTo
func ReplaceTo[T any, S ~[]T](s S, from, to int, vals ...T) S {
	if from < 0 {
		from += len(s)
	}
	if to < 0 {
		to += len(s)
	} else if to == 0 {
		to = len(s)
	}
	return slices.Replace(s, from, to, vals...)
}

// RemoveN
func RemoveN[T any, S ~[]T](s S, idx, n int) S {
	if idx < 0 {
		idx += len(s)
	}
	return slices.Delete(s, idx, idx+n)
}

// RemoveTo
func RemoveTo[T any, S ~[]T](s S, from, to int) S {
	if from < 0 {
		from += len(s)
	}
	if to < 0 {
		to += len(s)
	} else if to == 0 {
		to = len(s)
	}
	return slices.Delete(s, from, to)
}

// Prefix
func Prefix[T any, S ~[]T](s S, idx int) S {
	if idx < 0 {
		idx += len(s)
	}
	return s[:idx]
}

// PrefixFunc
func PrefixFuncX[T any, S ~[]T](s S, fn func(T) (bool, error)) (S, error) {
	for i, v := range s {
		ok, err := fn(v)
		if err != nil {
			return nil, err
		}
		if !ok {
			return s[:i], nil
		}
	}
	return s, nil
}
func PrefixFunc[T any, S ~[]T](s S, fn func(T) bool) S {
	result, _ := PrefixFuncX(s, func(t T) (bool, error) {
		return fn(t), nil
	})
	return result
}

// Suffix
func Suffix[T any, S ~[]T](s S, idx int) S {
	if idx < 0 {
		idx += len(s)
	}
	return s[idx:]
}

// Rindex
func Rindex[T comparable, S ~[]T](s S, v T) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == v {
			return i
		}
	}
	return -1
}

// RindexFunc
func RindexFuncX[T any, S ~[]T](s S, fn func(T) (bool, error)) (int, error) {
	for i := len(s) - 1; i >= 0; i-- {
		ok, err := fn(s[i])
		if err != nil {
			return -1, err
		}
		if ok {
			return i, nil
		}
	}
	return -1, nil
}
func RindexFunc[T any, S ~[]T](s S, fn func(T) bool) int {
	result, _ := RindexFuncX(s, func(t T) (bool, error) {
		return fn(t), nil
	})
	return result
}

// SuffixFunc
func SuffixFuncX[T any, S ~[]T](s S, fn func(T) (bool, error)) (S, error) {
	for i := len(s) - 1; i >= 0; i-- {
		ok, err := fn(s[i])
		if err != nil {
			return nil, err
		}
		if !ok {
			return s[i+1:], nil
		}
	}
	return s, nil
}
func SuffixFunc[T any, S ~[]T](s S, fn func(T) bool) S {
	result, _ := SuffixFuncX(s, func(t T) (bool, error) {
		return fn(t), nil
	})
	return result
}

// SliceN
func SliceN[T any, S ~[]T](s S, idx, n int) S {
	if idx < 0 {
		idx += len(s)
	}
	return s[idx : idx+n]
}

// SliceTo
func SliceTo[T any, S ~[]T](s S, from, to int) S {
	if from < 0 {
		from += len(s)
	}
	if to < 0 {
		to += len(s)
	} else if to == 0 {
		to = len(s)
	}
	return s[from:to]
}

// Each
func EachXI[T any, S ~[]T](s S, fn func(int, T) error) error {
	for i, v := range s {
		if err := fn(i, v); err != nil {
			return err
		}
	}
	return nil
}
func EachX[T any, S ~[]T](s S, fn func(T) error) error {
	return EachXI(s, func(_ int, t T) error {
		return fn(t)
	})
}
func EachI[T any, S ~[]T](s S, fn func(int, T)) {
	_ = EachXI(s, func(i int, t T) error {
		fn(i, t)
		return nil
	})
}
func Each[T any, S ~[]T](s S, fn func(T)) {
	_ = EachXI(s, func(_ int, t T) error {
		fn(t)
		return nil
	})
}

// Map
func MapXI[T, U any, S ~[]T](s S, fn func(int, T) (U, error)) ([]U, error) {
	if len(s) == 0 {
		return nil, nil
	}
	result := make([]U, 0, len(s))
	for i, v := range s {
		out, err := fn(i, v)
		if err != nil {
			return nil, err
		}
		result = append(result, out)
	}
	return result, nil
}
func MapX[T, U any, S ~[]T](s S, fn func(T) (U, error)) ([]U, error) {
	return MapXI(s, func(_ int, t T) (U, error) {
		return fn(t)
	})
}
func MapI[T, U any, S ~[]T](s S, fn func(int, T) U) []U {
	result, _ := MapXI(s, func(i int, t T) (U, error) {
		return fn(i, t), nil
	})
	return result
}
func Map[T, U any, S ~[]T](s S, fn func(T) U) []U {
	result, _ := MapXI(s, func(_ int, t T) (U, error) {
		return fn(t), nil
	})
	return result
}

// Accum
func AccumX[T any, S ~[]T](s S, fn func(T, T) (T, error)) (T, error) {
	if len(s) == 0 {
		var zero T
		return zero, nil
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		var err error
		result, err = fn(result, s[i])
		if err != nil {
			return result, err
		}
	}
	return result, nil
}
func Accum[T any, S ~[]T](s S, fn func(T, T) T) T {
	result, _ := AccumX(s, func(a, b T) (T, error) {
		return fn(a, b), nil
	})
	return result
}

// Filter
func FilterX[T any, S ~[]T](s S, fn func(T) (bool, error)) (S, error) {
	var result S
	for _, v := range s {
		ok, err := fn(v)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, v)
		}
	}
	return result, nil
}
func Filter[T any, S ~[]T](s S, fn func(T) bool) S {
	result, _ := FilterX(s, func(t T) (bool, error) {
		return fn(t), nil
	})
	return result
}

// Group
func GroupX[T any, K comparable, S ~[]T](s S, fn func(T) (K, error)) (map[K]S, error) {
	result := make(map[K]S)
	for _, v := range s {
		key, err := fn(v)
		if err != nil {
			return nil, err
		}
		result[key] = append(result[key], v)
	}
	return result, nil
}
func Group[T any, K comparable, S ~[]T](s S, fn func(T) K) map[K]S {
	result, _ := GroupX(s, func(t T) (K, error) {
		return fn(t), nil
	})
	return result
}

// Rotate
func Rotate[T any, S ~[]T](s S, n int) {
	if n < 0 {
		n = -n
		n %= len(s)
		n = len(s) - n
	} else {
		n %= len(s)
	}
	if n == 0 {
		return
	}
	tmp := make([]T, n)
	copy(tmp, s[len(s)-n:])
	copy(s[n:], s)
	copy(s, tmp)
}
