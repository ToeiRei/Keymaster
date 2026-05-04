// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

const (
	exprAnd         rune   = '&'
	exprOr          rune   = '|'
	exprNot         string = "!"
	exprBracesOpen  string = "("
	exprBracesClose string = ")"
	exprWildcard    string = "*"
	exprWildcards   string = "**"
)

type Expr interface {
	fmt.Stringer
	Eval(tags Tags) bool
	applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode, negate bool) bun.QueryBuilder
}

type ValueExpr struct{ Value string }
type AndExpr struct{ Exprs []Expr }
type OrExpr struct{ Exprs []Expr }
type NotExpr struct{ Expr Expr }
type BracesExpr struct{ Expr Expr }

// [ValueExpr] implements [Expr]
// [AndExpr] implements [Expr]
// [OrExpr] implements [Expr]
// [NotExpr] implements [Expr]
// [BracesExpr] implements [Expr]

var _ Expr = ValueExpr{}
var _ Expr = AndExpr{}
var _ Expr = OrExpr{}
var _ Expr = NotExpr{}
var _ Expr = BracesExpr{}

// --- [fmt.Stringer] implementations ---

func (e ValueExpr) String() string {
	return e.Value
}
func (e AndExpr) String() string {
	return strings.Join(slicest.Map(e.Exprs, func(e Expr) string { return e.String() }), " "+string(exprAnd)+" ")
}
func (e OrExpr) String() string {
	return strings.Join(slicest.Map(e.Exprs, func(e Expr) string { return e.String() }), " "+string(exprOr)+" ")
}
func (e NotExpr) String() string {
	return exprNot + e.Expr.String()
}
func (e BracesExpr) String() string {
	return exprBracesOpen + e.Expr.String() + exprBracesClose
}

// --- [Expr.Eval] implementations ---

func (e ValueExpr) Eval(tags Tags) bool {
	expr := regexp.QuoteMeta(e.Value)
	expr = strings.ReplaceAll(expr, regexp.QuoteMeta(exprWildcards), ".*")
	expr = strings.ReplaceAll(expr, regexp.QuoteMeta(exprWildcard), ".")
	regexpr := regexp.MustCompile("^" + expr + "$")
	return slicest.Contains(tags, func(tag Tag) bool {
		return regexpr.MatchString(string(tag))
	})
}
func (e AndExpr) Eval(tags Tags) bool {
	return !slicest.Contains(e.Exprs, func(expr Expr) bool { return !expr.Eval(tags) })
}
func (e OrExpr) Eval(tags Tags) bool {
	return slicest.Contains(e.Exprs, func(expr Expr) bool { return expr.Eval(tags) })
}
func (e NotExpr) Eval(tags Tags) bool {
	return !e.Expr.Eval(tags)
}
func (e BracesExpr) Eval(tags Tags) bool {
	return e.Expr.Eval(tags)
}
