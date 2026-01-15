package tags

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/uptrace/bun"
)

const tagDelimiterChar string = "|"
const tagMatcherPatternExp string = `^[a-zA-Z0-9_\-+*/.:~=]+$`
const sqlEscapeChar string = "!"
const sqlEscapedChars string = `%_[]^-{}`

var tagMatcherPattern = regexp.MustCompile(tagMatcherPatternExp)

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

func parseTagMatcher(expr string, qb bun.QueryBuilder, mode bool, negate bool) (bun.QueryBuilder, error) {
	var err error

	expr = strings.TrimSpace(expr)

	// and
	if exprs := splitOnTopLevelChar(expr, '&'); len(exprs) > 1 {
		// TODO test & comment
		return reducex(exprs, func(expr string, qb bun.QueryBuilder) (bun.QueryBuilder, error) {
			return parseTagMatcher(expr, qb, !negate, negate)
		})
	}

	// or
	if exprs := splitOnTopLevelChar(expr, '|'); len(exprs) > 1 {
		// TODO test & comment
		return reducex(exprs, func(expr string, qb bun.QueryBuilder) (bun.QueryBuilder, error) {
			return parseTagMatcher(expr, qb, negate, negate)
		})
	}

	// negation
	expr, negated := strings.CutPrefix(expr, "!")

	expr = strings.TrimSpace(expr)

	// braces
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		// removes braces
		expr = expr[1 : len(expr)-1]
		// get WhereGroup prefix
		operator := map[bool]string{
			true:  " AND ",
			false: " OR ",
		}[mode]
		// flip negate flag for braces parsing, when braces are negated
		if negated {
			// return nil, fmt.Errorf("negating braces is unsupported: %s", expr)
			// Does not work because bun is a *****....
			// ... is what i would say, but i didn't even check if sql supports it.
			// operator += "NOT "

			// well, i think i got an idea ^^
			negate = !negate
		}
		// apply WhereGroup to query builder
		qb = qb.WhereGroup(operator, func(qb bun.QueryBuilder) bun.QueryBuilder {
			qb, err = parseTagMatcher(expr, qb, !negate, negate)
			return qb
		})
		// handle error from WhereGroup callback using global err variable
		if err != nil {
			return nil, err
		}
		return qb, nil
	}

	// raw tag value
	{
		// validate against tagPattern
		if !tagMatcherPattern.MatchString(expr) {
			return nil, fmt.Errorf("invalid tag: %s", expr)
		}
		// escape special chars just to be sure
		for _, c := range sqlEscapedChars {
			expr = strings.ReplaceAll(expr, string(c), sqlEscapeChar+string(c))
		}
		// enable wildcards
		expr = strings.ReplaceAll(expr, "**", "%")
		expr = strings.ReplaceAll(expr, "*", "_")
		// add delimiters
		expr = JoinTags([]string{expr})
		// construct query
		query := map[bool]string{
			true:  "tag NOT LIKE ? ESCAPE '" + sqlEscapeChar + "'",
			false: "tag     LIKE ? ESCAPE '" + sqlEscapeChar + "'",
		}[negated != negate]
		// apply to query builder
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

	return append(result, expr[start:])
}

func ValidateTagMatcher(tag_matcher string) error {
	// create mock QueryBuilder (fine for validation, but panics when used to render sql as it has no underlying formatter!)
	qb := (&bun.SelectQuery{}).QueryBuilder()
	_, err := parseTagMatcher(tag_matcher, qb, true, false)
	return err
}

func QueryBuilderFromTagMatcher(tag_matcher string) (func(bun.QueryBuilder) bun.QueryBuilder, error) {
	// validate before returning QueryBuilder, because errors can't be returned from the QueryBuilder callback
	if err := ValidateTagMatcher(tag_matcher); err != nil {
		return nil, err
	}
	// return QueryBuilder with safe callback
	return func(qb bun.QueryBuilder) bun.QueryBuilder {
		qb, _ = parseTagMatcher(tag_matcher, qb, true, false)
		return qb
	}, nil
}

func SplitTags(tags string) ([]string, error) {
	// validate and strip prefix & suffix
	tags, exists_prefix := strings.CutPrefix(tags, tagDelimiterChar)
	tags, exists_suffix := strings.CutSuffix(tags, tagDelimiterChar)
	if !exists_prefix || !exists_suffix {
		return nil, errors.New("prefix or suffix is missing")
	}
	// split and return tags
	return strings.Split(tags, tagDelimiterChar), nil
}

func SplitTagsSafe(tag string) []string {
	tags, err := SplitTags(tag)
	if err != nil {
		return []string{}
	}
	return tags
}

func JoinTags(tags []string) string {
	return tagDelimiterChar + strings.Join(tags, tagDelimiterChar) + tagDelimiterChar
}
