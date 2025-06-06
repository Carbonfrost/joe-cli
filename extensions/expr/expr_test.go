// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/exprfakes"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Expr", func() {

	It("context contains the expression", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Action: act,
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
								Name: "expr",
								Args: cli.Args("a", cli.Bool()),
							},
						},
					},
				},
			},
		}
		args, _ := cli.Split("app x -expr true")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())

		captured := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(captured.Value("e")).NotTo(BeNil())
	})

	Describe("prototype support", func() {

		DescribeTable("examples", func(proto cli.Prototype, expected Fields) {
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
									Name: "expr",
									Args: cli.Args("a", cli.Bool()),
									Uses: proto,
									Data: map[string]any{"A": 1},
								},
							},
						},
					},
				},
			}
			args, _ := cli.Split("app x -expr true")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())

			_ = app.RunContext(context.Background(), []string{"app"})
			expr := app.Args[1].Value.(*expr.Expression).Exprs[0]
			Expect(expr).To(PointTo(MatchFields(IgnoreExtras, expected)))
		},
			Entry("Description", cli.Prototype{Description: "d"}, Fields{"Description": Equal("d")}),
			Entry("Category", cli.Prototype{Category: "f"}, Fields{"Category": Equal("f")}),
			Entry("HelpText", cli.Prototype{HelpText: "new help text"}, Fields{"HelpText": Equal("new help text")}),
			Entry("ManualText", cli.Prototype{ManualText: "explain"}, Fields{"ManualText": Equal("explain")}),
			Entry("UsageText", cli.Prototype{UsageText: "nom"}, Fields{"UsageText": Equal("nom")}),
			Entry("Data", cli.Prototype{Data: map[string]any{"B": 3}}, Fields{"Data": Equal(map[string]any{"A": 1, "B": 3})}),
			Entry("Aliases", cli.Prototype{Aliases: []string{"e", "f"}}, Fields{"Aliases": Equal([]string{"e", "f"})}),
		)

		It("copies Options from prototype", func() {
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
									Name: "expr",
									Args: cli.Args("a", cli.Bool()),
									Uses: cli.Prototype{Options: cli.Hidden},
									Data: map[string]any{"A": 1},
								},
							},
						},
					},
				},
			}
			args, _ := cli.Split("app x -expr true")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())

			_ = app.RunContext(context.Background(), []string{"app"})
			myExpr := app.Args[1].Value.(*expr.Expression).Exprs[0]
			Expect(expr.IsVisible(myExpr)).To(BeFalse())
		})
	})

	Describe("naming", func() {

		var (
			beInvalid = MatchError(ContainSubstring("not a valid name"))
		)

		DescribeTable("undefined behavior", func(e *expr.Expr) {
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{Name: "inuse", Aliases: []string{"alsoinuse"}},
							},
						},
						Uses: expr.AddExpr(e),
					},
				},
			}

			_, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())
		},
			Entry(
				"expr duplicates alias", &expr.Expr{Name: "alsoinuse"}),
		)

		DescribeTable("errors", func(e *expr.Expr, expected types.GomegaMatcher) {
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{Name: "inuse", Aliases: []string{"alsoinuse"}},
							},
						},
						Uses: expr.AddExpr(e),
					},
				},
			}

			_, err := app.Initialize(context.Background())
			Expect(err).To(expected)
		},
			Entry(
				"no name",
				&expr.Expr{},
				MatchError(ContainSubstring("expr at index #1 must have a name"))),
			Entry(
				"duplicate expr name",
				&expr.Expr{Name: "inuse"},
				MatchError(ContainSubstring(`duplicate name used: "inuse"`))),

			Entry("expr with dashes", &expr.Expr{Name: "expr-dash"}, Succeed()),
			Entry("expr with underscores", &expr.Expr{Name: "expr_under"}, Succeed()),
			Entry("expr with numeric", &expr.Expr{Name: "123"}, Succeed()),
			Entry("expr with special char @", &expr.Expr{Name: "@"}, Succeed()),
			Entry("expr with special char #", &expr.Expr{Name: "#"}, Succeed()),
			Entry("expr with special char *", &expr.Expr{Name: "*"}, Succeed()),
			Entry("expr with special char +", &expr.Expr{Name: "+"}, Succeed()),
			Entry("expr with special char :", &expr.Expr{Name: ":"}, Succeed()),
			Entry("expr with spaces", &expr.Expr{Name: "expr name with spaces"}, beInvalid),
			Entry("expr with invalid cahr", &expr.Expr{Name: "expr&expr"}, beInvalid),
		)
	})

	Describe("instancing", func() {

		DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
			var seen []any
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "more",
									Args: cli.Args("i", v),
									Evaluate: func(c *cli.Context, _ any) {
										val := c.Values()[0]
										// Some types need to be copied
										if cv, ok := val.(*cli.NameValue); ok {
											var v cli.NameValue = *cv
											val = &v
										}

										seen = append(seen, val)
										Expect(c.Values()).To(HaveLen(1))
									},
								},
							},
						},
					},
				},
				Action: act,
			}
			args, _ := cli.Split("app -- -more 1 -more 2 -more 3")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())

			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			expr.FromContext(captured, "expression").Evaluate(captured, 0)

			Expect(seen).To(expected)
		},
			Entry("int", new(int), Equal([]any{int(1), int(2), int(3)})),
			Entry("NameValue (resettable)", new(cli.NameValue), Equal([]any{
				&cli.NameValue{Name: "1", Value: "true"},
				&cli.NameValue{Name: "2", Value: "true"},
				&cli.NameValue{Name: "3", Value: "true"}},
			)),
		)

	})

	It("invokes the action on the arg", func() {
		act := new(joeclifakes.FakeAction)
		appAct := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr",
								Args: []*cli.Arg{
									{
										Name:   "a",
										Value:  cli.Bool(),
										Action: act,
									},
								},
							},
						},
					},
				},
			},
			Action: appAct,
		}
		args, _ := cli.Split("app -- -expr true")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))

		captured := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(captured.Value("")).To(Equal(true))
		Expect(captured.Command().Name).To(Equal("app"))
		Expect(captured.Path().String()).To(Equal("app <expression> <-expr> <a>"))
	})

	It("the action on the arg can resolve peer values", func() {
		act := new(joeclifakes.FakeAction)
		appAct := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr",
								Args: []*cli.Arg{
									{
										Name:   "a",
										Value:  cli.Bool(),
										Action: act,
									},
									{
										Name:  "b",
										Value: cli.String(),
									},
								},
							},
						},
					},
				},
			},
			Action: appAct,
		}
		args, _ := cli.Split("app -- -expr true blood")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))

		captured := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(captured.Value("b")).To(Equal("blood"))
	})

	It("invokes the action on the arg on each occurrence", func() {
		act := new(joeclifakes.FakeAction)
		data := []int{}
		act.ExecuteCalls(func(ctx context.Context) error {
			c := cli.FromContext(ctx)
			data = append(data, c.Int(""))
			return nil
		})
		appAct := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr",
								Args: []*cli.Arg{
									{
										Name:    "a",
										Value:   cli.Int(),
										Action:  act,
										Options: cli.EachOccurrence,
									},
								},
							},
						},
					},
				},
			},
			Action: appAct,
		}

		args, _ := cli.Split("app -- -expr 1 -expr 2 -expr 3")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(act.ExecuteCallCount()).To(Equal(3))
		Expect(data).To(Equal([]int{1, 2, 3}))
	})

	Describe("arg events are invoked", func() {

		DescribeTable("examples", func(middleware func(cli.Action) cli.Action) {
			act := new(joeclifakes.FakeAction)
			appAct := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Args: []*cli.Arg{
					{
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "expr",
									Args: []*cli.Arg{
										{
											Name:  "a",
											Value: cli.Bool(),
											Uses:  middleware(act),
										},
									},
								},
							},
						},
					},
				},
				Action: appAct,
			}
			args, _ := cli.Split("app -- -expr true")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())

			captured := cli.FromContext(appAct.ExecuteArgsForCall(0))
			expr.FromContext(captured, "expression").Evaluate(captured, 0)

			Expect(act.ExecuteCallCount()).To(Equal(1))

			captured = cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Path().String()).To(Equal("app <expression> <-expr> <a>"))
		},
			Entry("initializer", cli.Initializer),
			Entry("before", cli.Before),
			Entry("after", cli.After),
		)
	})

	It("names it expression by default", func() {
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr",
								Args: cli.Args("a", cli.Bool()),
							},
						},
					},
				},
			},
		}
		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Args[0].Name).To(Equal("expression"))
	})

	It("can add additional args dynamically", func() {
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "expr",
								Args: cli.Args("a", cli.Bool()),
								Uses: cli.AddArgs(cli.Args("o", new(int))...),
							},
						},
					},
				},
			},
		}

		app.Initialize(context.Background())
		expr := app.Args[0].Value.(*expr.Expression).Exprs[0]
		Expect(expr.Args).To(HaveLen(2))
	})

	It("is treats evaluator as Action if it implements it", func() {
		evaluatorWithAction := new(struct {
			cli.Action
			expr.Evaluator
		})
		act := new(joeclifakes.FakeAction)
		evaluatorWithAction.Action = act

		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Name: "f",
					NArg: 0,
				},
				{
					Name: "expression",
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name:     "expr",
								Args:     cli.Args("f", new(bool)),
								Evaluate: evaluatorWithAction,
							},
						},
					},
				},
			},
		}

		_, _ = app.Initialize(context.Background())
		Expect(act.ExecuteCallCount()).To(Equal(1))
	})

	Describe("Evaluate", func() {
		var (
			act *exprfakes.FakeEvaluator
			app *cli.App
		)

		BeforeEach(func() {
			act = new(exprfakes.FakeEvaluator)
			app = &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:  "flag",
						Value: new(int),
					},
				},
				Args: []*cli.Arg{
					{
						Name: "f",
						NArg: 0,
					},
					{
						Name: "e",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "expr",
									Args: []*cli.Arg{
										{
											Name:  "f",
											Value: new(bool),
											NArg:  1,
										},
										{
											Name:  "g",
											Value: new(int),
											NArg:  2,
										},
										{
											Name:  "h",
											Value: new([]string),
											NArg:  -2,
										},
									},
									Evaluate: act,
								},
							},
						},
					},
				},
				Action: func(c *cli.Context) {
					expr.FromContext(c, "e").Evaluate(c, "items")
				},
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split("app --flag 9 arg -expr true 1 2 a b c")
			app.RunContext(context.Background(), args)
		})

		It("executes action on setting Arg", func() {
			Expect(act.EvaluateCallCount()).To(Equal(1))
		})

		It("contains args in captured context", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(cli.FromContext(captured).Args()).To(Equal([]string{"true", "1", "2", "a", "b", "c"}))
		})

		It("contains values in captured context", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(cli.FromContext(captured).Values()).To(Equal([]any{true, 2, []string{"a", "b", "c"}}))
		})

		It("provides context Name", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(cli.FromContext(captured).Name()).To(Equal("<-expr>"))
		})

		It("provides context Path", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(cli.FromContext(captured).Path().String()).To(Equal("app <-expr>"))
		})

		It("provides values from flags", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(cli.FromContext(captured).Int("flag")).To(Equal(9))
		})

		It("provides bindings that were selected", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			exp := expr.FromContext(cli.FromContext(captured), "e")
			b := slices.Collect(exp.Bindings())
			Expect(b).To(HaveLen(1))
			Expect(b[0].Expr().Name).To(Equal("expr"))
		})
	})

	Describe("Before", func() {
		var (
			act *joeclifakes.FakeAction
			app *cli.App
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			app = &cli.App{
				Name: "app",
				Args: []*cli.Arg{
					{
						Name: "f",
						NArg: 0,
					},
					{
						Name: "expression",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "expr",
									Args: []*cli.Arg{
										{
											Name:  "f",
											Value: new(bool),
											NArg:  1,
										},
									},
									Before: act,
								},
							},
						},
					},
				},
				Action: func(c *cli.Context) {
					expr.FromContext(c, "expression").Evaluate(c, nil)
				},
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split("app arg -expr true")
			app.RunContext(context.Background(), args)
		})

		It("executes before", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("provides context Name", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Name()).To(Equal("<-expr>"))
		})

		It("provides context Path", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Path().String()).To(Equal("app <expression> <-expr>"))
		})
	})

	Describe("After", func() {
		var (
			act *joeclifakes.FakeAction
			app *cli.App
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			app = &cli.App{
				Name: "app",
				Args: []*cli.Arg{
					{
						Name: "f",
						NArg: 0,
					},
					{
						Name: "expression",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "expr",
									Args: []*cli.Arg{
										{
											Name:  "f",
											Value: new(bool),
											NArg:  1,
										},
									},
									After: act,
								},
							},
						},
					},
				},
				Action: func(c *cli.Context) {
					expr.FromContext(c, "expression").Evaluate(c, nil)
				},
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split("app arg -expr true")
			app.RunContext(context.Background(), args)
		})

		It("executes before", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("provides context Name", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Name()).To(Equal("<-expr>"))
		})

		It("provides context Path", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Path().String()).To(Equal("app <expression> <-expr>"))
		})
	})

	Describe("parsing", func() {
		DescribeTable(
			"examples",
			func(arguments string, match types.GomegaMatcher) {
				var captured bytes.Buffer
				evaluator := func(c *cli.Context, in any, yield func(any) error) error {
					fmt.Fprintf(c.Stdout, "-> %s=%v ", c.Name(), c.Values())
					return yield(in)
				}

				app := &cli.App{
					Name: "app",
					Action: func(c *cli.Context) {
						items := make([]any, 0)
						for _, v := range c.List("f") {
							items = append(items, v)
						}
						expr.FromContext(c, "expression").Evaluate(c, items...)
					},
					Stdout: &captured,
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: -2,
						},
						{
							Name: "expression",
							Value: &expr.Expression{
								Exprs: []*expr.Expr{
									{
										Name: "offset",
										Args: []*cli.Arg{
											{
												Name:  "value",
												Value: new(int),
												NArg:  1,
											},
										},
										Evaluate: evaluator,
									},
									{
										Name: "multi",
										Args: []*cli.Arg{
											{
												NArg: 1,
											},
											{
												NArg: 1,
											},
										},
										Evaluate: evaluator,
									},
								},
							},
						},
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(context.Background(), args)
				Expect(err).NotTo(HaveOccurred())

				Expect(captured.String()).To(match)

			},
			Entry(
				"end argument list",
				"arg -multi a b -offset 2",
				Equal(`-> <-multi>=[a b] -> <-offset>=[2] `),
			),
		)

		DescribeTable(
			"errors",
			func(arguments string, match types.GomegaMatcher) {
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: -2,
						},
						{
							Name: "e",
							Value: &expr.Expression{
								Exprs: []*expr.Expr{
									{
										Name: "expr",
									},
									{
										Name: "offset",
										Args: []*cli.Arg{
											{
												Name:  "value",
												Value: new(int),
												NArg:  1,
											},
										},
									},
								},
							},
						},
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(context.Background(), args)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(match))
			},
			Entry("args after expr", "arg -expr unbound", Equal(`arguments must precede expressions: "unbound"`)),
			Entry(
				"missing argument",
				"arg -offset",
				Equal(`expected argument`),
			),
		)
	})

	Describe("visibility", func() {

		It("disables implicitly hidden behavior of a expr via parent", func() {
			finder := strings.Fields("app parent <expr> <-_hidden>")

			var found *expr.Expr
			app := &cli.App{
				Commands: []*cli.Command{
					{
						Name:    "parent",
						Options: cli.DisableAutoVisibility,
						Args: cli.Args("expr", &expr.Expression{
							Exprs: []*expr.Expr{
								{Name: "_hidden"},
							},
						}),
					},
				},
				Action: func(c *cli.Context) {
					parent, _ := c.FindTarget(finder)
					found = parent.Target().(*expr.Expr)
				},
				Name: "app",
			}
			args, _ := cli.Split("app")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(expr.IsVisible(found)).To(BeTrue())
		})

		DescribeTable("examples", func(ex *expr.Expr, finder string, visibleExpected bool) {
			var found *expr.Expr
			app := &cli.App{
				Args: cli.Args(
					"expr",
					&expr.Expression{
						Exprs: []*expr.Expr{ex},
					},
				),
				Action: func(c *cli.Context) {
					parent, _ := c.FindTarget(strings.Fields("app <expr> " + finder))
					found = parent.Target().(*expr.Expr)
				},
				Name: "app",
			}
			args, _ := cli.Split("app")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(expr.IsVisible(found)).To(Equal(visibleExpected))
		},
			Entry("nominal", &expr.Expr{Name: "visible"}, "<-visible>", true),
			Entry("visible", &expr.Expr{Name: "visible", Options: cli.Visible}, "<-visible>", true),
			Entry("implicitly hidden by name", &expr.Expr{Name: "_hidden"}, "<-_hidden>", false),
			Entry("disable implicitly hidden behavior (self)", &expr.Expr{Name: "_hidden", Options: cli.DisableAutoVisibility}, "<-_hidden>", true),
			Entry("explicitly made visible implicitly hidden behavior", &expr.Expr{Name: "_hidden", Options: cli.Visible}, "<-_hidden>", true),
			Entry("hidden wins over visible", &expr.Expr{Name: "hidden", Options: cli.Visible | cli.Hidden}, "<-hidden>", false),
		)
	})

	Context("when evaluating", func() {

		var (
			arguments string
			captured  bytes.Buffer
		)

		BeforeEach(func() {
			arguments = "app x y z -expr a b c"
		})
		JustBeforeEach(func() {
			app := &cli.App{
				Action: func(c *cli.Context) {
					items := make([]any, 0)
					for _, v := range c.List("start") {
						items = append(items, v)
					}
					expr.FromContext(c, "e").Evaluate(c, items...)
				},
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
									Name: "expr",
									Args: cli.Args("a", cli.String(), "b", cli.String(), "c", cli.String()),
									Evaluate: func(c *cli.Context, in any, yield func(any) error) error {
										fmt.Fprintf(c.Stdout, "%s%s%s%s ", in, c.String("a"), c.String("b"), c.String("c"))
										return nil
									},
								},
							},
						},
					},
				},

				Stdout: &captured,
			}
			args, _ := cli.Split(arguments)
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
		})

		It("prints the expected output", func() {
			Expect(captured.String()).To(Equal("xabc yabc zabc "))
		})

	})

	Describe("Synopsis", func() {

		DescribeTable("examples",
			func(f *expr.Expr, expected string) {
				Expect(f.Synopsis()).To(Equal(expected))
			},
			Entry(
				"simple expr",
				&expr.Expr{
					Name: "expr",
					Args: cli.Args("item", cli.Bool()),
				},
				"-expr ITEM",
			),
			Entry(
				"expr from usage",
				&expr.Expr{
					Name:     "expr",
					HelpText: "Get it from the {PLACEHOLDER}",
					Args:     cli.Args("item", cli.Bool()),
				},
				"-expr PLACEHOLDER",
			),
			Entry(
				"repeat expr",
				&expr.Expr{
					Name: "expr",
					Args: []*cli.Arg{
						{
							Name: "file",
							NArg: -1,
						},
					},
				},
				"-expr FILE...",
			),
		)

	})

})
var _ = Describe("Predicate", func() {

	It("yields if true", func() {
		ev := expr.Predicate(func(v any) bool {
			return true
		})
		yield := new(exprfakes.FakeYielder)
		err := ev.Evaluate(&cli.Context{}, "input", yield.Spy)
		Expect(err).NotTo(HaveOccurred())
		Expect(yield.CallCount()).To(Equal(1))
		Expect(yield.ArgsForCall(0)).To(Equal("input"))
	})

	It("does not yield if false", func() {
		ev := expr.Predicate(func(v any) bool {
			return false
		})
		yield := new(exprfakes.FakeYielder)
		err := ev.Evaluate(&cli.Context{}, nil, yield.Spy)
		Expect(err).NotTo(HaveOccurred())
		Expect(yield.CallCount()).To(Equal(0))
	})
})

