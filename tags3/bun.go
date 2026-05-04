// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags3

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
	return pushNegatesToValues(expr).applyToBunQuery(qb, column, bunAnd)
}

func (e ValueExpr) applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode) bun.QueryBuilder {
	return applyValueToBunQuery(e.Value, false, qb, column, mode)
}

func (e AndExpr) applyToBunQuery(qb bun.QueryBuilder, column string, _ bunMode) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.Exprs, qb, func(expr Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		//
		switch expr.(type) {
		case OrExpr:
			return applyBracesToBunQuery(expr, qb, column, bunAnd)
		default:
			return expr.applyToBunQuery(qb, column, bunAnd)
		}
	})
}

func (e OrExpr) applyToBunQuery(qb bun.QueryBuilder, column string, _ bunMode) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.Exprs, qb, func(expr Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		switch expr.(type) {
		case AndExpr, OrExpr:
			return applyBracesToBunQuery(expr, qb, column, bunOr)
		default:
			return expr.applyToBunQuery(qb, column, bunOr)
		}
	})
}

func (e NotExpr) applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode) bun.QueryBuilder {
	// make sure, NotExpr contains only a ValueExpr
	expr, ok := e.Expr.(ValueExpr)
	if !ok {
		panic("NotExpr contained non ValueExpr. This behavior is not compatible with bun.QueryBuilder and should have been avoided by pushNegatesToValues().")
	}

	return applyValueToBunQuery(expr.Value, true, qb, column, mode)
}

func applyBracesToBunQuery(e Expr, qb bun.QueryBuilder, column string, mode bunMode) bun.QueryBuilder {
	var seperator string
	switch mode {
	case bunAnd:
		seperator = " AND "
	case bunOr:
		seperator = " OR "
	}

	return qb.WhereGroup(seperator, func(qb bun.QueryBuilder) bun.QueryBuilder {
		return e.applyToBunQuery(qb, column, mode)
	})
}
func applyValueToBunQuery(value string, negated bool, qb bun.QueryBuilder, column string, mode bunMode) bun.QueryBuilder {
	sqlExpr := value

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
	if negated {
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

func pushNegatesToValues(expr Expr) Expr {
	switch expr := expr.(type) {
	case NotExpr:
		switch subExpr := expr.Expr.(type) {
		case AndExpr:
			// flip by switching to OrExpr and negating its content
			return pushNegatesToValues(OrExpr{slicest.Map(subExpr.Exprs, func(expr Expr) Expr { return NotExpr{expr} })})
		case OrExpr:
			// flip by switching to AndExpr and negating its content
			return pushNegatesToValues(AndExpr{slicest.Map(subExpr.Exprs, func(expr Expr) Expr { return NotExpr{expr} })})
		default:
			return expr
		}
	case AndExpr:
		return AndExpr{slicest.Map(expr.Exprs, func(expr Expr) Expr { return pushNegatesToValues(expr) })}
	case OrExpr:
		return OrExpr{slicest.Map(expr.Exprs, func(expr Expr) Expr { return pushNegatesToValues(expr) })}
	default:
		return expr
	}
}
