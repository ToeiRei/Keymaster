package db

import (
	"errors"
	"testing"
)

func TestMapDBError_DuplicateStrings(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"mysql duplicate entry", errors.New("Error 1062: Duplicate entry 'x' for key 'PRIMARY'")},
		{"postgres unique violation", errors.New("pq: duplicate key value violates unique constraint \"users_pkey\" (SQLSTATE 23505)")},
		{"sqlite unique constraint", errors.New("UNIQUE constraint failed: public_keys.comment")},
		{"generic duplicate word", errors.New("duplicate row")},
		{"unique word present", errors.New("column has unique constraint")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mapped := MapDBError(c.err)
			if !errors.Is(mapped, ErrDuplicate) {
				t.Fatalf("expected ErrDuplicate for case %s, got: %v", c.name, mapped)
			}
		})
	}
}

func TestMapDBError_NonDuplicatePassthrough(t *testing.T) {
	e := errors.New("some network error")
	mapped := MapDBError(e)
	if mapped == nil {
		t.Fatalf("expected non-nil error for non-duplicate input")
	}
	if errors.Is(mapped, ErrDuplicate) {
		t.Fatalf("did not expect ErrDuplicate for non-duplicate error")
	}
	if mapped.Error() != e.Error() {
		t.Fatalf("expected original error to be returned unchanged, got: %v", mapped)
	}
}
