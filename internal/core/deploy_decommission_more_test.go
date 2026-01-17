package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/toeirei/keymaster/internal/model"
)

type fakeRemoteDeployer3 struct {
	getContent []byte
	deployErr  error
	deployed   string
}

func (f *fakeRemoteDeployer3) DeployAuthorizedKeys(content string) error {
	f.deployed = content
	return f.deployErr
}
func (f *fakeRemoteDeployer3) GetAuthorizedKeys() ([]byte, error) { return f.getContent, nil }
func (f *fakeRemoteDeployer3) Close()                             {}

func TestRemoveSelectiveKeymasterContent_FinalContentEmpty_DeploysEmpty(t *testing.T) {
	// no user/global keys -> keymasterContent empty; authorized_keys contains only keymaster section
	kl := &fakeKL2{globals: nil, acc: map[int][]model.PublicKey{}}
	SetDefaultKeyLister(kl)
	defer SetDefaultKeyLister(nil)

	content := "# Keymaster Managed Keys (Serial: 1)\nssh-rsa AAAAB3...\n# end\n"
	fd := &fakeRemoteDeployer3{getContent: []byte(content), deployErr: nil}
	res := &DecommissionResult{}

	if err := removeSelectiveKeymasterContent(fd, res, 99, nil, true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if fd.deployed != "" {
		t.Fatalf("expected empty deployed content, got %q", fd.deployed)
	}
	if !res.RemoteCleanupDone {
		t.Fatalf("expected RemoteCleanupDone true")
	}
}

func TestRemoveSelectiveKeymasterContent_FinalContentEmpty_DeployFails(t *testing.T) {
	kl := &fakeKL2{globals: nil, acc: map[int][]model.PublicKey{}}
	SetDefaultKeyLister(kl)
	defer SetDefaultKeyLister(nil)

	content := "# Keymaster Managed Keys (Serial: 1)\nssh-rsa AAAAB3...\n# end\n"
	fd := &fakeRemoteDeployer3{getContent: []byte(content), deployErr: errors.New("write failed")}
	res := &DecommissionResult{}

	err := removeSelectiveKeymasterContent(fd, res, 100, nil, true)
	if err == nil {
		t.Fatalf("expected error due to deploy failure")
	}
	if !strings.Contains(err.Error(), "failed to remove empty authorized_keys file") && !strings.Contains(err.Error(), "failed to remove empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}
