// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package cli

import (
	"runtime/debug"
	"testing"
)

func TestResolveBuildVersion_MainVersion(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/toeirei/keymaster", Version: "v1.2.3"},
	}
	v, c, d := resolveBuildVersion(info)
	if v != "v1.2.3" {
		t.Fatalf("expected v1.2.3 got %s", v)
	}
	if c != gitCommit {
		t.Fatalf("expected commit to equal package gitCommit (default) got %s", c)
	}
	if d != buildDate {
		t.Fatalf("expected date to equal package buildDate (default) got %s", d)
	}
}

func TestResolveBuildVersion_DependencyFallback(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/toeirei/keymaster", Version: "(devel)"},
		Deps: []*debug.Module{
			{Path: "github.com/toeirei/keymaster", Version: "v1.5.1-0.20251130131337-d1692e4643ee"},
		},
	}
	v, _, _ := resolveBuildVersion(info)
	if v != "v1.5.1-0.20251130131337-d1692e4643ee" {
		t.Fatalf("expected dependency version fallback got %s", v)
	}
}

func TestResolveBuildVersion_GitCommitFallback(t *testing.T) {
	// preserve original
	orig := gitCommit
	defer func() { gitCommit = orig }()
	gitCommit = "deadbeef"
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/toeirei/keymaster", Version: "(devel)"},
	}
	v, _, _ := resolveBuildVersion(info)
	if v != "deadbeef" {
		t.Fatalf("expected gitCommit fallback got %s", v)
	}
}
