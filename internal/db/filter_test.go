package db

import (
	"reflect"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

func mkKey(alg, data, comment string) model.PublicKey {
	return model.PublicKey{Algorithm: alg, KeyData: data, Comment: comment}
}

func TestFilterPublicKeysByTokens(t *testing.T) {
	k1 := mkKey("ssh-ed25519", "AAAAB3NzaEd25519data", "alice@example.com")
	k2 := mkKey("ssh-rsa", "AAAAB3NzaRSAdata", "bob@host")
	k3 := mkKey("ecdsa-sha2-nistp256", "ECdata123", "service-key")

	keys := []model.PublicKey{k1, k2, k3}

	tests := []struct {
		name   string
		tokens []string
		want   []model.PublicKey
	}{
		{name: "no tokens - nil", tokens: nil, want: keys},
		{name: "no tokens - empty", tokens: []string{}, want: keys},
		{name: "match comment", tokens: []string{"alice"}, want: []model.PublicKey{k1}},
		{name: "match algorithm", tokens: []string{"rsa"}, want: []model.PublicKey{k2}},
		{name: "match keydata", tokens: []string{"ecdata"}, want: []model.PublicKey{k3}},
		{name: "multiple tokens", tokens: []string{"alice", "ed25519"}, want: []model.PublicKey{k1}},
		{name: "case insensitive", tokens: []string{"ALICE"}, want: []model.PublicKey{k1}},
		{name: "spaces and empty token", tokens: []string{" ", "bob"}, want: []model.PublicKey{k2}},
		{name: "no matches", tokens: []string{"nomatch"}, want: []model.PublicKey{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterPublicKeysByTokens(keys, tt.tokens)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FilterPublicKeysByTokens(%v) = %v, want %v", tt.tokens, got, tt.want)
			}
		})
	}
}
