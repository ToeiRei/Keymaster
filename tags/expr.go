// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package tags

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/toeirei/keymaster/util/slicest"
	"github.com/uptrace/bun"
)

const (
	exprAnd           rune   = '&'
	exprOr            rune   = '|'
	exprNot           string = "!"
	exprBracesOpen    rune   = '('
	exprBracesClose   rune   = ')'
	exprWildcard      string = "*"
	exprWildcards     string = "**"
	exprHashDelimiter rune   = ';'
)

// var hashWildcardCharSet string = "[^" + regexp.QuoteMeta(string(exprAnd)+string(exprOr)+string(exprNot)+string(exprBracesOpen)+string(exprBracesClose)+string(exprHashDelimiter)) + "]"

type Expr interface {
	fmt.Stringer
	Eval(tags Tags) bool
	// Modifier at the beginning.
	// Always use braces for sub expressions.
	// Always sort sub expressions.
	// Use semicolon as delimiter for sub expressions.
	hash() string
	Optimize() Expr
	applyToBunQuery(qb bun.QueryBuilder, column string, mode bunMode) bun.QueryBuilder
}

type ValueExpr struct{ Value string }
type AndExpr struct{ Exprs []Expr }
type OrExpr struct{ Exprs []Expr }
type NotExpr struct{ Expr Expr }

// [ValueExpr] implements [Expr]
// [AndExpr] implements [Expr]
// [OrExpr] implements [Expr]
// [NotExpr] implements [Expr]

var _ Expr = ValueExpr{}
var _ Expr = AndExpr{}
var _ Expr = OrExpr{}
var _ Expr = NotExpr{}

// func ExprSubsumes(expr1, expr2 Expr) bool {
// 	hash1, hash2 := expr1.hash(), expr2.hash()

// 	hash1 = regexp.QuoteMeta(hash1)
// 	hash1 = strings.ReplaceAll(hash1, regexp.QuoteMeta(exprWildcards), hashWildcardCharSet+"*")
// 	hash1 = strings.ReplaceAll(hash1, regexp.QuoteMeta(exprWildcard), hashWildcardCharSet)

// 	regexpr := regexp.MustCompile("^" + hash1 + "$")
// 	return regexpr.MatchString(hash2)
// }

func (e NotExpr) tryResolve() Expr {
	switch expr := e.Expr.(type) {
	case AndExpr:
		return OrExpr{slicest.Map(expr.Exprs, func(expr Expr) Expr { return NotExpr{expr} })}
	case OrExpr:
		return AndExpr{slicest.Map(expr.Exprs, func(expr Expr) Expr { return NotExpr{expr} })}
	default:
		return e
	}
}

// --- [fmt.Stringer] implementations ---

