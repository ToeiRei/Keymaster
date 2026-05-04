// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags2

import (
	"strings"

	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

const (
	bunAnd bunMode = true
	bunOr  bunMode = false

	// tuned for postgres
	bunEscape       string = "!"
	bunEscapedChars string = `%_[]^-{}`
	bunWildcard     string = "_"
	bunWildcards    string = "%"
	bunTagDelimiter string = "|"
)

type bunMode bool

func ToBunString(tags Tags) string {
	return bunTagDelimiter + strings.Join(tags.Slice(), bunTagDelimiter) + bunTagDelimiter
}

func FromBunString(str string) Tags {
	if str == "" {
		return Tags{}
	}
	str, _ = strings.CutPrefix(str, bunTagDelimiter)
	str, _ = strings.CutSuffix(str, bunTagDelimiter)
	strs := strings.Split(str, bunTagDelimiter)
	return slicest.Map(strs, func(str string) Tag { return Tag(str) })
}

func ApplyToBunQuery(expr Expr, qb bun.QueryBuilder, column string) bun.QueryBuilder {
	return expr.applyToBunQuery(qb, column, bunAnd, false)
}

func (e ValueExpr) applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode, negate bool) bun.QueryBuilder {
	sqlExpr := e.Value

	// escape special chars
	sqlExpr = slicest.ReduceD([]rune(bunEscapedChars), sqlExpr, func(char rune, sqlexpr string) string {
		return strings.ReplaceAll(sqlexpr, string(char), bunEscape+string(char))
	})

	// enable wildcards
	sqlExpr = strings.ReplaceAll(sqlExpr, exprWildcards, bunWildcards)
	sqlExpr = strings.ReplaceAll(sqlExpr, exprWildcard, bunWildcard)

	// add delimiters and wildcards to not match across multiple tags
	sqlExpr = bunWildcards + bunTagDelimiter + sqlExpr + bunTagDelimiter + bunWildcards

	// setup negation string
	var queryNot string
	if negate {
		queryNot = " NOT"
	}

	// build query string
	query := column + queryNot + " LIKE ? ESCAPE '" + bunEscape + "'"

	// apply query
	if mode == bunAnd {
		return qb.Where(query, sqlExpr)
	} else {
		return qb.WhereOr(query, sqlExpr)
	}
}

func (e AndExpr) applyToBunQuery(qb bun.QueryBuilder, column string, _ bunMode, negate bool) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.Exprs, qb, func(expr Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		// flips mode AND to OR, if negated
		switch expr.(type) {
		case OrExpr:
			return applyBracesToBunQuery(expr, qb, column, bunMode(!negate), negate)
		default:
			return expr.applyToBunQuery(qb, column, bunMode(!negate), negate)
		}
	})
}

func (e OrExpr) applyToBunQuery(qb bun.QueryBuilder, column string, _ bunMode, negate bool) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.Exprs, qb, func(expr Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		// flips mode OR to AND, if negated
		switch expr.(type) {
		case AndExpr, OrExpr:
			return applyBracesToBunQuery(expr, qb, column, bunMode(negate), negate)
		default:
			return expr.applyToBunQuery(qb, column, bunMode(negate), negate)
		}
	})
}

func (e NotExpr) applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode, negate bool) bun.QueryBuilder {
	// flips negate for subexpression
	switch e.Expr.(type) {
	case AndExpr, OrExpr:
		return applyBracesToBunQuery(e.Expr, qb, column, mode, !negate)
	default:
		return e.Expr.applyToBunQuery(qb, column, mode, !negate)
	}
}

func applyBracesToBunQuery(e Expr, qb bun.QueryBuilder, column string, mode bunMode, negate bool) bun.QueryBuilder {
	var seperator string
	switch mode {
	case bunAnd:
		seperator = " AND "
	case bunOr:
		seperator = " OR "
	}

	return qb.WhereGroup(seperator, func(qb bun.QueryBuilder) bun.QueryBuilder {
		// braces them self can't be negated.
		// flipping mode and passing negated flag, has the effect of negating the whole braces content.
		return e.applyToBunQuery(qb, column, !bunMode(negate), negate)
	})
}

func pushNegatesToValues(expr Expr) Expr {
	switch expr := expr.(type) {
	case NotExpr:
		switch subExpr := expr.Expr.(type) {
		case AndExpr:
			// flip negated and expressions
			return pushNegatesToValues(OrExpr{slicest.Map(subExpr.Exprs, func(expr Expr) Expr { return NotExpr{expr} })})
		case OrExpr:
			// flip negated or expressions
			return pushNegatesToValues(AndExpr{slicest.Map(subExpr.Exprs, func(expr Expr) Expr { return NotExpr{expr} })})
		default:
			return expr
		}
	case AndExpr:
		expr.Exprs = slicest.Map(expr.Exprs, func(expr Expr) Expr { return pushNegatesToValues(expr) })
		return expr
	case OrExpr:
		expr.Exprs = slicest.Map(expr.Exprs, func(expr Expr) Expr { return pushNegatesToValues(expr) })
		return expr
	default:
		return expr
	}
}
