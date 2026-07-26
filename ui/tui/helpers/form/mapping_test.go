// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

import (
	"reflect"
	"testing"
)

type tagged struct {
	Username string `form:"username"`
	Port     int    `form:"port"`
	Untagged string
	hidden   string `form:"hidden"`
}

func TestMapFrom(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    map[string]any
		wantErr bool
	}{
		{"struct", tagged{"root", 22, "skipped", "skipped"}, map[string]any{"username": "root", "port": 22}, false},
		{"pointer to struct", &tagged{"root", 22, "", ""}, map[string]any{"username": "root", "port": 22}, false},
		{"empty struct", struct{}{}, map[string]any{}, false},
		{"map[string]any", map[string]any{"a": 1, "b": nil}, map[string]any{"a": 1, "b": nil}, false},
		{"map[string]string", map[string]string{"a": "x"}, map[string]any{"a": "x"}, false},
		{"nil map", map[string]string(nil), map[string]any{}, false},
		{"non string keys", map[int]string{1: "x"}, nil, true},
		{"not a struct or map", 42, nil, true},
		{"nil", nil, nil, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := mapFrom(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("mapFrom() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && !reflect.DeepEqual(got, test.want) {
				t.Errorf("mapFrom() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestMapToStruct(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
		want  tagged
	}{
		{"assignable", map[string]any{"username": "root", "port": 22}, tagged{"root", 22, "", ""}},
		{"convertible", map[string]any{"port": int64(22)}, tagged{"", 22, "", ""}},
		{"stringified", map[string]any{"username": true}, tagged{"true", 0, "", ""}},
		{"absent key", map[string]any{}, tagged{}},
		{"unknown key", map[string]any{"nope": "x"}, tagged{}},
		{"untagged and unexported are skipped", map[string]any{"Untagged": "x", "hidden": "x"}, tagged{}},
		{"nil value", map[string]any{"username": nil}, tagged{}},
		{"unconvertible value", map[string]any{"port": "twentytwo"}, tagged{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got tagged
			if err := mapTo(test.input, &got); err != nil {
				t.Fatalf("mapTo() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("mapTo() = %#v, want %#v", got, test.want)
			}
		})
	}
}

// an integer must not be converted to its rune, which reflect.Convert would do
func TestMapToStringField(t *testing.T) {
	var got struct {
		Value string `form:"value"`
	}
	if err := mapTo(map[string]any{"value": 65}, &got); err != nil {
		t.Fatalf("mapTo() error = %v", err)
	}
	if got.Value != "65" {
		t.Errorf("mapTo() = %q, want %q", got.Value, "65")
	}
}

func TestMapToMap(t *testing.T) {
	t.Run("allocates and skips valueless items", func(t *testing.T) {
		var got map[string]any
		if err := mapTo(map[string]any{"username": "root", "_cancel": nil}, &got); err != nil {
			t.Fatalf("mapTo() error = %v", err)
		}
		if want := (map[string]any{"username": "root"}); !reflect.DeepEqual(got, want) {
			t.Errorf("mapTo() = %#v, want %#v", got, want)
		}
	})

	t.Run("coerces to a narrower value type", func(t *testing.T) {
		var got map[string]string
		if err := mapTo(map[string]any{"user": "x", "flag": true, "_submit": nil}, &got); err != nil {
			t.Fatalf("mapTo() error = %v", err)
		}
		if want := (map[string]string{"user": "x", "flag": "true"}); !reflect.DeepEqual(got, want) {
			t.Errorf("mapTo() = %#v, want %#v", got, want)
		}
	})

	t.Run("keeps existing entries", func(t *testing.T) {
		got := map[string]string{"kept": "y"}
		if err := mapTo(map[string]any{"user": "x"}, &got); err != nil {
			t.Fatalf("mapTo() error = %v", err)
		}
		if want := (map[string]string{"kept": "y", "user": "x"}); !reflect.DeepEqual(got, want) {
			t.Errorf("mapTo() = %#v, want %#v", got, want)
		}
	})

	t.Run("non string keys", func(t *testing.T) {
		var got map[int]string
		if err := mapTo(map[string]any{"1": "x"}, &got); err == nil {
			t.Error("mapTo() error = nil, want an error")
		}
	})

	t.Run("not a struct or map", func(t *testing.T) {
		var got int
		if err := mapTo(map[string]any{"a": 1}, &got); err == nil {
			t.Error("mapTo() error = nil, want an error")
		}
	})
}

func TestMapRoundTrip(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		want := tagged{"root", 22, "", ""}
		values, err := mapFrom(want)
		if err != nil {
			t.Fatalf("mapFrom() error = %v", err)
		}

		var got tagged
		if err := mapTo(values, &got); err != nil {
			t.Fatalf("mapTo() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("round trip = %#v, want %#v", got, want)
		}
	})

	t.Run("map", func(t *testing.T) {
		want := map[string]string{"user": "root", "passphrase": ""}
		values, err := mapFrom(want)
		if err != nil {
			t.Fatalf("mapFrom() error = %v", err)
		}

		var got map[string]string
		if err := mapTo(values, &got); err != nil {
			t.Fatalf("mapTo() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("round trip = %#v, want %#v", got, want)
		}
	})
}
