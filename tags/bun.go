// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

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
	strs := slicest.Map(tags, func(tag Tag) string { return string(tag) })
	return bunTagDelimiter + strings.Join(strs, bunTagDelimiter) + bunTagDelimiter
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
	sqlExpr := e.value

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
	return slicest.ReduceD(e.exprs, qb, func(expr Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		// flips mode AND to OR, if negated
		return expr.applyToBunQuery(qb, column, bunMode(!negate), negate)
	})
}

func (e OrExpr) applyToBunQuery(qb bun.QueryBuilder, column string, _ bunMode, negate bool) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.exprs, qb, func(expr Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		// flips mode OR to AND, if negated
		return expr.applyToBunQuery(qb, column, bunMode(negate), negate)
	})
}

func (e NotExpr) applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode, negate bool) bun.QueryBuilder {
	// flips negate for subexpression
	return e.expr.applyToBunQuery(qb, column, mode, !negate)
}

func (e BracesExpr) applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode, negate bool) bun.QueryBuilder {
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
		return e.expr.applyToBunQuery(qb, column, !bunMode(negate), negate)
	})
}
