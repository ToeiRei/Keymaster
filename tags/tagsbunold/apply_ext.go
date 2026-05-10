// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tagsbunold

import (
	"fmt"

	"github.com/toeirei/keymaster/tags"
	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

func applyToBunQuery(qb bun.QueryBuilder, expr tags.Expr, column string, mode Operator) bun.QueryBuilder {
	switch expr := expr.(type) {
	case tags.ValueExpr:
		return applyValueExprToBunQuery(qb, expr, column, mode)
	case tags.NotExpr:
		return applyNotExprToBunQuery(qb, expr, column, mode)
	case tags.AndExpr:
		return applyAndExprToBunQuery(qb, expr, column, mode)
	case tags.OrExpr:
		return applyOrExprToBunQuery(qb, expr, column, mode)
	}

	panic(fmt.Sprintf("Expr of type %T not supported", expr))
}

func applyValueExprToBunQuery(qb bun.QueryBuilder, e tags.ValueExpr, column string, mode Operator) bun.QueryBuilder {
	return applyValueToBunQuery(e.Value, false, qb, column, mode)
}

func applyAndExprToBunQuery(qb bun.QueryBuilder, e tags.AndExpr, column string, _ Operator) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.Exprs, qb, func(expr tags.Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		//
		switch expr.(type) {
		case tags.OrExpr:
			return applyBracesToBunQuery(expr, qb, column, And)
		default:
			return applyToBunQuery(qb, expr, column, And)
		}
	})
}

func applyOrExprToBunQuery(qb bun.QueryBuilder, e tags.OrExpr, column string, _ Operator) bun.QueryBuilder {
	// applys all sub expressions
	return slicest.ReduceD(e.Exprs, qb, func(expr tags.Expr, qb bun.QueryBuilder) bun.QueryBuilder {
		switch expr.(type) {
		case tags.AndExpr, tags.OrExpr:
			return applyBracesToBunQuery(expr, qb, column, Or)
		default:
			return applyToBunQuery(qb, expr, column, Or)
		}
	})
}

func applyNotExprToBunQuery(qb bun.QueryBuilder, e tags.NotExpr, column string, mode Operator) bun.QueryBuilder {
	// make sure, NotExpr contains only a ValueExpr
	expr, ok := e.Expr.(tags.ValueExpr)
	if !ok {
		panic("NotExpr contained non ValueExpr. This behavior is not compatible with bun.QueryBuilder and should have been avoided by pushNegatesToValues().")
	}

	return applyValueToBunQuery(expr.Value, true, qb, column, mode)
}
