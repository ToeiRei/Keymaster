package core

import (
	"reflect"
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
)

func TestGetSetAccountIsDirty(t *testing.T) {
	a := model.Account{ID: 1, Username: "u", Hostname: "h", IsDirty: false}
	if GetAccountIsDirty(a) {
		t.Fatalf("expected not dirty")
	}
	b := WithAccountIsDirty(a, true)
	if !GetAccountIsDirty(b) {
		t.Fatalf("expected dirty after set")
	}
	// original should remain unchanged
	if GetAccountIsDirty(a) {
		t.Fatalf("original mutated")
	}
}

func TestSetAccountsIsDirtyByID(t *testing.T) {
	accs := []model.Account{{ID: 1}, {ID: 2}, {ID: 3}}
	ids := map[int]struct{}{2: {}, 3: {}}
	out := SetAccountsIsDirtyByID(accs, ids, true)
	if reflect.DeepEqual(accs, out) {
		t.Fatalf("expected new slice different from input")
	}
	for _, a := range out {
		if a.ID == 1 && a.IsDirty {
			t.Fatalf("id=1 should not be dirty")
		}
		if (a.ID == 2 || a.ID == 3) && !a.IsDirty {
			t.Fatalf("id=%d should be dirty", a.ID)
		}
	}
}
