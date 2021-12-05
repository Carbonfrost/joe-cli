package cli_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
			},
			Exprs: []*cli.Expr{
				{
					Name: "expr",
					Args: cli.Args("a", cli.Bool()),
				},
			},
		}
		args, _ := cli.Split("app x -expr true")
		app.RunContext(context.TODO(), args)

		captured := act.ExecuteArgsForCall(0)
		Expect(captured.Expression())

	})

	Describe("Evaluate", func() {
		var (
			act *joeclifakes.FakeEvaluator
			app *cli.App
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeEvaluator)
			app = &cli.App{
				Name: "app",
				Args: []*cli.Arg{
					{
						Name: "f",
						NArg: 0,
					},
				},
				Action: func(c *cli.Context) {
					c.Expression().Evaluate(c, "items")
				},
				Exprs: []*cli.Expr{
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
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split("app arg -expr true 1 2 a b c")
			app.RunContext(context.TODO(), args)
		})

		It("executes action on setting Arg", func() {
			Expect(act.EvaluateCallCount()).To(Equal(1))
		})

		XIt("contains args in captured context", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(captured.Args()).To(Equal([]string{"true", "1", "2", "a", "b", "c"}))
		})

		It("contains values in captured context", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(captured.Values()).To(Equal([]interface{}{true, 2, []string{"a", "b", "c"}}))
		})

		It("provides context Name", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(captured.Name()).To(Equal("<-expr>"))
		})

		It("provides context Path", func() {
			captured, _, _ := act.EvaluateArgsForCall(0)
			Expect(captured.Path().String()).To(Equal("app <-expr>"))
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
				},
				Exprs: []*cli.Expr{
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
				Action: func(c *cli.Context) {
					c.Expression().Evaluate(c, nil)
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
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Name()).To(Equal("<-expr>"))
		})

		It("provides context Path", func() {
			captured := act.ExecuteArgsForCall(0)
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
				},
				Exprs: []*cli.Expr{
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
				Action: func(c *cli.Context) {
					c.Expression().Evaluate(c, nil)
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
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Name()).To(Equal("<-expr>"))
		})

		It("provides context Path", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Path().String()).To(Equal("app <expression> <-expr>"))
		})
	})

	Describe("EvaluatorOf", func() {

		var called bool
		act := func() { called = true }

		DescribeTable("examples",
			func(thunk interface{}) {
				var handler cli.Evaluator
				Expect(func() {
					handler = cli.EvaluatorOf(thunk)
				}).NotTo(Panic())

				called = false
				handler.Evaluate(&cli.Context{}, nil, new(joeclifakes.FakeYielder).Spy)
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
			ev := cli.EvaluatorOf(true)
			yield := new(joeclifakes.FakeYielder)
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
						c.Expression().Evaluate(c, items...)
					},
					Stdout: &captured,
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: -2,
						},
					},
					Exprs: []*cli.Expr{
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
					},
					Exprs: []*cli.Expr{
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
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(context.TODO(), args)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(match)
			},
			Entry("args after expr", "arg -expr unbound", Equal(`arguments must precede expressions: "unbound"`)),
			Entry(
				"missing argument",
				"arg -offset",
				Equal(`expected argument`),
			),
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
					c.Expression().Evaluate(c, items...)
				},
				Args: []*cli.Arg{
					{
						Name: "start",
						NArg: -2,
					},
				},
				Exprs: []*cli.Expr{
					{
						Name: "expr",
						Args: cli.Args("a", cli.String(), "b", cli.String(), "c", cli.String()),
						Evaluate: func(c *cli.Context, in interface{}, yield func(interface{}) error) error {
							fmt.Fprintf(c.Stdout, "%s%s%s%s ", in, c.String("a"), c.String("b"), c.String("c"))
							return nil
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
			func(f *cli.Expr, expected string) {
				Expect(f.Synopsis()).To(Equal(expected))
			},
			Entry(
				"simple expr",
				&cli.Expr{
					Name: "expr",
					Args: cli.Args("item", cli.Bool()),
				},
				"-expr ITEM",
			),
			Entry(
				"expr from usage",
				&cli.Expr{
					Name:     "expr",
					HelpText: "Get it from the {PLACEHOLDER}",
					Args:     cli.Args("item", cli.Bool()),
				},
				"-expr PLACEHOLDER",
			),
			Entry(
				"repeat expr",
				&cli.Expr{
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
