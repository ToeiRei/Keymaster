// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"errors"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func TestAssignAndUnassignKeyOps(t *testing.T) {
	keys := []model.PublicKey{{ID: 1, Comment: "one"}, {ID: 2, Comment: "two"}}
	var calledAssign bool
	assignFn := func(kid, aid int) error {
		if kid != 2 || aid != 10 {
			return errors.New("unexpected args")
		}
		calledAssign = true
		return nil
	}

	assigned := make(map[int]struct{})
	var err error
	assigned, err = AssignKeyToAccount(keys, assigned, 2, 10, assignFn)
	if err != nil {
		t.Fatalf("unexpected assign error: %v", err)
	}
	if !calledAssign {
		t.Fatalf("assign func not called")
	}
	if _, ok := assigned[2]; !ok {
		t.Fatalf("key not present after assign")
	}

	var calledUnassign bool
	unassignFn := func(kid, aid int) error {
		if kid != 2 || aid != 10 {
			return errors.New("unexpected args")
		}
		calledUnassign = true
		return nil
	}
	assigned, err = UnassignKeyFromAccount(assigned, 2, 10, unassignFn)
	if err != nil {
		t.Fatalf("unexpected unassign error: %v", err)
	}
	if !calledUnassign {
		t.Fatalf("unassign func not called")
	}
	if _, ok := assigned[2]; ok {
		t.Fatalf("key still present after unassign")
	}
}
