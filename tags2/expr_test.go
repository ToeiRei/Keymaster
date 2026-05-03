// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags2_test

import (
	"fmt"
	"testing"

	tags "github.com/toeirei/keymaster/tags2"
)

type exprTest struct {
	matcher  string
	rendered string
	cases    []exprTestCase
}

type exprTestCase struct {
	tags   tags.Tags
	result bool
}

var exprTests = []exprTest{
	// --- Basic Exact Matches ---
	{"prod", "prod", []exprTestCase{
		{tags.Tags{"prod"}, true},
		{tags.Tags{"dev"}, false},
		{tags.Tags{"production"}, false},
	}},

	// --- Wildcards (* = single char, ** = any sequence) ---
	{"v*", "v*", []exprTestCase{
		{tags.Tags{"v1"}, true},
		{tags.Tags{"v2"}, true},
		{tags.Tags{"v10"}, false},
	}},
	{"api-**", "api-**", []exprTestCase{
		{tags.Tags{"api-v1"}, true},
		{tags.Tags{"api-internal-v2"}, true},
		{tags.Tags{"web-v1"}, false},
	}},

	// --- NOT (!) ---
	{"!deprecated", "!deprecated", []exprTestCase{
		{tags.Tags{"stable"}, true},
		{tags.Tags{"deprecated"}, false},
	}},

	// --- AND (&) ---
	{"golang&backend", "golang & backend", []exprTestCase{
		{tags.Tags{"golang", "backend"}, true},
		{tags.Tags{"golang"}, false},
		{tags.Tags{"backend"}, false},
	}},

	// --- OR (|) ---
	{"ios|android", "ios | android", []exprTestCase{
		{tags.Tags{"ios"}, true},
		{tags.Tags{"android"}, true},
		{tags.Tags{"web"}, false},
	}},

	// --- Complex Nesting ---
	{"(aws|gcp)&!legacy", "(aws | gcp) & !legacy", []exprTestCase{
		{tags.Tags{"aws", "cloud"}, true},
		{tags.Tags{"gcp"}, true},
		{tags.Tags{"aws", "legacy"}, false},
		{tags.Tags{"azure"}, false},
	}},
	{"auth&(**-admin|super-**)", "auth & (**-admin | super-**)", []exprTestCase{
		{tags.Tags{"auth", "sys-admin"}, true},
		{tags.Tags{"auth", "super-user"}, true},
		{tags.Tags{"auth", "user"}, false},
		{tags.Tags{"sys-admin"}, false},
	}},
	{"!(test|stage)&prod", "!(test | stage) & prod", []exprTestCase{
		{tags.Tags{"prod"}, true},
		{tags.Tags{"test", "prod"}, false},
		{tags.Tags{"stage", "prod"}, false},
	}},

	// --- Edge Cases ---
	{"**", "**", []exprTestCase{
		{tags.Tags{"anything"}, true},
		{tags.Tags{""}, true},
	}},
	{"*", "*", []exprTestCase{
		{tags.Tags{"ab"}, false},
		{tags.Tags{"a"}, true},
		{tags.Tags{""}, false},
	}},
}

func TestExpr(t *testing.T) {
	for _, et := range exprTests {
		t.Run(fmt.Sprintf("Matcher(%q)", et.matcher), func(t *testing.T) {
			expr, err := tags.ParseMatcher(et.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", et.matcher, err)
			}

			t.Run("String()", func(t *testing.T) {
				got := expr.String()
				if got != et.rendered {
					t.Errorf("got = %q; expected = %q", got, et.rendered)
				}

				t.Log(got)
			})

			for _, tc := range et.cases {
				t.Run(fmt.Sprintf("Eval(%v)", tc.tags), func(t *testing.T) {
					got := expr.Eval(tc.tags)
					if got != tc.result {
						t.Errorf("got = %v; expected = %v", got, tc.result)
					}

					t.Log(got)
				})
			}
		})
	}
}
