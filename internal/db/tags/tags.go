package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/uptrace/bun"
)

var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-+*/\\.:~=<>]+$`)

// TODO vendor out to seperate package
func reducex[T any, S ~[]T, U any](s S, f func(T, U) (U, error)) (U, error) {
	var zero U
	var result U
	for _, t := range s {
		var err error
		result, err = f(t, result)
		if err != nil {
			return zero, err
		}
	}
	return result, nil
}

func parseTag(expr string, qb bun.QueryBuilder, mode bool, negate bool) (bun.QueryBuilder, error) {
	var err error

	expr = strings.TrimSpace(expr)

	// and
	if exprs := splitOnTopLevelChar(expr, '&'); len(exprs) > 1 {
		// TODO test
		return reducex(exprs, func(expr string, qb bun.QueryBuilder) (bun.QueryBuilder, error) {
			return parseTag(expr, qb, true != negate, negate)
		})

		// for _, expr = range exprs {
		// 	qb, err = parseTag(expr, qb, true != negate, negate)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		// return qb, nil
	}

	// or
	if exprs := splitOnTopLevelChar(expr, '|'); len(exprs) > 1 {
		// TODO test
		return reducex(exprs, func(expr string, qb bun.QueryBuilder) (bun.QueryBuilder, error) {
			return parseTag(expr, qb, false != negate, negate)
		})

		// for _, expr = range exprs {
		// 	qb, err = parseTag(expr, qb, false != negate, negate)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }
		// return qb, nil
	}

	// negation
	expr, negated := strings.CutPrefix(expr, "!")

	expr = strings.TrimSpace(expr)

	// braces
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = expr[1 : len(expr)-1] // removes braces

		operator := map[bool]string{
			true:  " AND ",
			false: " OR ",
		}[mode]

		if negated {
			// return nil, fmt.Errorf("negating braces is unsupported: %s", expr)
			// Does not work because bun is a *****....
			// ... is what i would say, but i didn't even check if sql supports it.
			// operator += "NOT "

			// well, i think i got an idea ^^
			negate = !negate
		}

		qb = qb.WhereGroup(operator, func(qb bun.QueryBuilder) bun.QueryBuilder {
			qb, err = parseTag(expr, qb, true != negate, negate)
			return qb
		})

		if err != nil {
			return nil, err
		}
		return qb, nil
	}

	// raw tag value
	{
		if !tagPattern.MatchString(expr) {
			return nil, fmt.Errorf("invalid tag: %s", expr)
		}

		query := map[bool]string{
			true:  "tag <> ?",
			false: "tag = ?",
		}[negated != negate]

		if mode {
			return qb.Where(query, expr), nil
		} else {
			return qb.WhereOr(query, expr), nil
		}
	}
}

func splitOnTopLevelChar(expr string, op rune) []string {
	var result []string
	depth := 0
	start := 0

	for i, ch := range expr {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case op:
			if depth == 0 {
				result = append(result, expr[start:i])
				start = i + 1
			}
		}
	}

	result = append(result, expr[start:])
	return result
}

func ValidateTag(tag string) error {
	sq := &bun.SelectQuery{}
	_, err := parseTag(tag, sq.QueryBuilder(), true, false)
	return err
}

func GetTagQueryBuilder(tag string) (func(bun.QueryBuilder) bun.QueryBuilder, error) {
	if err := ValidateTag(tag); err != nil {
		return nil, err
	}
	return func(qb bun.QueryBuilder) bun.QueryBuilder {
		qb, _ = parseTag(tag, qb, true, false)
		return qb
	}, nil
}
