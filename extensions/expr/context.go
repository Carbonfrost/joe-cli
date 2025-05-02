// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr

import (
	"github.com/Carbonfrost/joe-cli"
)

// FromContext obtains the expression from the context
func FromContext(c *cli.Context, name string) *Expression {
	return c.Value(name).(*Expression)
}

// SetEvaluator provides an action used in the Uses pipeline of the Expr
// which updates its evaluator
func SetEvaluator(e Evaluator) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		c.Target().(*Expr).Evaluate = e
		return nil
	})
}

// AddExpr will add an expression operator to the containing Expression
func AddExpr(e *Expr) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		return updateExprs(c, func(ee []*Expr) []*Expr {
			return append(ee, e)
		})
	})
}

// AddExprs will add multiple expression operators to the containing Expression
func AddExprs(exprs ...*Expr) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		return updateExprs(c, func(ee []*Expr) []*Expr {
			return append(ee, exprs...)
		})
	})
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
