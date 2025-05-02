// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/exprfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetEvaluator", func() {

	It("sets the evaluator", func() {
		ev := new(exprfakes.FakeEvaluator)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "e",
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr",
								Args: cli.Args("a", cli.Bool()),
								Uses: expr.SetEvaluator(ev),
							},
						},
					},
				},
			},
		}

		_, _ = app.Initialize(context.Background())

		expression := app.Args[0].Value.(*expr.Expression)
		Expect(expression.Exprs[0].Evaluate).To(BeIdenticalTo(ev))
	})
})

var _ = Describe("AddExpr", func() {

	It("inserts the expr on arg Uses pipeline", func() {
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{},
					Uses:  expr.AddExpr(&expr.Expr{Name: "expr"}),
				},
			},
		}

		_, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Args[0].Value.(*expr.Expression).Exprs).To(HaveLen(1))
	})

	It("inserts the expr on expr Uses pipeline", func() {
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr1",
								Uses: expr.AddExpr(&expr.Expr{Name: "expr2"}),
							},
						},
					},
				},
			},
		}
		_, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Args[0].Value.(*expr.Expression).Exprs).To(HaveLen(2))
	})
})