var _ = Describe("ComposeEvaluator", func() {

	It("handles all evaluators", func() {
		fake1 := new(exprfakes.FakeEvaluator)
		fake2 := new(exprfakes.FakeEvaluator)

		compose := expr.ComposeEvaluator(fake1, fake2)
		compose.Evaluate(context.Background(), nil, nil)
		Expect(fake1.EvaluateCallCount()).To(Equal(1))
		Expect(fake2.EvaluateCallCount()).To(Equal(1))
	})

	It("stops on first evaluator to yield", func() {
		fake := new(exprfakes.FakeYielder)
		value := new(struct{})
		yieldEvaluator := new(exprfakes.FakeEvaluator)
		yieldEvaluator.EvaluateStub = func(_ context.Context, v any, y func(any) error) error {
			Expect(v).To(BeIdenticalTo(value))
			return y(v)
		}
		blockedEvaluator := new(exprfakes.FakeEvaluator)

		compose := expr.ComposeEvaluator(yieldEvaluator, blockedEvaluator)
		compose.Evaluate(context.Background(), value, fake.Spy)
		Expect(yieldEvaluator.EvaluateCallCount()).To(Equal(1))
		Expect(blockedEvaluator.EvaluateCallCount()).To(Equal(0))
		Expect(fake.CallCount()).To(Equal(1))
	})

	It("stops on first truthful Predicate", func() {
		// This test mainly addresses the conjecture of the ComposeEvaluator documentation
		// which is that Predicates are useful to ComposeEvaluator!
		fake := new(exprfakes.FakeYielder)
		value := new(struct{})
		yieldEvaluator := expr.Predicate(func(any) bool {
			return true
		})
		blockedEvaluator := new(exprfakes.FakeEvaluator)

		compose := expr.ComposeEvaluator(yieldEvaluator, blockedEvaluator)
		compose.Evaluate(context.Background(), value, fake.Spy)
		Expect(blockedEvaluator.EvaluateCallCount()).To(Equal(0))
		Expect(fake.CallCount()).To(Equal(1))
	})

	It("does not propagate when evaluator returns error", func() {
		yieldsErrorEvaluator := new(exprfakes.FakeEvaluator)
		yieldsErrorEvaluator.EvaluateStub = func(context.Context, any, func(any) error) error {
			return errors.New("an error")
		}
		blockedEvaluator := new(exprfakes.FakeEvaluator)

		compose := expr.ComposeEvaluator(yieldsErrorEvaluator, blockedEvaluator)
		err := compose.Evaluate(context.Background(), nil, nil)
		Expect(yieldsErrorEvaluator.EvaluateCallCount()).To(Equal(1))
		Expect(blockedEvaluator.EvaluateCallCount()).To(Equal(0))
		Expect(err).To(MatchError("an error"))
	})
})

