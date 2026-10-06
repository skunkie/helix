// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package search

import (
	"errors"
)

func consumeToken(tokens []token) (token, []token) {
	if len(tokens) == 0 {
		return token{eof, ""}, nil
	}
	return tokens[0], tokens[1:]
}
func peekToken(tokens []token) token {
	if len(tokens) == 0 {
		return token{eof, ""}
	}
	return tokens[0]
}

func query(tokens []token) (Criteria, error) {
	var err error
	var expr Expr
	switch peekToken(tokens).Kind {
	case asterisk:
		_, tokens = consumeToken(tokens)
		if token, _ := consumeToken(tokens); token.Kind != eof {
			return nil, errors.New("asterisk must be alone")
		}
		return Everything{}, nil
	default:
		tokens, expr, err = expression(tokens)
		if err != nil {
			return nil, err
		}
		if token, _ := consumeToken(tokens); token.Kind != eof {
			return nil, errors.New("unexpected end of query")
		}
		return Query{expr}, nil
	}
}

// parser consumes a prefix of tokens and returns the remaining tokens with the
// parsed expression.
type parser func(tokens []token) ([]token, Expr, error)

// expression parses `or`-separated groups of `and`-separated terms, so `and`
// binds more tightly than `or` as the ContentDirectory grammar requires.
func expression(tokens []token) ([]token, Expr, error) {
	return logicExpression(tokens, Or, andExpression)
}

func andExpression(tokens []token) ([]token, Expr, error) {
	return logicExpression(tokens, And, term)
}

// logicExpression parses one or more operands joined by op into a single flat
// LogicExpr, returning a lone operand unwrapped.
func logicExpression(tokens []token, op LogicOp, operand parser) ([]token, Expr, error) {
	var exprs []Expr
	for {
		var err error
		var expr Expr
		tokens, expr, err = operand(tokens)
		if err != nil {
			return nil, nil, err
		}
		exprs = append(exprs, expr)

		if peekToken(tokens) != (token{bareString, string(op)}) {
			break
		}
		_, tokens = consumeToken(tokens)
	}
	if len(exprs) == 1 {
		return tokens, exprs[0], nil
	}
	return tokens, LogicExpr{op, exprs}, nil
}

func term(tokens []token) ([]token, Expr, error) {
	if peekToken(tokens).Kind != openParenthesis {
		return binaryExpression(tokens)
	}
	_, tokens = consumeToken(tokens)
	tokens, expr, err := expression(tokens)
	if err != nil {
		return nil, nil, err
	}
	var t token
	if t, tokens = consumeToken(tokens); t.Kind != closeParenthesis {
		return nil, nil, errors.New("could not match '(' with ')'")
	}
	return tokens, expr, nil
}

func binaryExpression(tokens []token) ([]token, Expr, error) {
	var t token
	t, tokens = consumeToken(tokens)
	if t.Kind != bareString {
		return nil, nil, errors.New("expected property")
	}
	property := t.Text

	var op BinaryOp
	switch t, tokens = consumeToken(tokens); t {
	case token{bareString, "exists"}:
		switch t, tokens = consumeToken(tokens); t {
		case token{bareString, "true"}:
			return tokens, ExistsExpr{property, true}, nil
		case token{bareString, "false"}:
			return tokens, ExistsExpr{property, false}, nil
		default:
			return nil, nil, errors.New("expected true or false")
		}
	case token{equal, "="}:
		op = Equal
	case token{notEqual, "!="}:
		op = NotEqual
	case token{greaterThan, ">"}:
		op = GreaterThan
	case token{greaterThanEqual, ">="}:
		op = GreaterThanEqual
	case token{lessThan, "<"}:
		op = LessThan
	case token{lessThanEqual, "<="}:
		op = LessThanEqual
	case token{bareString, "contains"}:
		op = Contains
	case token{bareString, "doesNotContain"}:
		op = DoesNotContain
	case token{bareString, "derivedfrom"}:
		op = DerivedFrom
	default:
		return nil, nil, errors.New("unexpected binary operation")
	}

	t, tokens = consumeToken(tokens)
	if t.Kind != quotedString {
		return nil, nil, errors.New("expected quoted string")
	}
	operand := t.Text

	return tokens, BinaryExpr{property, op, operand}, nil
}
