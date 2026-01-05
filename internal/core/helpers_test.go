package core

import "testing"

func TestContainsIgnoreCase(t *testing.T) {
	cases := []struct {
		s, sub string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello", "WORLD", false},
		{"", "", true},
		{"abc", "", true},
		{"abc", "d", false},
	}
	for _, c := range cases {
		if got := ContainsIgnoreCase(c.s, c.sub); got != c.want {
			t.Fatalf("ContainsIgnoreCase(%q,%q) = %v; want %v", c.s, c.sub, got, c.want)
		}
	}
}