func (e ValueExpr) String() string {
	return e.Value
}
func (e AndExpr) String() string {
	return strings.Join(slicest.Map(e.Exprs, func(e Expr) string {
		switch e.(type) {
		case AndExpr, OrExpr:
			return string(exprBracesOpen) + e.String() + string(exprBracesClose)
		default:
			return e.String()
		}
	}), " "+string(exprAnd)+" ")
}
func (e OrExpr) String() string {
	return strings.Join(slicest.Map(e.Exprs, func(e Expr) string {
		switch e.(type) {
		case AndExpr, OrExpr:
			return string(exprBracesOpen) + e.String() + string(exprBracesClose)
		default:
			return e.String()
		}
	}), " "+string(exprOr)+" ")
}
func (e NotExpr) String() string {
	switch e.Expr.(type) {
	case AndExpr, OrExpr:
		return exprNot + string(exprBracesOpen) + e.Expr.String() + string(exprBracesClose)
	default:
		return exprNot + e.Expr.String()
	}
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

// --- [Expr.hash] implementations ---

func (e ValueExpr) hash() string {
	return e.Value
}
func (e AndExpr) hash() string {
	hashes := slicest.Map(e.Exprs, func(e Expr) string { return e.hash() })
	slices.Sort(hashes)
	return string(exprAnd) + string(exprBracesOpen) + strings.Join(hashes, string(exprHashDelimiter)) + string(exprBracesClose)
}
func (e OrExpr) hash() string {
	hashes := slicest.Map(e.Exprs, func(e Expr) string { return e.hash() })
	slices.Sort(hashes)
	return string(exprOr) + string(exprBracesOpen) + strings.Join(hashes, string(exprHashDelimiter)) + string(exprBracesClose)
}
func (e NotExpr) hash() string {
	return exprNot + string(exprBracesOpen) + e.Expr.hash() + string(exprBracesClose)
}

// --- [Expr.Optimize] implementations ---

func (e ValueExpr) Optimize() Expr { return e }
func (e AndExpr) Optimize() Expr {
	e.Exprs = slicest.Map(e.Exprs, func(expr Expr) Expr { return expr.Optimize() })

	// flatten nested and expressions // & negated or expressions
	e.Exprs = slicest.Flatten(slicest.Map(e.Exprs, func(expr Expr) []Expr {
		// if notExpr, ok := expr.(NotExpr); ok {
		// 	if _, ok := notExpr.Expr.(OrExpr); ok {
		// 		expr = notExpr.Flip()
		// 	}
		// }
		if expr, ok := expr.(AndExpr); ok {
			return expr.Exprs
		}
		return []Expr{expr}
	}))

	// deduplicate nested expressions
	e.Exprs = sliceDeduplicateFunc(e.Exprs, func(expr Expr) string { return expr.hash() })
	// e.Exprs = slicest.FilterI(e.Exprs, func(i1 int, expr Expr) bool {
	// 	// is expression not contained by any other expression in its parent expression
	// 	// return !slicest.ContainsI(e.Exprs, func(i2 int, otherExpr Expr) bool { return i1 != i2 && ExprSubsumes(otherExpr, expr) })
	// 	return !slicest.ContainsI(e.Exprs, func(i2 int, otherExpr Expr) bool {
	// 		differentIndex := i1 != i2
	// 		subsumes := ExprSubsumes(otherExpr, expr)
	// 		if i1 > i2 {
	// 			return differentIndex && subsumes && !ExprSubsumes(expr, otherExpr)
	// 		}
	// 		return differentIndex && subsumes
	// 	})
	// })

	// remove redundant nested or expressions
	e.Exprs = slicest.Filter(e.Exprs, func(expr Expr) bool {
		if orExpr, ok := expr.(OrExpr); ok {
			// does or expression not contain any expression...
			return !slices.ContainsFunc(orExpr.Exprs, func(orSubExpr Expr) bool {
				// ... wich is contained in the and expression
				return slices.ContainsFunc(e.Exprs, func(andSubExpr Expr) bool {
					// return ExprSubsumes(andSubExpr, orSubExpr)
					return orSubExpr.hash() == andSubExpr.hash()
				})
			})
		}
		return true
	})

	// return remaining expression when its the only one
	if len(e.Exprs) == 1 {
		return e.Exprs[0]
	}

	return e
}
func (e OrExpr) Optimize() Expr {
	e.Exprs = slicest.Map(e.Exprs, func(expr Expr) Expr { return expr.Optimize() })

	// flatten nested or expressions // & negated and expressions
	e.Exprs = slicest.Flatten(slicest.Map(e.Exprs, func(expr Expr) []Expr {
		// if notExpr, ok := expr.(NotExpr); ok {
		// 	if _, ok := notExpr.Expr.(AndExpr); ok {
		// 		expr = notExpr.Flip()
		// 	}
		// }
		if expr, ok := expr.(OrExpr); ok {
			return expr.Exprs
		}
		return []Expr{expr}
	}))

	// deduplicate nested expressions
	e.Exprs = sliceDeduplicateFunc(e.Exprs, func(expr Expr) string { return expr.hash() })
	// e.Exprs = slicest.FilterI(e.Exprs, func(i1 int, expr Expr) bool {
	// 	// is expression not contained by any other expression in its parent expression
	// 	// return !slicest.ContainsI(e.Exprs, func(i2 int, otherExpr Expr) bool { return i1 != i2 && ExprSubsumes(otherExpr, expr) })
	// 	return !slicest.ContainsI(e.Exprs, func(i2 int, otherExpr Expr) bool {
	// 		differentIndex := i1 != i2
	// 		subsumes := ExprSubsumes(otherExpr, expr)
	// 		if i1 > i2 {
	// 			return differentIndex && subsumes && !ExprSubsumes(expr, otherExpr)
	// 		}
	// 		return differentIndex && subsumes
	// 	})
	// })

	// remove redundant nested and expressions
	e.Exprs = slicest.Filter(e.Exprs, func(expr Expr) bool {
		if andExpr, ok := expr.(AndExpr); ok {
			// does and expression contain any expression...
			return !slices.ContainsFunc(andExpr.Exprs, func(andSubExpr Expr) bool {
				// ... wich is contained in the or expression
				return slices.ContainsFunc(e.Exprs, func(orSubExpr Expr) bool {
					// return ExprSubsumes(orSubExpr, andSubExpr)
					return andSubExpr.hash() == orSubExpr.hash()
				})
			})
		}
		return true
	})

	// return remaining expression when its the only one
	if len(e.Exprs) == 1 {
		return e.Exprs[0]
	}

	return e
}
func (e NotExpr) Optimize() Expr {
	e.Expr = e.Expr.Optimize()

	// remove double negation
	if expr, ok := e.Expr.(NotExpr); ok {
		return expr.Expr
	}

	return e
}
