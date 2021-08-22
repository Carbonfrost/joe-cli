package cli_test

import (
	"bytes"
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
		act := new(joeclifakes.FakeActionHandler)
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
		app.RunContext(nil, args)

		captured := act.ExecuteArgsForCall(0)
		Expect(captured.Expression())

	})

	Describe("parsing", func() {
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
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(nil, args)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(match)
			},
			Entry("args after expr", "arg -expr unbound", Equal(`arguments must precede expressions: "unbound"`)),
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
			err := app.RunContext(nil, args)
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
