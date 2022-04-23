package cli_test

import (
	"bytes"
	"context"
	"os"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("usage", func() {

	Describe("parse", func() {

		DescribeTable("extract placeholders",
			func(text string, expected []string) {
				Expect(cli.ParseUsage(text).Placeholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", []string{}),
			Entry("placeholder", "{PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("placeholder used twice", "{PLACEHOLDER} {PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("2 placeholders", "{A} {B}", []string{"A", "B"}),
			Entry("2 placeholders with indexes", "{1:A} {0:B}", []string{"B", "A"}),
		)

		DescribeTable("without placeholders text",
			func(text string, expected string) {
				Expect(cli.ParseUsage(text).WithoutPlaceholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", "Literal text"),
			Entry("placeholder", "Load configuration from {FILE}s", "Load configuration from FILEs"),
		)
	})
})

var _ = Describe("Wrap", func() {
	DescribeTable("examples", func(width int, indent string, text string, expected string) {
		var buf bytes.Buffer
		cli.Wrap(&buf, text, indent, width)
		Expect(buf.String()).To(Equal(expected))
	},
		Entry("empty has trailing newline", 8, "", "", "\n"),
		Entry("no wraps", 80, "", "this text will not wrap", "this text will not wrap\n"),
		Entry("wraps", 8, "", "some text wraps", "some text\nwraps\n"),
		Entry("wraps with indent", 8, "  ", "some text wraps", "some text\n  wraps\n"),
		Entry("large indent trivializes width", 8, "    ", "some text wraps past", "some text\n    wraps\n    past\n"),
		Entry("leading spaces removed on next line", 10, "", "some  text   wraps", "some  text\nwraps\n"),
		Entry("retain user's leading spaces", 10, "", "    some text", "    some text\n"),
		Entry("retain user's leading spaces on wrapping", 10, "", "some  text\n   I indented", "some  text\n   I indented\n"),
		Entry("ANSI control codes don't get wrapped",
			3,
			"",
			"\x1B[38;2;249;38;114m(\x1B[0m\x1B[38;2;248;248;242mwell wishing well\x1B[38;2;249;38;114m)\x1B[0m",
			"\x1B[38;2;249;38;114m(\x1B[0m\x1B[38;2;248;248;242mwell\nwishing\nwell\x1B[38;2;249;38;114m)\x1B[0m\n",
		),
	)
})

var _ = Describe("RenderTemplate", func() {
	It("uses the template and custom funcs", func() {
		app := &cli.App{
			Name: "demo",
			Before: cli.Pipeline(
				cli.RegisterTemplateFunc("CustomFunc", func() string {
					return "customFunc result"
				}),
				cli.RegisterTemplate("custom", "template {{ CustomFunc }} {{ .Data }}"),
			),
			Action: cli.RenderTemplate("custom", func(_ *cli.Context) interface{} {
				return struct{ Data int }{1}
			}),
		}
		Expect(renderScreen(app, "app")).To(ContainSubstring("template customFunc result 1"))
	})
})

var _ = Describe("PrintVersion", func() {
	It("uses the version template", func() {
		app := &cli.App{
			Name:    "demo",
			Version: "hello.5.0",
			Before:  cli.RegisterTemplate("Version", "custom template {{ .App.Name }} {{ .App.Version }}"),
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderScreen(app, "app --version")).To(ContainSubstring("custom template demo hello.5.0"))
	})
})

var _ = Describe("SetColor", func() {
	It("sets whether color will be enabled", func() {
		app := &cli.App{
			Name: "demo",
			Action: cli.Pipeline(cli.SetColor(true), func(c *cli.Context) {
				c.Stdout.SetStyle(cli.Bold)
				c.Stdout.WriteString(" BOLD TEXT ")
				c.Stdout.Reset()
			}),
		}
		Expect(renderScreen(app, "demo")).To(
			Equal("\x1b[1m BOLD TEXT \x1b[0m"),
		)
	})

	It("sets whether color will be enabled nested", func() {
		app := &cli.App{
			Name: "demo",
			Uses: cli.SetColor(true),
			Commands: []*cli.Command{
				{
					Name: "sub",
					Action: func(c *cli.Context) {
						c.Stdout.SetStyle(cli.Bold)
						c.Stdout.WriteString(" BOLD TEXT ")
						c.Stdout.Reset()
					},
				},
			},
		}
		Expect(renderScreen(app, "demo sub")).To(
			Equal("\x1b[1m BOLD TEXT \x1b[0m"),
		)
	})
})

var _ = Describe("AutodetectColor", func() {
	It("will disable when TERM=dumb", func() {
		app := &cli.App{
			Name: "demo",
			Action: cli.Pipeline(cli.AutodetectColor(), func(c *cli.Context) {
				c.Stdout.SetStyle(cli.Bold)
				c.Stdout.WriteString("BOLD TEXT")
				c.Stdout.Reset()
			}),
		}
		Expect(renderScreen(app, "demo")).To(Equal("BOLD TEXT"))
	})
})

var _ = Describe("DisplayHelpScreen", func() {
	It("is the default action for an app with sub-commands", func() {
		app := &cli.App{
			Name: "demo",
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderScreen(app, "demo")).To(ContainSubstring("usage: demo"))
	})

	It("uses the help template", func() {
		app := &cli.App{
			Name:   "demo",
			Before: cli.RegisterTemplate("Help", "custom help template"),
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderScreen(app, "app help")).To(ContainSubstring("custom help template"))
	})

	DescribeTable("examples",
		func(app *cli.App, expected types.GomegaMatcher) {
			Expect(renderScreen(app, "app --help")).To(expected)
		},
		Entry("shows normal flags",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name: "normal",
					},
				},
			},
			ContainSubstring("--normal")),
		Entry("replace usage placeholders",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name:     "normal",
						HelpText: "Loads configuration from {FILE}s",
					},
				},
			},
			ContainSubstring("Loads configuration from FILEs")),
		Entry("does not show hidden flags",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "hidden",
						Options: cli.Hidden,
					},
				},
			},
			Not(ContainSubstring("--hidden"))),
		Entry("display action-like flags",
			&cli.App{},
			ContainSubstring("{--help | --version}")),
		Entry("display sub-command",
			&cli.App{
				Commands: []*cli.Command{
					{
						Name: "ok",
					},
				},
			},
			ContainSubstring("<command> [<args>]")),
		Entry("display description",
			&cli.App{
				Description: "description text",
			},
			ContainSubstring("description text")),
		Entry("display expression",
			&cli.App{
				Args: cli.Args("expression", &cli.Expression{}),
			},
			ContainSubstring("<expression>...")),
		Entry("display expression description",
			&cli.App{
				Args: cli.Args(
					"expr",
					&cli.Expression{
						Exprs: []*cli.Expr{
							{
								Name:     "cname",
								HelpText: "Gets the cname value",
							},
						},
					},
				),
			},
			And(
				ContainSubstring("Expressions:"),
				ContainSubstring("-cname"),
				ContainSubstring("Gets the cname value"),
			)),
		Entry("display arg description",
			&cli.App{
				Args: []*cli.Arg{
					{
						Name:        "e",
						Description: "e argument description",
					},
				},
			},
			ContainSubstring("e argument description")),
	)

	DescribeTable("sub-command examples",
		func(app *cli.App, args string, expected types.GomegaMatcher) {
			Expect(renderScreen(app, args)).To(expected)
		},
		Entry("shows sub-command using help switch",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			},
			"app --help sub",
			ContainSubstring("usage: app sub")),
		Entry("show sub-command using help command",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			},
			"app help sub",
			ContainSubstring("usage: app sub")),
		Entry("display expression",
			&cli.App{
				Commands: []*cli.Command{
					{
						Name: "sub",
						Args: []*cli.Arg{
							{
								Value: &cli.Expression{
									Exprs: []*cli.Expr{
										{
											Name: "expr",
										},
									},
								},
							},
						},
					},
				},
			},
			"app help sub",
			ContainSubstring("<expression>...")),
	)

})

func disableConsoleColor() func() {
	os.Setenv("TERM", "dumb")
	return func() {
		os.Setenv("TERM", "0")
	}
}

func renderScreen(app *cli.App, args string) string {
	defer disableConsoleColor()()

	arguments, _ := cli.Split(args)
	var buffer bytes.Buffer
	app.Stderr = &buffer
	app.Stdout = &buffer
	_ = app.RunContext(context.TODO(), arguments)
	return buffer.String()
}
