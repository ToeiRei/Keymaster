// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package bunhelpers

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

const (
	bunAnd bunMode = true
	bunOr  bunMode = false
)

type bunMode bool

type TagsExprToSubqueryConfig2 struct {
	TaggedTable    string
	TaggedColumnId string

	TaggedToTagTable          string
	TaggedToTagColumnTagId    string
	TaggedToTagColumnTaggedId string

	TagTable       string
	TagColumnId    string
	TagColumnValue string
}

func applyExprToSelectQuery(sq *bun.SelectQuery, cfg TagsExprToSubqueryConfig2, joinAliasMap map[string]int, expr tags.Expr, parentMode bunMode, negated bool) *bun.SelectQuery {
	switch expr := expr.(type) {
	case tags.ValueExpr:
		tagAlias := "tag_" + fmt.Sprint(joinAliasMap[expr.Value])

		strCond := " IS NOT NULL"
		if negated {
			strCond = " IS NULL"
		}

		return sq.Where(tagAlias + "." + cfg.TagColumnId + strCond)

	case tags.NotExpr:
		return applyExprToSelectQuery(sq, cfg, joinAliasMap, expr.Expr, parentMode, !negated)

	case tags.AndExpr:
		return slicest.ReduceD(expr.Exprs, sq, func(subExpr tags.Expr, sq *bun.SelectQuery) *bun.SelectQuery {
			seperator := " AND "
			if negated {
				seperator = " OR "
			}

			return sq.WhereGroup(seperator, func(sq *bun.SelectQuery) *bun.SelectQuery {
				return applyExprToSelectQuery(sq, cfg, joinAliasMap, subExpr, bunMode(!negated), negated)
			})
		})

	case tags.OrExpr:
		return slicest.ReduceD(expr.Exprs, sq, func(subExpr tags.Expr, sq *bun.SelectQuery) *bun.SelectQuery {
			seperator := " OR "
			if negated {
				seperator = " AND "
			}

			return sq.WhereGroup(seperator, func(sq *bun.SelectQuery) *bun.SelectQuery {
				return applyExprToSelectQuery(sq, cfg, joinAliasMap, subExpr, bunMode(negated), negated)
			})
		})
	}

	panic(fmt.Sprintf("Expr of type %T not supported", expr))
}

func resolveValueExprs(expr tags.Expr) []tags.ValueExpr {
	switch expr := expr.(type) {
	case tags.ValueExpr:
		return []tags.ValueExpr{expr}

	case tags.NotExpr:
		return resolveValueExprs(expr.Expr)

	case tags.AndExpr:
		return slicest.Flatten(slicest.Map(expr.Exprs, func(subExpr tags.Expr) []tags.ValueExpr { return resolveValueExprs(subExpr) }))

	case tags.OrExpr:
		return slicest.Flatten(slicest.Map(expr.Exprs, func(subExpr tags.Expr) []tags.ValueExpr { return resolveValueExprs(subExpr) }))
	}

	return nil
}

// TagsExprToSubquery applies a tags Expression tree to a Bun SelectQuery
func TagsExprToSubquery2(sq *bun.SelectQuery, cfg TagsExprToSubqueryConfig2, expr tags.Expr) *bun.SelectQuery {
	var joinCounter int
	joinAliasMap := make(map[string]int)
	valueExprs := resolveValueExprs(expr)

	for _, expr := range valueExprs {
		joinCounter++
		joinAliasMap[expr.Value] = joinCounter
	}

	// add join for each value expression
	for value, i := range joinAliasMap {
		// escape value chars
		value = slicest.ReduceD([]rune(bunEscapedChars), value, func(char rune, value string) string {
			return strings.ReplaceAll(value, string(char), bunEscape+string(char))
		})

		// check if value contains a wildcard
		containsWildcard := strings.Contains(value, tags.ExprWildcards) || strings.Contains(value, tags.ExprWildcard)

		// create table aliases
		taggedToTagAlias := "tagged_to_tag_" + fmt.Sprint(i)
		tagAlias := "tag_" + fmt.Sprint(i)

		sq = sq.
			// JOIN tagged_to_tag AS tagged_to_tag_1
			Join("/* " + value + " */ JOIN " + cfg.TaggedToTagTable + " AS " + taggedToTagAlias).
			// ON tagged_to_tag_1.tagged_id = tagged.id
			JoinOn(taggedToTagAlias + "." + cfg.TaggedToTagColumnTaggedId + " = " + cfg.TaggedTable + "." + cfg.TaggedColumnId).
			// JOIN tag AS tag_1
			Join("JOIN " + cfg.TagTable + " AS " + tagAlias)

		if containsWildcard {
			value = strings.ReplaceAll(value, tags.ExprWildcards, bunWildcards)
			value = strings.ReplaceAll(value, tags.ExprWildcard, bunWildcard)

			// ON tag_1.id = tagged_to_tag_1.tag_id AND tag_1.value LIKE ? ESCAPE '!'
			sq = sq.JoinOn(tagAlias+"."+cfg.TagColumnId+" = "+taggedToTagAlias+"."+cfg.TaggedToTagColumnTagId+" AND "+tagAlias+"."+cfg.TagColumnValue+" LIKE ? ESCAPE '"+bunEscape+"'", value)
		} else {
			// ON tag_1.id = tagged_to_tag_1.tag_id AND tag_1.value = ?
			sq = sq.JoinOn(tagAlias+"."+cfg.TagColumnId+" = "+taggedToTagAlias+"."+cfg.TaggedToTagColumnTagId+" AND "+tagAlias+"."+cfg.TagColumnValue+" = ?", value)
		}
	}

	// apply where clauses recursively
	return applyExprToSelectQuery(sq, cfg, joinAliasMap, expr, bunAnd, false)
}
