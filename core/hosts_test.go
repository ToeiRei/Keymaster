// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"reflect"
	"testing"

	"github.com/toeirei/keymaster/core/model"
)

func TestBuildAccountsByHost(t *testing.T) {
	tests := []struct {
		name     string
		accounts []model.Account
		want     map[string][]model.Account
	}{
		{
			name:     "empty accounts",
			accounts: []model.Account{},
			want:     map[string][]model.Account{},
		},
		{
			name: "single host",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: "host1.example.com"},
				{ID: 2, Username: "bob", Hostname: "host1.example.com"},
			},
			want: map[string][]model.Account{
				"host1.example.com": {
					{ID: 1, Username: "alice", Hostname: "host1.example.com"},
					{ID: 2, Username: "bob", Hostname: "host1.example.com"},
				},
			},
		},
		{
			name: "multiple hosts",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: "host1.example.com"},
				{ID: 2, Username: "bob", Hostname: "host2.example.com"},
				{ID: 3, Username: "charlie", Hostname: "host1.example.com"},
			},
			want: map[string][]model.Account{
				"host1.example.com": {
					{ID: 1, Username: "alice", Hostname: "host1.example.com"},
					{ID: 3, Username: "charlie", Hostname: "host1.example.com"},
				},
				"host2.example.com": {
					{ID: 2, Username: "bob", Hostname: "host2.example.com"},
				},
			},
		},
		{
			name: "account with no hostname",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: ""},
				{ID: 2, Username: "bob", Hostname: "host1.example.com"},
			},
			want: map[string][]model.Account{
				unknownHostLabel: {
					{ID: 1, Username: "alice", Hostname: ""},
				},
				"host1.example.com": {
					{ID: 2, Username: "bob", Hostname: "host1.example.com"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAccountsByHost(tt.accounts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildAccountsByHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueHosts(t *testing.T) {
	tests := []struct {
		name     string
		accounts []model.Account
		want     []string
	}{
		{
			name:     "empty accounts",
			accounts: []model.Account{},
			want:     []string{},
		},
		{
			name: "single host",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: "host1.example.com"},
				{ID: 2, Username: "bob", Hostname: "host1.example.com"},
			},
			want: []string{"host1.example.com"},
		},
		{
			name: "multiple hosts sorted",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: "zebra.example.com"},
				{ID: 2, Username: "bob", Hostname: "alpha.example.com"},
				{ID: 3, Username: "charlie", Hostname: "beta.example.com"},
			},
			want: []string{"alpha.example.com", "beta.example.com", "zebra.example.com"},
		},
		{
			name: "duplicate hosts",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: "host1.example.com"},
				{ID: 2, Username: "bob", Hostname: "host1.example.com"},
				{ID: 3, Username: "charlie", Hostname: "host2.example.com"},
			},
			want: []string{"host1.example.com", "host2.example.com"},
		},
		{
			name: "account with no hostname",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: ""},
				{ID: 2, Username: "bob", Hostname: "host1.example.com"},
			},
			want: []string{"host1.example.com", unknownHostLabel},
		},
		{
			name: "whitespace-only hostname treated as empty",
			accounts: []model.Account{
				{ID: 1, Username: "alice", Hostname: "   "},
				{ID: 2, Username: "bob", Hostname: "host1.example.com"},
			},
			want: []string{"host1.example.com", unknownHostLabel},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueHosts(tt.accounts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueHosts() = %v, want %v", got, tt.want)
			}
		})
	}
}
