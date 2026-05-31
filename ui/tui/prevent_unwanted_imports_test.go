// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tui_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"regexp"
	"testing"
)

type Package struct {
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
}

func TestPreventUnwantedImports(t *testing.T) {
	blackList := map[string]string{
		regexp.QuoteMeta("github.com/toeirei/keymaster/") + ".+": "no keymaster imports except for the whitelisted allowed",
	}

	whiteList := []string{
		regexp.QuoteMeta("github.com/toeirei/keymaster/ui/tui") + ".*",
		regexp.QuoteMeta("github.com/toeirei/keymaster/util") + ".*",
		regexp.QuoteMeta("github.com/toeirei/keymaster/tags") + ".*",
		regexp.QuoteMeta("github.com/toeirei/keymaster/client") + ".*",
		regexp.QuoteMeta("github.com/toeirei/keymaster/ui/i18n"),
		regexp.QuoteMeta("github.com/toeirei/keymaster/buildvars"),
	}

	cmd := exec.Command("go", "list", "-json", "./...")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to execute 'go list': %v", err)
	}

	decoder := json.NewDecoder(&stdout)
	for decoder.More() {
		var pkg Package
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("failed to decode 'go list' output: %v", err)
		}

		t.Run(pkg.ImportPath, func(t *testing.T) {
			for _, pkgImport := range pkg.Imports {
				var blacklisted bool
				var reason string

				for regexpStr, _reason := range blackList {
					blacklisted = regexp.MustCompile(regexpStr).MatchString(pkgImport)
					if blacklisted {
						reason = _reason
						break
					}
				}

				if blacklisted {
					var whitelisted bool

					for _, regexpStr := range whiteList {
						whitelisted = regexp.MustCompile(regexpStr).MatchString(pkgImport)
						if whitelisted {
							break
						}
					}

					if !whitelisted {
						t.Errorf("FAIL: Package %q imports banned package %q. Reason: %s", pkg.ImportPath, pkgImport, reason)
						return
					}
				}
			}
		})
	}
}
