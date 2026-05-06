// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags_test

import (
	"fmt"
	"testing"

	tags "github.com/toeirei/keymaster/tags"
)

type exprTest struct {
	matcher           string
	rendered          string
	renderedOptimized string
	cases             []exprTestCase
}

type exprTestCase struct {
	tags   tags.Tags
	result bool
}

var exprTests = []exprTest{
	// --- Basic Exact Matches ---
	{
		"prod",
		"prod",
		"prod",
		[]exprTestCase{
			{tags.Tags{"prod"}, true},
			{tags.Tags{"dev"}, false},
			{tags.Tags{"production"}, false},
		},
	},

	// --- Wildcards (* = single char, ** = any sequence) ---
	{
		"v*",
		"v*",
		"v*",
		[]exprTestCase{
			{tags.Tags{"v1"}, true},
			{tags.Tags{"v2"}, true},
			{tags.Tags{"v10"}, false},
		},
	},
	{
		"api-**",
		"api-**",
		"api-**",
		[]exprTestCase{
			{tags.Tags{"api-v1"}, true},
			{tags.Tags{"api-internal-v2"}, true},
			{tags.Tags{"web-v1"}, false},
		},
	},

	// --- NOT (!) ---
	{
		"!deprecated",
		"!deprecated",
		"!deprecated",
		[]exprTestCase{
			{tags.Tags{"stable"}, true},
			{tags.Tags{"deprecated"}, false},
		},
	},

	// --- AND (&) ---
	{
		"golang&backend",
		"golang & backend",
		"golang & backend",
		[]exprTestCase{
			{tags.Tags{"golang", "backend"}, true},
			{tags.Tags{"golang"}, false},
			{tags.Tags{"backend"}, false},
		},
	},

	// --- OR (|) ---
	{
		"ios|android",
		"ios | android",
		"ios | android",
		[]exprTestCase{
			{tags.Tags{"ios"}, true},
			{tags.Tags{"android"}, true},
			{tags.Tags{"web"}, false},
		},
	},

	// --- Complex Nesting ---
	{
		"(aws|gcp)&!legacy",
		"(aws | gcp) & !legacy",
		"(aws | gcp) & !legacy",
		[]exprTestCase{
			{tags.Tags{"aws", "cloud"}, true},
			{tags.Tags{"gcp"}, true},
			{tags.Tags{"aws", "legacy"}, false},
			{tags.Tags{"azure"}, false},
		},
	},
	{
		"auth&(**-admin|super-**)",
		"auth & (**-admin | super-**)",
		"auth & (**-admin | super-**)",
		[]exprTestCase{
			{tags.Tags{"auth", "sys-admin"}, true},
			{tags.Tags{"auth", "super-user"}, true},
			{tags.Tags{"auth", "user"}, false},
			{tags.Tags{"sys-admin"}, false},
		},
	},
	{
		"!(test|stage)&prod",
		"!(test | stage) & prod",
		"!(test | stage) & prod",
		[]exprTestCase{
			{tags.Tags{"prod"}, true},
			{tags.Tags{"test", "prod"}, false},
			{tags.Tags{"stage", "prod"}, false},
		},
	},

	// --- Edge Cases ---
	{
		"**",
		"**",
		"**",
		[]exprTestCase{
			{tags.Tags{"anything"}, true},
			{tags.Tags{""}, true},
		},
	},
	{
		"*",
		"*",
		"*",
		[]exprTestCase{
			{tags.Tags{"ab"}, false},
			{tags.Tags{"a"}, true},
			{tags.Tags{""}, false},
		},
	},

	// --- Optimizeable ---
	{
		"(aws|gcp)|legacy",
		"(aws | gcp) | legacy",
		"aws | gcp | legacy",
		[]exprTestCase{
			{tags.Tags{"aws", "cloud"}, true},
			{tags.Tags{"gcp"}, true},
			{tags.Tags{"legacy"}, true},
			{tags.Tags{"azure"}, false},
		},
	},
	{
		"(aws&gcp)&legacy",
		"(aws & gcp) & legacy",
		"aws & gcp & legacy",
		[]exprTestCase{
			{tags.Tags{"aws", "cloud"}, false},
			{tags.Tags{"gcp"}, false},
			{tags.Tags{"aws", "gcp", "legacy"}, true},
		},
	},
	{
		"auth&!(admin|super)",
		"auth & !(admin | super)",
		"auth & !(admin | super)",
		[]exprTestCase{
			{tags.Tags{"auth", "sys-admin"}, true},
			{tags.Tags{"auth", "super-user"}, true},
			{tags.Tags{"auth", "user"}, true},
			{tags.Tags{"sys-admin"}, false},
		},
	},
	{
		"!(test&stage)&prod",
		"!(test & stage) & prod",
		"!(test & stage) & prod",
		[]exprTestCase{
			{tags.Tags{"prod"}, true},
			{tags.Tags{"test", "prod"}, true},
			{tags.Tags{"stage", "test", "prod"}, false},
			{tags.Tags{"stage"}, false},
		},
	},
	{
		"!!prod",
		"!!prod",
		"prod",
		[]exprTestCase{
			{tags.Tags{"prod"}, true},
			{tags.Tags{"test"}, false},
		},
	},
	{
		"aa & (bb & cc)",
		"aa & (bb & cc)",
		"aa & bb & cc",
		[]exprTestCase{},
	},
	{
		"aa | (bb | cc)",
		"aa | (bb | cc)",
		"aa | bb | cc",
		[]exprTestCase{},
	},
	{
		"(aa & aa) | bb",
		"(aa & aa) | bb",
		"aa | bb",
		[]exprTestCase{},
	},
	{
		"(aa | aa) & bb",
		"(aa | aa) & bb",
		"aa & bb",
		[]exprTestCase{},
	},
	{
		"(aa | bb) & bb",
		"(aa | bb) & bb",
		"bb",
		[]exprTestCase{},
	},
	{
		"(aa & bb) | bb",
		"(aa & bb) | bb",
		"bb",
		[]exprTestCase{},
	},
	{
		"!!((!!tag_1&tag_1&(tag_1|tag_2))|((tag_1&tag_3)|(tag_1&tag_1))|!!(tag_4&(tag_4|tag_5)&!(!(tag_6|(tag_6&tag_7)))))",
		"!!((!!tag_1 & tag_1 & (tag_1 | tag_2)) | ((tag_1 & tag_3) | (tag_1 & tag_1)) | !!(tag_4 & (tag_4 | tag_5) & !!(tag_6 | (tag_6 & tag_7))))",
		"tag_1 | (tag_4 & tag_6)",
		[]exprTestCase{},
	},
}

