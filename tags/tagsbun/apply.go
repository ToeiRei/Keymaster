// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tagsbun

import (
	"fmt"
	"strings"

	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

type Operator bool

const (
	And Operator = true
	Or  Operator = false

	bunEscape       string = "!"
	bunEscapedChars string = bunEscape + "%_[]^-{}"
	bunWildcard     string = "_"
	bunWildcards    string = "%"
)

type TagsExprToSubqueryConfig struct {
	TaggedTable    string
	TaggedColumnId string

	TaggedToTagTable          string
	TaggedToTagColumnTagId    string
	TaggedToTagColumnTaggedId string

	TagTable       string
	TagColumnId    string
	TagColumnValue string
}

func seperatorFromOperator(operator Operator) string {
	switch operator {
	case And:
		return " AND "
	case Or:
		return " OR "
	}
	return ""
}

func TagsExprToWhere(expr tags.Expr, cfg TagsExprToSubqueryConfig) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		return applyTagsExprToSelectQuery(sq, expr.Optimize(), cfg, And, false)
	}
}

func applyTagsExprToSelectQuery(sq *bun.SelectQuery, expr tags.Expr, cfg TagsExprToSubqueryConfig, parentOperator Operator, negated bool) *bun.SelectQuery {
	switch expr := expr.(type) {
	case tags.ValueExpr:
		// create sub query to search for tag on tagged
		subQuery := sq.DB().NewSelect().
			ColumnExpr("1").
			Table(cfg.TaggedToTagTable).
			Join("INNER JOIN " + cfg.TagTable).
			JoinOn(cfg.TagTable + "." + cfg.TagColumnId + " = " + cfg.TaggedToTagTable + "." + cfg.TaggedToTagColumnTagId).
			Where(cfg.TaggedToTagTable + "." + cfg.TaggedToTagColumnTaggedId + " = " + cfg.TaggedTable + "." + cfg.TaggedColumnId)

		// check if expr value contains any wildcards
		value := expr.Value
		containsWildcard := strings.Contains(value, tags.ExprWildcards) || strings.Contains(value, tags.ExprWildcard)

		if containsWildcard {
			// escape value
			value = slicest.ReduceD([]rune(bunEscapedChars), value, func(char rune, sqlValue string) string {
				return strings.ReplaceAll(sqlValue, string(char), bunEscape+string(char))
			})

			// convert wildcards to db dialect
			value = strings.ReplaceAll(value, tags.ExprWildcards, bunWildcards)
			value = strings.ReplaceAll(value, tags.ExprWildcard, bunWildcard)

			// apply value to sub query
			subQuery = subQuery.Where(cfg.TagTable+"."+cfg.TagColumnValue+" LIKE ? ESCAPE '"+bunEscape+"'", value)
		} else {
			// apply value to sub query
			subQuery = subQuery.Where(cfg.TagTable+"."+cfg.TagColumnValue+" = ?", value)
		}

		// where based on parent operator
		whereFn := sq.Where
		if parentOperator == Or {
			whereFn = sq.WhereOr
		}

		// query based on negated flag
		queryStr := "EXISTS (?)"
		if negated {
			queryStr = "NOT EXISTS (?)"
		}

		// apply sub query to query
		return whereFn(queryStr, subQuery)

	case tags.NotExpr:
		// flip negated flag
		return applyTagsExprToSelectQuery(sq, expr.Expr, cfg, parentOperator, !negated)

	case tags.AndExpr:
		// create qhere group seperated (to other groups) by parentOperator
		return sq.WhereGroup(seperatorFromOperator(parentOperator), func(sq *bun.SelectQuery) *bun.SelectQuery {
			// apply sub expressions
			return slicest.ReduceD(expr.Exprs, sq, func(subExpr tags.Expr, sq *bun.SelectQuery) *bun.SelectQuery {
				// when negated: flip operator and pass negated flag
				return applyTagsExprToSelectQuery(sq, subExpr, cfg, And != Operator(negated), negated)
			})
		})

	case tags.OrExpr:
		// create qhere group seperated (to other groups) by parentOperator
		return sq.WhereGroup(seperatorFromOperator(parentOperator), func(sq *bun.SelectQuery) *bun.SelectQuery {
			// apply sub expressions
			return slicest.ReduceD(expr.Exprs, sq, func(subExpr tags.Expr, sq *bun.SelectQuery) *bun.SelectQuery {
				// when negated: flip operator and pass negated flag
				return applyTagsExprToSelectQuery(sq, subExpr, cfg, Or != Operator(negated), negated)
			})
		})
	}

	panic(fmt.Sprintf("Expr of type %T not supported", expr))
}
