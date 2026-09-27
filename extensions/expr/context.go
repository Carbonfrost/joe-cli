// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

// Action represents the building block of the various actions that
// the expr extension supports
type Action = cli.Action

// FromContext obtains the expression from the context.
// When nameopt is not specified or is the empty string, then the expression
// from the current arg or apparent from one of the args in the current command
// is returned.
func FromContext(c context.Context, nameopt ...string) *Expression {
	if len(nameopt) == 0 || nameopt[0] == "" {
		ctx := cli.FromContext(c)
		if ctx.IsCommand() {
			for _, arg := range ctx.LocalArgs() {
				e := exprFromArg(arg)
				if e != nil {
					return e
				}
			}
		}
		return exprFromArg(ctx.Arg())
	}
	return c.Value(nameopt[0]).(*Expression)
}

func exprFromArg(a *cli.Arg) *Expression {
	if a == nil {
		return nil
	}
	result, _ := a.Value.(*Expression)
	return result
}

// SetEvaluator provides an action used in the Uses pipeline of the Expr
// which updates its evaluator
func SetEvaluator(e Evaluator) Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		c.Target().(*Expr).Evaluate = e
		return nil
	})
}

// AddExpr will add an expression operator to the containing Expression
func AddExpr(e *Expr) Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		return updateExprs(c, func(ee []*Expr) []*Expr {
			return append(ee, e)
		})
	})
}

// AddExprs will add multiple expression operators to the containing Expression
func AddExprs(exprs ...*Expr) Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		return updateExprs(c, func(ee []*Expr) []*Expr {
			return append(ee, exprs...)
		})
	})
}

// Evaluate provides an action which evaluates the expression with the given input
// items. The expression is retrieved from the arg in scope.
func Evaluate(items ...any) Action {
	return bind.Call2(
		exprEvaluate, bind.Context(), bind.Exact(items),
	)
}

// EvaluateParallel provides an action which evaluates the expression with the given input
// items. The expression is retrieved from the arg in scope and runs
// in parallel. By convention, if this is present within the Uses pipeline, it
// registers a flag to configure the number of jobs.
func EvaluateParallel(items ...any) Action {
	var useJobs = bind.NewActionBinder(
		cli.AddFlag(&cli.Flag{
			Name:     "jobs",
			HelpText: "Maximum {NUMBER} of jobs to run in parallel",
			Uses:     cli.OptionalAlias("j"),
		}),
		bind.Int("jobs"),
	)
	return bind.Call3(
		exprEvaluateParallel, bind.Context(), useJobs, bind.Exact(items),
	)
}

func exprEvaluate(ctx *cli.Context, items []any) error {
	return FromContext(ctx).Evaluate(ctx, items...)
}

func exprEvaluateParallel(ctx *cli.Context, jobs int, items []any) error {
	return FromContext(ctx).EvaluateParallel(ctx, jobs, items...)
}

func updateExprs(c *cli.Context, fn func([]*Expr) []*Expr) error {
	if err := requireInit(c); err != nil {
		return err
	}
	exp := c.Arg().Value.(*Expression)
	exp.Exprs = fn(exp.Exprs)
	return nil
}

func requireInit(c *cli.Context) error {
	if !c.IsInitializing() {
		return newInternalError(c, cli.ErrTimingTooLate)
	}
	return nil
}

func newInternalError(c *cli.Context, err error) error {
	return &cli.InternalError{Path: c.Path(), Timing: c.Timing(), Err: err}
}
