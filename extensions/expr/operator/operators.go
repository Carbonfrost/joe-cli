// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package operator provides the small predicate expression language.
package operator

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
)

// Evaluator provides the evaluation function for an expression operator.
type Evaluator = expr.Evaluator

type notEvaluator struct {
	evaluator Evaluator
}

type orEvaluator struct {
	left  Evaluator
	right Evaluator
}

type andEvaluator struct {
	left  Evaluator
	right Evaluator
}

// Not creates an evaluator that inverts the evaluation result of the given evaluator.
// The evaluator implements the unary NOT operator. It inverts the evaluation
// result: if the wrapped evaluator yields a value, the evaluator does not yield;
// if the wrapped evaluator does not yield, the evaluator yields the input value.
func Not(evaluator Evaluator) Evaluator {
	return &notEvaluator{
		evaluator: evaluator,
	}
}

// Or creates an evaluator that performs logical OR with short-circuit evaluation.
// It evaluates the left evaluator first. If the left evaluator yields a value,
// the evaluator yields and does not evaluate the right evaluator. If the left
// evaluator does not yield, the right evaluator is evaluated.
func Or(left, right Evaluator) Evaluator {
	return &orEvaluator{
		left:  left,
		right: right,
	}
}

// And creates an evaluator that performs explicit logical AND with short-circuit evaluation.
// The evaluator implements the explicit logical AND operator. Both evaluators
// must yield for the result to be yielded. The right evaluator is only
// evaluated if the left evaluator yields (short-circuit evaluation).
// The expression pipeline already provides implicit AND semantics through
// juxtaposition, but this evaluator can be used for explicit AND operations.
func And(left, right Evaluator) Evaluator {
	return &andEvaluator{
		left:  left,
		right: right,
	}
}

func (n *notEvaluator) Evaluate(c context.Context, v any, yield func(any) error) error {
	var yielded bool
	wrappedYield := func(val any) error {
		yielded = true
		return nil
	}

	err := n.evaluator.Evaluate(c, v, wrappedYield)
	if err != nil {
		return err
	}

	if !yielded {
		return yield(v)
	}
	return nil
}

func (o *orEvaluator) Evaluate(c context.Context, v any, yield func(any) error) error {
	var yielded bool
	wrappedYield := func(val any) error {
		yielded = true
		return yield(val)
	}

	err := o.left.Evaluate(c, v, wrappedYield)
	if err != nil {
		return err
	}

	if yielded {
		return nil
	}

	return o.right.Evaluate(c, v, yield)
}

func (a *orEvaluator) Initializer() cli.Action {
	return &cli.Prototype{
		Name:     "or",
		HelpText: "The logical OR operator",
		Uses:     cli.OptionalAlias("o"),
	}
}

func (a *andEvaluator) Evaluate(c context.Context, v any, yield func(any) error) error {
	var yielded bool
	var yieldedValue any

	wrappedYield := func(val any) error {
		yielded = true
		yieldedValue = val
		return nil
	}

	err := a.left.Evaluate(c, v, wrappedYield)
	if err != nil {
		return err
	}

	if !yielded {
		return nil
	}

	return a.right.Evaluate(c, yieldedValue, yield)
}

func (a *andEvaluator) Initializer() cli.Action {
	return &cli.Prototype{
		Name:     "and",
		HelpText: "The logical AND operator",
		Uses:     cli.OptionalAlias("a"),
	}
}
