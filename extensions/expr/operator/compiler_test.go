// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package operator_test

import (
	"context"
	"fmt"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/operator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Compile", func() {

	var (
		captured []string
		yielded  bool
	)

	newApp := func(options cli.Option, compiler expr.Compiler) *cli.App {
		return &cli.App{
			Name: "find",
			Args: []*cli.Arg{
				{
					// A single start path because TakeUntilNextFlag would
					// consume leading operators such as ! and (
					Name: "f",
				},
				{
					Name:    "e",
					Options: options,
					Value: &expr.Expression{
						Compiler: compiler,
						Exprs: []*expr.Expr{
							{Evaluate: expr.AlwaysTrue},
							{Evaluate: expr.AlwaysFalse},
							{
								Name: "print",
								Evaluate: func(v any) {
									captured = append(captured, fmt.Sprintf("print(%v)", v))
								},
							},
							{
								Name: "twice",
								Evaluate: func(v any, yield func(any) error) error {
									captured = append(captured, "twice")
									if err := yield("first"); err != nil {
										return err
									}
									return yield("second")
								},
							},
						},
					},
				},
			},
			Action: func(c *cli.Context) error {
				return expr.FromContext(c, "e").Evaluate(c, "in")
			},
		}
	}

	recordingCompiler := func(items []expr.BindingEvaluator) (expr.Evaluator, error) {
		pipe, err := operator.Compile(items)
		if err != nil {
			return nil, err
		}
		return expr.EvaluatorOf(func(c context.Context, v any, _ func(any) error) error {
			return pipe.Evaluate(c, v, func(any) error {
				yielded = true
				return nil
			})
		}), nil
	}

	run := func(app *cli.App, arguments string) error {
		args, _ := cli.Split("find . " + arguments)
		return app.RunContext(context.Background(), args...)
	}

	BeforeEach(func() {
		captured = nil
		yielded = false
	})

	DescribeTable("examples", func(arguments string, expectedYield bool, expected types.GomegaMatcher) {
		err := run(newApp(expr.ParseOperators, recordingCompiler), arguments)
		Expect(err).NotTo(HaveOccurred())
		Expect(yielded).To(Equal(expectedYield))
		Expect(strings.Join(captured, " ")).To(expected)
	},
		Entry("trivial pipeline", "-true", true, BeEmpty()),
		Entry("binary operator", "-true -and -true", true, BeEmpty()),
		Entry("binary operator alias", "-true -a -true", true, BeEmpty()),
		Entry("not operator", "! -true", false, BeEmpty()),
		Entry("implied and short circuits", "-false -true", false, BeEmpty()),
		Entry("implied and short circuits evaluator", "-false -print", false, BeEmpty()),
		Entry("non-predicate evaluators", "-print -and -true", true, Equal("print(in)")),
		Entry("grouped expressions", "( -true -or -false ) -and -print", true, Equal("print(in)")),
		Entry("or short circuits", "-true -or -print", true, BeEmpty()),
		Entry("or alias", "-false -o -print", true, Equal("print(in)")),
		Entry("and binds tighter than or", "-print -false -o -print", true, Equal("print(in) print(in)")),
		Entry("not binds tighter than and", "! -false -print", true, Equal("print(in)")),
		Entry("not of group", "! ( -false -o -false )", true, BeEmpty()),
		Entry("double not", "! ! -true", true, BeEmpty()),
		Entry("nested groups", "( ( -false ) -o ( -print ) )", true, Equal("print(in)")),
		Entry("lifted evaluator yields input once", "-twice -print", true, Equal("twice print(in)")),
	)

	DescribeTable("errors", func(arguments string, expected string) {
		err := run(newApp(expr.ParseOperators, operator.Compile), arguments)
		Expect(err).To(MatchError(expected))
	},
		Entry("trailing binary operator", "-true -and", "invalid expression: expected an expression after -and"),
		Entry("trailing not operator", "-true !", "invalid expression: expected an expression after !"),
		Entry("leading binary operator", "-or -true", "invalid expression: expected an expression before -or"),
		Entry("consecutive binary operators", "-true -or -and -true", "invalid expression: expected an expression between -or and -and"),
		Entry("unclosed paren", "( -true", "invalid expression: expected ')'"),
		Entry("unmatched close paren", "-true )", "invalid expression: unexpected )"),
		Entry("empty parens", "( )", "invalid expression: empty parentheses are not allowed"),
		Entry("binary operator after open paren", "( -a -true )", "invalid expression: expected an expression between ( and -and"),
	)

	It("does not parse operators without ParseOperators", func() {
		err := run(newApp(0, operator.Compile), "-true ! -true")
		Expect(err).To(MatchError(ContainSubstring("arguments must precede expressions")))
	})

	It("fails when operators are used with the default compiler", func() {
		err := run(newApp(expr.ParseOperators, nil), "! -true")
		Expect(err).To(MatchError("default compiler does not support operators: !"))
	})

	It("yields every value when there are no binding evaluators", func() {
		pipe, err := operator.Compile(nil)
		Expect(err).NotTo(HaveOccurred())

		var actual []any
		_ = pipe.Evaluate(context.Background(), "in", func(v any) error {
			actual = append(actual, v)
			return nil
		})
		Expect(actual).To(Equal([]any{"in"}))
	})

	It("compiles operator placeholders directly", func() {
		pipe, err := operator.Compile([]expr.BindingEvaluator{
			expr.Not,
			expr.NewBindingEvaluator(expr.AlwaysTrue),
			expr.Or,
			expr.NewBindingEvaluator(expr.AlwaysTrue),
		})
		Expect(err).NotTo(HaveOccurred())

		var actual []any
		_ = pipe.Evaluate(context.Background(), "in", func(v any) error {
			actual = append(actual, v)
			return nil
		})
		Expect(actual).To(Equal([]any{"in"}))
	})
})

var _ = Describe("Lift", func() {

	It("yields the input value when the evaluator yields", func() {
		var actual []any
		e := operator.Lift(expr.EvaluatorOf(func(_ any, yield func(any) error) error {
			_ = yield("a")
			return yield("b")
		}))

		_ = e.Evaluate(context.Background(), "in", func(v any) error {
			actual = append(actual, v)
			return nil
		})
		Expect(actual).To(Equal([]any{"in"}))
	})

	It("does not yield when the evaluator does not yield", func() {
		var called bool
		e := operator.Lift(expr.AlwaysFalse)

		_ = e.Evaluate(context.Background(), "in", func(any) error {
			called = true
			return nil
		})
		Expect(called).To(BeFalse())
	})
})
