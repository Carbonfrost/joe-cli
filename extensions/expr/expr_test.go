package expr_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/exprfakes"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		captured := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(captured.Value("e")).NotTo(BeNil())
	})

	It("does instance expressions", func() {
		var seen []int
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name: "more",
								Args: cli.Args("i", new(int)),
								Evaluate: func(c *cli.Context, _ any) {
									seen = append(seen, c.Values()[0].(int))
								},
							},
						},
					},
				},
			},
			Action: act,
		}
		args, _ := cli.Split("app -- -more 1 -more 2 -more 3")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		captured := cli.FromContext(act.ExecuteArgsForCall(0))
		expr.FromContext(captured, "expression").Evaluate(captured, 0)

		Expect(seen).To(Equal([]int{1, 2, 3}))
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
		err := app.RunContext(context.TODO(), args)
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
		err := app.RunContext(context.TODO(), args)
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
		err := app.RunContext(context.TODO(), args)
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
			err := app.RunContext(context.TODO(), args)
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

	It("marks the arg with expressions as initialized", func() {
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
		// args, _ := cli.Split("app -- -expr true")
		_, _ = app.Initialize(context.TODO())

		// FIXME Asserts
		// myArg := app.Args[0].Value.(*expr.Expression).Exprs[0].Args[0]
		// Expect(cli.IsInitialized(myArg)).To(BeTrue())
		// Expect(cli.IsDestinationImplicitlyCreated(myArg)).To(BeTrue())
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
		err := app.RunContext(context.TODO(), []string{"app"})
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Args[0].Name).To(Equal("expression"))
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
			app.RunContext(context.TODO(), args)
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
			Expect(cli.FromContext(captured).Values()).To(Equal([]interface{}{true, 2, []string{"a", "b", "c"}}))
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
			app.RunContext(context.TODO(), args)
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
			app.RunContext(context.TODO(), args)
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

	Describe("EvaluatorOf", func() {

		var called bool
		act := func() { called = true }

		DescribeTable("examples",
			func(thunk interface{}) {
				var handler expr.Evaluator
				Expect(func() {
					handler = expr.EvaluatorOf(thunk)
				}).NotTo(Panic())

				called = false
				handler.Evaluate(&cli.Context{}, nil, new(exprfakes.FakeYielder).Spy)
				Expect(called).To(BeTrue())
			},
			Entry("func(*Context, interface{}, func(interface{}) error) error", func(*cli.Context, interface{}, func(interface{}) error) error { act(); return nil }),
			Entry("func(*Context, interface{}) error", func(*cli.Context, interface{}) error { act(); return nil }),
			Entry("func(*Context, interface{}) bool", func(*cli.Context, interface{}) bool { act(); return false }),
			Entry("func(*Context, interface{})", func(*cli.Context, interface{}) { act() }),
			Entry("func(interface{}, func(interface{}) error) error", func(interface{}, func(interface{}) error) error { act(); return nil }),
			Entry("func(interface{}) error", func(interface{}) error { act(); return nil }),
			Entry("func(interface{}) bool", func(interface{}) bool { act(); return false }),
			Entry("func(interface{})", func(interface{}) { act() }),
		)

		It("always yields from boolean", func() {
			ev := expr.EvaluatorOf(true)
			yield := new(exprfakes.FakeYielder)
			err := ev.Evaluate(&cli.Context{}, nil, yield.Spy)
			Expect(err).NotTo(HaveOccurred())
			Expect(yield.CallCount()).To(Equal(1))
		})
	})

	Describe("parsing", func() {
		DescribeTable(
			"examples",
			func(arguments string, match types.GomegaMatcher) {
				var captured bytes.Buffer
				evaluator := func(c *cli.Context, in interface{}, yield func(interface{}) error) error {
					fmt.Fprintf(c.Stdout, "-> %s=%v ", c.Name(), c.Values())
					return yield(in)
				}

				app := &cli.App{
					Name: "app",
					Action: func(c *cli.Context) {
						items := make([]interface{}, 0)
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
				err := app.RunContext(context.TODO(), args)
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
				err := app.RunContext(context.TODO(), args)

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
			err := app.RunContext(context.TODO(), args)
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
			err := app.RunContext(context.TODO(), args)
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
					items := make([]interface{}, 0)
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
									Evaluate: func(c *cli.Context, in interface{}, yield func(interface{}) error) error {
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
			err := app.RunContext(context.TODO(), args)
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
