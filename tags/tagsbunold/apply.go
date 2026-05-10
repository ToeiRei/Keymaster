// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tagsbunold

import (
	"strings"

	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

const (
	And Operator = true
	Or  Operator = false

	// tuned for postgres
	bunEscape       string = "!"
	bunEscapedChars string = bunEscape + "%_[]^-{}"
	bunWildcard     string = "_"
	bunWildcards    string = "%"
	bunTagDelimiter string = "|"
)

type Operator bool

func ToBunString(tags tags.Tags) string {
	return bunTagDelimiter + strings.Join(tags.Slice(), bunTagDelimiter) + bunTagDelimiter
}

func FromBunString(str string) tags.Tags {
	if str == "" {
		return tags.Tags{}
	}
	str, _ = strings.CutPrefix(str, bunTagDelimiter)
	str, _ = strings.CutSuffix(str, bunTagDelimiter)
	strs := strings.Split(str, bunTagDelimiter)
	return slicest.Map(strs, func(str string) tags.Tag { return tags.Tag(str) })
}

// Only works as a db pre filter to reduce the number of results.
// Due to wildcards matching over [bunTagDelimiter] it does not produce 100% acurate results.
// Use [Expr.Eval] to ensure correct results.
func ApplyToBunQuery(expr tags.Expr, qb bun.QueryBuilder, column string) bun.QueryBuilder {
	return applyToBunQuery(qb, pushNegatesToValues(expr), column, And)
}

func applyBracesToBunQuery(e tags.Expr, qb bun.QueryBuilder, column string, mode Operator) bun.QueryBuilder {
	var seperator string
	switch mode {
	case And:
		seperator = " AND "
	case Or:
		seperator = " OR "
	}

	return qb.WhereGroup(seperator, func(qb bun.QueryBuilder) bun.QueryBuilder {
		return applyToBunQuery(qb, e, column, mode)
	})
}
func applyValueToBunQuery(value string, negated bool, qb bun.QueryBuilder, column string, mode Operator) bun.QueryBuilder {
	sqlExpr := value

	// escape special chars
	sqlExpr = slicest.ReduceD([]rune(bunEscapedChars), sqlExpr, func(char rune, sqlexpr string) string {
		return strings.ReplaceAll(sqlexpr, string(char), bunEscape+string(char))
	})

	// enable wildcards
	sqlExpr = strings.ReplaceAll(sqlExpr, tags.ExprWildcards, bunWildcards)
	sqlExpr = strings.ReplaceAll(sqlExpr, tags.ExprWildcard, bunWildcard)

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
	if mode == And {
		return qb.Where(query, sqlExpr)
	} else {
		return qb.WhereOr(query, sqlExpr)
	}
}

func pushNegatesToValues(expr tags.Expr) tags.Expr {
	switch expr := expr.(type) {
	case tags.NotExpr:
		switch expr.Expr.(type) {
		case tags.AndExpr, tags.OrExpr:
			return pushNegatesToValues(tryResolveNotExpr(expr))
		default:
			return expr
		}
	case tags.AndExpr:
		return tags.AndExpr{slicest.Map(expr.Exprs, func(expr tags.Expr) tags.Expr { return pushNegatesToValues(expr) })}
	case tags.OrExpr:
		return tags.OrExpr{slicest.Map(expr.Exprs, func(expr tags.Expr) tags.Expr { return pushNegatesToValues(expr) })}
	default:
		return expr
	}
}

func tryResolveNotExpr(e tags.NotExpr) tags.Expr {
	switch expr := e.Expr.(type) {
	case tags.AndExpr:
		return tags.OrExpr{slicest.Map(expr.Exprs, func(expr tags.Expr) tags.Expr { return tags.NotExpr{expr} })}
	case tags.OrExpr:
		return tags.AndExpr{slicest.Map(expr.Exprs, func(expr tags.Expr) tags.Expr { return tags.NotExpr{expr} })}
	default:
		return e
	}
}
