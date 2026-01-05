package core

import (
	"errors"
	"testing"
)

func TestAssignKeys_LoopsAndStopsOnError(t *testing.T) {
	calls := 0
	assignFn := func(kid, aid int) error {
		calls++
		if kid == 2 {
			return errors.New("fail on 2")
		}
		return nil
	}
	err := AssignKeys([]int{1, 2, 3}, 10, assignFn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
