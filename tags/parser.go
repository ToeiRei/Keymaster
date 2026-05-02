// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/toeirei/keymaster/util/slicest"
)

var tagValidationRegexpr = regexp.MustCompile(`^[a-zA-Z0-9_\-+*/.:~=]+$`)

func ParseMatcher(matcher string) (Expr, error) {
	matcher = strings.TrimSpace(matcher)

	// and
	if parts := splitOnTopLevelChar(matcher, exprAnd); len(parts) > 1 {
		exprs, err := slicest.MapX(parts, func(part string) (Expr, error) {
			return ParseMatcher(part)
		})
		return AndExpr{exprs}, err
	}

	// or
	if parts := splitOnTopLevelChar(matcher, exprOr); len(parts) > 1 {
		exprs, err := slicest.MapX(parts, func(part string) (Expr, error) {
			return ParseMatcher(part)
		})
		return OrExpr{exprs}, err
	}

	// negation
	if matcher, negated := strings.CutPrefix(matcher, exprNot); negated {
		expr, err := ParseMatcher(matcher)
		return NotExpr{expr}, err
	}

	// braces
	if strings.HasPrefix(matcher, exprBracesOpen) && strings.HasSuffix(matcher, exprBracesClose) {
		matcher = matcher[1 : len(matcher)-1]
		expr, err := ParseMatcher(matcher)
		return BracesExpr{expr}, err
	}

	// raw value
	if tagValidationRegexpr.MatchString(matcher) {
		return ValueExpr{matcher}, nil
	}

	return nil, fmt.Errorf(`invalid tag: "%s"`, matcher)
}

func splitOnTopLevelChar(expr string, char rune) []string {
	var result []string
	var depth int
	var start int

	for i, ch := range expr {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case char:
			if depth == 0 {
				result = append(result, expr[start:i])
				start = i + 1
			}
		}
	}

	return append(result, expr[start:])
}
