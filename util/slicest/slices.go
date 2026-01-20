package slicest

// Conversion

func ToMap[T any, K comparable, V any, S ~[]T](s S, fn func(T) (K, V)) map[K]V {
	result := make(map[K]V, len(s))
	for _, t := range s {
		k, v := fn(t)
		result[k] = v
	}
	return result
}

// Reduce

// Reduce reduces slice S to type U.
func Reduce[T any, S ~[]T, U any](s S, fn func(T, U) U) U {
	return ReduceI(s, func(_ int, t T, u U) U {
		return fn(t, u)
	})
}

// ReduceI reduces slice S to type U.
// - I: Provides index to callback.
func ReduceI[T any, S ~[]T, U any](s S, fn func(int, T, U) U) U {
	result, _ := ReduceXI(s, func(i int, t T, u U) (U, error) {
		return fn(i, t, u), nil
	})
	return result
}

// ReduceXI reduces slice S to type U with error propagation.
// - X: Stops on failure and returns error.
// - I: Provides index to callback.
func ReduceXI[T any, S ~[]T, U any](s S, fn func(int, T, U) (U, error)) (U, error) {
	var zero U
	return ReduceXDI(s, zero, fn)
}

// ReduceDI reduces slice S to type U using explicit initial value.
// - D: Uses init parameter as starting accumulator.
// - I: Provides index to callback.
func ReduceDI[T any, S ~[]T, U any](s S, init U, fn func(int, T, U) U) U {
	result, _ := ReduceXDI(s, init, func(i int, t T, u U) (U, error) {
		return fn(i, t, u), nil
	})
	return result
}

// ReduceXDI reduces slice S to type U with initial value and error propagation.
// - X: Stops on failure and returns error.
// - D: Uses init parameter as starting accumulator.
// - I: Provides index to callback.
func ReduceXDI[T any, S ~[]T, U any](s S, init U, fn func(int, T, U) (U, error)) (U, error) {
	var zero U
	for i, t := range s {
		var err error
		init, err = fn(i, t, init)
		if err != nil {
			return zero, err
		}
	}
	return init, nil
}

// ReduceX reduces slice S to type U with error propagation.
// - X: Stops on failure and returns error.
func ReduceX[T any, S ~[]T, U any](s S, fn func(T, U) (U, error)) (U, error) {
	return ReduceXI(s, func(_ int, t T, u U) (U, error) {
		return fn(t, u)
	})
}

// ReduceXD reduces slice S to type U with initial value and error propagation.
// - X: Stops on failure and returns error.
// - D: Uses init parameter as starting accumulator.
func ReduceXD[T any, S ~[]T, U any](s S, init U, fn func(T, U) (U, error)) (U, error) {
	return ReduceXDI(s, init, func(_ int, t T, u U) (U, error) {
		return fn(t, u)
	})
}

// ReduceD reduces slice S to type U using explicit initial value.
// - D: Uses init parameter as starting accumulator.
func ReduceD[T any, S ~[]T, U any](s S, init U, fn func(T, U) U) U {
	result, _ := ReduceXD(s, init, func(t T, u U) (U, error) {
		return fn(t, u), nil
	})
	return result
}

// Map

func MapXI[T, U any, S ~[]T](s S, fn func(int, T) (U, error)) ([]U, error) {
	result := make([]U, len(s))
	for i, v := range s {
		out, err := fn(i, v)
		if err != nil {
			return nil, err
		}
		result[i] = out
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
