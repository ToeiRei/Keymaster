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
	return parseMatcher(matcher, matcher, 0)
}

func parseMatcher(matcher string, originalMatcher string, pos int) (Expr, error) {
	matcherPrev := matcher
	matcher = strings.TrimSpace(matcher)

	// add removed whitespace to position
	if parts := strings.SplitN(matcherPrev, matcher, 2); len(parts) > 0 {
		pos += len(parts[0])
	}

	// and
	if parts := splitOnTopLevelChar(matcher, exprAnd); len(parts) > 1 {
		exprs, err := slicest.MapX(parts, func(part string) (Expr, error) {
			expr, err := parseMatcher(part, originalMatcher, pos)
			pos += len(part) + 1
			return expr, err
		})
		return AndExpr{exprs}, err
	}

	// or
	if parts := splitOnTopLevelChar(matcher, exprOr); len(parts) > 1 {
		exprs, err := slicest.MapX(parts, func(part string) (Expr, error) {
			expr, err := parseMatcher(part, originalMatcher, pos)
			pos += len(part) + 1
			return expr, err
		})
		return OrExpr{exprs}, err
	}

	// negation
	if matcher, negated := strings.CutPrefix(matcher, exprNot); negated {
		expr, err := parseMatcher(matcher, originalMatcher, pos+1)
		return NotExpr{expr}, err
	}

	// braces
	if strings.HasPrefix(matcher, exprBracesOpen) && strings.HasSuffix(matcher, exprBracesClose) {
		matcher = matcher[1 : len(matcher)-1]
		expr, err := parseMatcher(matcher, originalMatcher, pos+1)
		return BracesExpr{expr}, err
	}

	// raw value
	if tagValidationRegexpr.MatchString(matcher) {
		return ValueExpr{matcher}, nil
	}

	// invalid matcher string
	posFrom, posTo := pos+1, pos+len(matcher)
	if len(matcher) == 0 {
		return nil, fmt.Errorf("invalid tag %q in matcher %q at position %d", matcher, originalMatcher, posFrom)
	}
	return nil, fmt.Errorf("invalid tag %q in matcher %q at position %d-%d", matcher, originalMatcher, posFrom, posTo)
}

func splitOnTopLevelChar(expr string, char rune) []string {
	var result []string
	var depth int
	var cursor int

	for i, ch := range expr {
		switch ch {
		case rune(exprBracesOpen[0]):
			depth++
		case rune(exprBracesClose[0]):
			depth--
		case char:
			if depth <= 0 {
				result = append(result, expr[cursor:i])
				cursor = i + 1
			}
		}
	}

	return append(result, expr[cursor:])
}
