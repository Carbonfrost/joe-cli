// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operator

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carbonfrost/joe-cli/extensions/expr"
)

// BindingEvaluator provides the relationship between an evaluator and the evaluation context.
type BindingEvaluator = expr.BindingEvaluator

type liftedEvaluator struct {
	evaluator Evaluator
}

type compiler struct {
	items []BindingEvaluator
	pos   int
}

var (
	errEmptyParens = errors.New("invalid expression: empty parentheses are not allowed")
	errUnclosed    = errors.New("invalid expression: expected ')'")
)

var _ expr.Compiler = Compile

// Compile provides a compiler for an expression which interprets the operators
// ( ) ! -and -or, which are parsed when the expression has the option
// expr.ParseOperators.  Operators are converted to Not, And, and Or.
// Sub-expressions which use Predicate, PredicateContext, or Invariant as the
// evaluator are used directly.  Any other evaluator is lifted into a
// sub-expression which is true when the evaluator yields and false otherwise;
// in either case, the input value (rather than the value that was yielded)
// continues through the pipeline.  When no binary operator occurs between
// sub-expressions, -and is implied.  The usual precedence applies: ! binds
// tightest, then -and, then -or.
//
// An error is returned when the operators are not well-formed, such as a
// binary operator without a left or right operand, an unmatched parenthesis, or
// empty parentheses.  When there are no binding evaluators, the resulting
// evaluator yields every value.
func Compile(items []BindingEvaluator) (Evaluator, error) {
	if len(items) == 0 {
		return expr.AlwaysTrue, nil
	}

	c := &compiler{items: items}
	res, err := c.parseOr()
	if err != nil {
		return nil, err
	}
	if tok, ok := c.peek(); ok {
		return nil, fmt.Errorf("invalid expression: unexpected %v", tok)
	}
	return res, nil
}

// Lift converts an evaluator into a sub-expression which is true when the
// evaluator yields and false otherwise.  The resulting evaluator yields the
// input value at most once.
func Lift(e Evaluator) Evaluator {
	return &liftedEvaluator{evaluator: e}
}

func (c *compiler) peek() (BindingEvaluator, bool) {
	if c.pos >= len(c.items) {
		return nil, false
	}
	return c.items[c.pos], true
}

func (c *compiler) peekOperator(op expr.Operator) bool {
	tok, ok := c.peek()
	return ok && tok == op
}

// parseOr handles or := and ( -or and )*
func (c *compiler) parseOr() (Evaluator, error) {
	left, err := c.parseAnd()
	if err != nil {
		return nil, err
	}
	for c.peekOperator(expr.Or) {
		c.pos++
		right, err := c.parseAnd()
		if err != nil {
			return nil, err
		}
		left = Or(left, right)
	}
	return left, nil
}

// parseAnd handles and := unary ( [-and] unary )*
func (c *compiler) parseAnd() (Evaluator, error) {
	left, err := c.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		tok, ok := c.peek()
		if !ok || tok == expr.Or || tok == expr.RParen {
			return left, nil
		}
		if tok == expr.And {
			c.pos++
		}
		right, err := c.parseUnary()
		if err != nil {
			return nil, err
		}
		left = And(left, right)
	}
}

// parseUnary handles unary := ! unary | ( or ) | sub-expression
func (c *compiler) parseUnary() (Evaluator, error) {
	tok, ok := c.peek()
	if !ok {
		return nil, c.missingOperand()
	}

	switch tok {
	case expr.Not:
		c.pos++
		operand, err := c.parseUnary()
		if err != nil {
			return nil, err
		}
		return Not(operand), nil

	case expr.LParen:
		c.pos++
		if c.peekOperator(expr.RParen) {
			return nil, errEmptyParens
		}
		inner, err := c.parseOr()
		if err != nil {
			return nil, err
		}
		if !c.peekOperator(expr.RParen) {
			return nil, errUnclosed
		}
		c.pos++
		return inner, nil

	case expr.And, expr.Or, expr.RParen:
		return nil, c.missingOperand()
	}

	c.pos++
	return subExpression(tok), nil
}

func (c *compiler) missingOperand() error {
	if tok, ok := c.peek(); ok {
		if c.pos == 0 {
			return fmt.Errorf("invalid expression: expected an expression before %v", tok)
		}
		return fmt.Errorf("invalid expression: expected an expression between %v and %v", c.items[c.pos-1], tok)
	}
	if c.pos > 0 {
		if op, ok := c.items[c.pos-1].(expr.Operator); ok {
			return fmt.Errorf("invalid expression: expected an expression after %v", op)
		}
	}
	return errors.New("invalid expression: expected an expression")
}

func subExpression(b BindingEvaluator) Evaluator {
	if isPredicate(b) {
		return b
	}
	return Lift(b)
}

func isPredicate(b BindingEvaluator) bool {
	x := b.Expr()
	if x == nil {
		return false
	}
	switch expr.EvaluatorOf(x.Evaluate).(type) {
	case expr.Predicate, expr.PredicateContext, expr.Invariant:
		return true
	}
	return false
}

func (l *liftedEvaluator) Evaluate(c context.Context, v any, yield func(any) error) error {
	var yielded bool
	err := l.evaluator.Evaluate(c, v, func(any) error {
		yielded = true
		return nil
	})
	if err != nil {
		return err
	}
	if yielded {
		return yield(v)
	}
	return nil
}