var _ = Describe("Error", func() {

	It("returns the specified error", func() {
		ev := expr.EvaluatorOf(errors.New("an error"))
		yield := new(exprfakes.FakeYielder)
		err := ev.Evaluate(&cli.Context{}, nil, yield.Spy)
		Expect(err).To(MatchError("an error"))
		Expect(yield.CallCount()).To(Equal(0))
		Expect(ev).To(BeAssignableToTypeOf(expr.Error(nil)))
	})

	It("generates an error from nil", func() {
		ev := expr.Error(nil)
		yield := new(exprfakes.FakeYielder)
		err := ev.Evaluate(&cli.Context{}, "input", yield.Spy)
		Expect(err).To(MatchError("unsupported value: string"))
	})
})

var _ = Describe("EvaluatorOf", func() {

	var called bool
	act := func() { called = true }

	DescribeTable("examples",
		func(thunk any) {
			var handler expr.Evaluator
			Expect(func() {
				handler = expr.EvaluatorOf(thunk)
			}).NotTo(Panic())

			called = false
			handler.Evaluate(&cli.Context{}, nil, new(exprfakes.FakeYielder).Spy)
			Expect(called).To(BeTrue())
		},
		Entry("func(*Context, any, func(any) error) error", func(*cli.Context, any, func(any) error) error { act(); return nil }),
		Entry("func(*Context, any) error", func(*cli.Context, any) error { act(); return nil }),
		Entry("func(*Context, any) bool", func(*cli.Context, any) bool { act(); return false }),
		Entry("func(*Context, any)", func(*cli.Context, any) { act() }),
		Entry("func(any, func(any) error) error", func(any, func(any) error) error { act(); return nil }),
		Entry("func(any) error", func(any) error { act(); return nil }),
		Entry("func(any) bool", func(any) bool { act(); return false }),
		Entry("func(any)", func(any) { act() }),
		Entry("func() bool", func() bool { act(); return false }),
		Entry("func() error", func() error { act(); return nil }),
	)

	It("always yields from boolean", func() {
		ev := expr.EvaluatorOf(true)
		yield := new(exprfakes.FakeYielder)
		err := ev.Evaluate(&cli.Context{}, nil, yield.Spy)
		Expect(err).NotTo(HaveOccurred())
		Expect(yield.CallCount()).To(Equal(1))
		Expect(ev).To(BeAssignableToTypeOf(expr.Invariant(false)))
	})
})
