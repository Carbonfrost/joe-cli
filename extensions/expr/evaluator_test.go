// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr_test

import (
	"context"
	"errors"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/internal/exprfakes"
	"github.com/Carbonfrost/joe-cli/internal/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Do", func() {

	It("invokes the action", func() {
		act1 := new(joeclifakes.FakeAction)
		act2 := new(joeclifakes.FakeAction)
		generic := new(exprfakes.FakeEvaluator)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "start",
					NArg: -2,
				},
				{
					Name: "e",
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "act1",
								Args: []*cli.Arg{
									{NArg: 1},
								},
								Evaluate: expr.Do(act1),
							},
							{
								Name: "act2",
								Args: []*cli.Arg{
									{NArg: 1},
								},
								Evaluate: expr.EvaluatorOf(act2),
							},
							{
								Name: "expr",
								Args: []*cli.Arg{
									{NArg: 1},
								},
								Evaluate: generic,
							}},
					},
				},
			},
			Action: expr.Evaluate("item"),
		}
		args, _ := cli.Split("app _ -act1 1 -act2 2 -expr true")
		err := app.RunContext(context.Background(), args...)
		Expect(err).NotTo(HaveOccurred())

		Expect(act1.ExecuteCallCount()).To(Equal(1))
		Expect(act2.ExecuteCallCount()).To(Equal(1))

		_, item, _ := generic.EvaluateArgsForCall(0)
		Expect(item).To(Equal("item"))
	})

})

var _ = Describe("NewEvaluator", func() {

	It("passes the typed value and yields", func() {
		var (
			seen    string
			yielded any
		)
		ev := expr.NewEvaluator(func(_ context.Context, v string, yield func(any) error) error {
			seen = v
			return yield(v + "!")
		})

		err := ev.Evaluate(context.Background(), "item", func(v any) error {
			yielded = v
			return nil
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(seen).To(Equal("item"))
		Expect(yielded).To(Equal("item!"))
	})

})

var _ = Describe("NewEvaluator0", func() {

	It("passes the typed value and yields it on success", func() {
		var (
			seen    string
			yielded any
		)
		ev := expr.NewEvaluator0(func(_ context.Context, v string) error {
			seen = v
			return nil
		})

		err := ev.Evaluate(context.Background(), "item", func(v any) error {
			yielded = v
			return nil
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(seen).To(Equal("item"))
		Expect(yielded).To(Equal("item"))
	})

	It("stops evaluation when the function returns an error", func() {
		var yielded bool
		ev := expr.NewEvaluator0(func(_ context.Context, _ string) error {
			return errors.New("boom")
		})

		err := ev.Evaluate(context.Background(), "item", func(any) error {
			yielded = true
			return nil
		})

		Expect(err).To(MatchError("boom"))
		Expect(yielded).To(BeFalse())
	})

})
