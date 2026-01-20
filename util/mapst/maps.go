package mapst

// Map

func Map[K comparable, V any, R any, M ~map[K]V](m M, fn func(K, V) R) map[K]R {
	result, _ := Mapx(m, func(k K, v V) (R, error) {
		return fn(k, v), nil
	})
	return result
}

func Mapx[K comparable, V any, R any, M ~map[K]V](m M, fn func(K, V) (R, error)) (map[K]R, error) {
	if len(m) == 0 {
		return nil, nil
	}
	result := make(map[K]R, len(m))
	for k, v := range m {
		r, err := fn(k, v)
		if err != nil {
			return nil, err
		}
		result[k] = r
	}
	return result, nil
}

// Each

func Each[K comparable, V any, M ~map[K]V](m M, fn func(K, V)) {
	Eachx(m, func(k K, v V) error {
		fn(k, v)
		return nil
	})
}

func Eachx[K comparable, V any, M ~map[K]V](m M, fn func(K, V) error) error {
	for k, v := range m {
		err := fn(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// Filter

func Filter[K comparable, V any, M ~map[K]V](m M, fn func(K, V) bool) M {
	result, _ := Filterx(m, func(k K, v V) (bool, error) {
		return fn(k, v), nil
	})
	return result
}

func Filterx[K comparable, V any, M ~map[K]V](m M, fn func(K, V) (bool, error)) (M, error) {
	result := make(M)
	for k, v := range m {
		ok, err := fn(k, v)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		result[k] = v
	}
	return result, nil
}

// Keys

func Keys[K comparable, V any, M ~map[K]V](m M) []K {
	result := make([]K, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Values

func Values[K comparable, V any, M ~map[K]V](m M) []V {
	result := make([]V, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

// Reduce

func Reduce[K comparable, V any, M ~map[K]V, R any](m M, fn func(K, V, R) R) R {
	result, _ := ReduceX(m, func(k K, v V, r R) (R, error) {
		return fn(k, v, r), nil
	})
	return result
}

func ReduceX[K comparable, V any, M ~map[K]V, R any](m M, fn func(K, V, R) (R, error)) (R, error) {
	var result R
	for k, v := range m {
		var err error
		result, err = fn(k, v, result)
		if err != nil {
			var zero R
			return zero, err
		}
	}
	return result, nil
}