func TestExprString(t *testing.T) {
	for _, et := range exprTests {
		t.Run(fmt.Sprintf("Matcher(%q).String()", et.matcher), func(t *testing.T) {
			expr, err := tags.ParseMatcher(et.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", et.matcher, err)
			}

			got := expr.String()
			if got != et.rendered {
				t.Errorf("got = %q; expected = %q", got, et.rendered)
			}

			t.Log(got)
		})
	}
}

func TestExprEval(t *testing.T) {
	for _, et := range exprTests {
		t.Run(fmt.Sprintf("Matcher(%q)", et.matcher), func(t *testing.T) {
			expr, err := tags.ParseMatcher(et.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", et.matcher, err)
			}

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

func TestExprOptimize(t *testing.T) {
	for _, et := range exprTests {
		t.Run(fmt.Sprintf("Matcher(%q).Optimize()", et.matcher), func(t *testing.T) {
			expr, err := tags.ParseMatcher(et.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", et.matcher, err)
			}

			got := expr.Optimize().String()
			if got != et.renderedOptimized {
				t.Errorf("got = %q; expected = %q", got, et.renderedOptimized)
			}

			t.Log(got)
		})
	}
}

func TestExprStringReproduceable(t *testing.T) {
	for _, et := range exprTests {
		t.Run(fmt.Sprintf("Matcher(%q).String()", et.matcher), func(t *testing.T) {
			expr1, err := tags.ParseMatcher(et.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher %q: %v", et.matcher, err)
			}

			render1 := expr1.String()

			expr2, err := tags.ParseMatcher(et.matcher)
			if err != nil {
				t.Fatalf("failed to parse matcher from render %q: %v", et.matcher, err)
			}

			render2 := expr2.String()

			if render1 != render2 {
				t.Errorf("got = %q; expected = %q", render1, render2)
			}

			t.Log(render1)
		})
	}
}
