// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

import (
	"errors"

	"github.com/toeirei/keymaster/util/slicest"
)

func MatcherFrom(include, exclude []Tags) (Expr, error) {
	includeExprs := slicest.Map(include, func(tags Tags) Expr {
		return AndExpr{slicest.Map(tags, func(tag Tag) Expr {
			return ValueExpr{string(tag)}
		})}
	})

	excludeExprs := slicest.Map(exclude, func(tags Tags) Expr {
		return AndExpr{slicest.Map(tags, func(tag Tag) Expr {
			return ValueExpr{string(tag)}
		})}
	})

	expr := AndExpr{[]Expr{
		OrExpr{includeExprs},
		NotExpr{OrExpr{excludeExprs}},
	}}.Optimize()

	if slicest.Contains(include, func(tags Tags) bool {
		return !expr.Eval(tags)
	}) || slicest.Contains(exclude, func(tags Tags) bool {
		return expr.Eval(tags)
	}) {
		return nil, errors.New("unable to generate matcher for these conditions")
	}

	return expr, nil
}
