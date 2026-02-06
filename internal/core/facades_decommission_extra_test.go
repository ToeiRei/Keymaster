package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/core/model"
)

// reuse fakeStoreForDecom and fakeDMForFacades from facades_decommission_test.go

func TestDecommissionAccounts_GetActiveSystemKey_Error(t *testing.T) {
	st := &fakeStoreForDecom{sys: nil, ferr: errors.New("dbboom")}
	dm := &fakeDMForFacades{}
	_, err := DecommissionAccounts(context.TODO(), []model.Account{{ID: 1}, {ID: 2}}, nil, dm, st, nil)
	if err == nil || !strings.Contains(err.Error(), "get system key") {
		t.Fatalf("expected get system key error, got %v", err)
	}
}

func TestDecommissionAccounts_NoActiveSystemKey(t *testing.T) {
	st := &fakeStoreForDecom{sys: nil, ferr: nil}
	dm := &fakeDMForFacades{}
	_, err := DecommissionAccounts(context.TODO(), []model.Account{{ID: 1}}, nil, dm, st, nil)
	if err == nil || !strings.Contains(err.Error(), "no active system key") {
		t.Fatalf("expected no active system key error, got %v", err)
	}
}

func TestDecommissionAccounts_Bulk_DeployerError(t *testing.T) {
	st := &fakeStoreForDecom{sys: &model.SystemKey{Serial: 1, PublicKey: "k"}}
	dm := &fakeDMForFacades{bErr: errors.New("bulk fail")}
	_, err := DecommissionAccounts(context.TODO(), []model.Account{{ID: 1}, {ID: 2}}, nil, dm, st, nil)
	if err == nil || !strings.Contains(err.Error(), "bulk fail") {
		t.Fatalf("expected bulk deployer error, got %v", err)
	}
}

func TestDecommissionAccounts_Single_DeployerError(t *testing.T) {
	st := &fakeStoreForDecom{sys: &model.SystemKey{Serial: 1, PublicKey: "k"}}
	dm := &fakeDMForFacades{single: ResAndErr{R: DecommissionResult{}, E: errors.New("single fail")}}
	_, err := DecommissionAccounts(context.TODO(), []model.Account{{ID: 7}}, nil, dm, st, nil)
	if err == nil || !strings.Contains(err.Error(), "single fail") {
		t.Fatalf("expected single deployer error, got %v", err)
	}
}

// (no unused compile-time-only check needed here)
