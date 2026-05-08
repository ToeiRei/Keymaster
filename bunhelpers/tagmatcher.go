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
	bunEscape       string = "!"
	bunEscapedChars string = bunEscape + "%_[]^-{}"
	bunWildcard     string = "_"
	bunWildcards    string = "%"
)

type TagsExprToSubqueryConfig struct {
	Db bun.IDB

	// TaggedModel    any
	TaggedTable    string
	TaggedColumnId string

	TaggedToTagTable          string
	TaggedToTagColumnTagId    string
	TaggedToTagColumnTaggedId string

	TagTable       string
	TagColumnId    string
	TagColumnValue string
}

// TagsExprToSubquery converts the Expression tree into a Bun SelectQuery that returns pk_ids
func TagsExprToSubquery(cfg TagsExprToSubqueryConfig, expr tags.Expr) *bun.SelectQuery {
	switch expr := expr.(type) {
	case tags.ValueExpr:
		sqlValue := expr.Value
		sqlValue = slicest.ReduceD([]rune(bunEscapedChars), sqlValue, func(char rune, sqlValue string) string {
			return strings.ReplaceAll(sqlValue, string(char), bunEscape+string(char))
		})

		containsWildcard := strings.Contains(sqlValue, tags.ExprWildcards) || strings.Contains(sqlValue, tags.ExprWildcard)

		sqlValue = strings.ReplaceAll(sqlValue, tags.ExprWildcards, bunWildcards)
		sqlValue = strings.ReplaceAll(sqlValue, tags.ExprWildcard, bunWildcard)

		sq := cfg.Db.NewSelect().
			Comment(expr.String()).
			Table(cfg.TaggedToTagTable).
			Column(cfg.TaggedToTagColumnTaggedId).
			Join("JOIN " + cfg.TagTable + " AS tag").
			JoinOn("tag." + cfg.TagColumnId + " = " + cfg.TaggedToTagTable + "." + cfg.TaggedToTagColumnTagId)

		if containsWildcard {
			return sq.Where("tag."+cfg.TagColumnValue+" LIKE ? ESCAPE '"+bunEscape+"'", sqlValue)
		} else {
			return sq.Where("tag."+cfg.TagColumnValue+" = ?", sqlValue)
		}

	case tags.NotExpr:
		// Logic: tagged_id NOT IN (subquery)
		return cfg.Db.NewSelect().
			Comment(expr.String()).
			Table(cfg.TaggedTable).
			Column(cfg.TaggedColumnId).
			Where(cfg.TaggedColumnId+" NOT IN (?)", TagsExprToSubquery(cfg, expr.Expr))

	case tags.AndExpr:
		// Logic: tagged_id IN (subquery) AND tagged_id IN (subquery) AND ...
		sq := cfg.Db.NewSelect().
			Comment(expr.String()).
			Table(cfg.TaggedTable).
			Column(cfg.TaggedColumnId)
		return slicest.ReduceD(expr.Exprs, sq, func(subExpr tags.Expr, sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cfg.TaggedColumnId+" IN (?)", TagsExprToSubquery(cfg, subExpr))
		})

	case tags.OrExpr:
		// Logic: tagged_id IN (subquery) OR tagged_id IN (subquery) OR ...
		sq := cfg.Db.NewSelect().
			Comment(expr.String()).
			Table(cfg.TaggedTable).
			Column(cfg.TaggedColumnId)
		return slicest.ReduceD(expr.Exprs, sq, func(subExpr tags.Expr, sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(cfg.TaggedColumnId+" IN (?)", TagsExprToSubquery(cfg, subExpr))
		})
	}

	panic(fmt.Sprintf("Expr of type %T not supported", expr))
}
