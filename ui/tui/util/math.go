package util

import "cmp"

func Clamp[T cmp.Ordered](_min, _wanted, _max T) T {
	return min(max(_min, _wanted), _max)
}
