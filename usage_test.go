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

var _ = Describe("ExecuteTemplate", func() {
	It("uses the template and custom funcs", func() {
		app := &cli.App{
			Name: "demo",
			Before: cli.Pipeline(
				cli.RegisterTemplateFunc("CustomFunc", func() string {
					return "customFunc result"
				}),
				cli.RegisterTemplate("custom", "template {{ CustomFunc }} {{ .Data }}"),
			),
			Action: cli.ExecuteTemplate("custom", func(_ *cli.Context) any {
				return struct{ Data int }{1}
			}),
		}
		Expect(renderScreen(app, "app")).To(ContainSubstring("template customFunc result 1"))
	})

	It("is error when not registered", func() {
		app := &cli.App{
			Name: "demo",
			Action: cli.ExecuteTemplate("custom", func(_ *cli.Context) any {
				return nil
			}),
		}
		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).To(MatchError(ContainSubstring(`template does not exist: "custom"`)))
	})
})

var _ = Describe("Template", func() {
	It("is nil when not registered", func() {
		tpl := &cli.Template{}
		app := &cli.App{
			Name: "demo",
			Action: func(c *cli.Context) {
				tpl = c.Template("missing")
			},
		}
		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(tpl).To(BeNil())
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

var _ = Describe("NewBuffer", func() {
	It("sets whether color will be enabled", func() {
		var actual string
		app := &cli.App{
			Name: "demo",
			Action: cli.Pipeline(cli.SetColor(true), func(c *cli.Context) {
				buf := c.NewBuffer()
				buf.SetStyle(cli.Bold)
				buf.WriteString(" BOLD TEXT ")
				buf.Reset()
				actual = buf.String()
			}),
		}
		renderScreen(app, "demo")
		Expect(actual).To(Equal("\x1b[1m BOLD TEXT \x1b[0m"))
	})

	It("sets whether color will be enabled nested", func() {
		var actual string
		app := &cli.App{
			Name: "demo",
			Uses: cli.SetColor(true),
			Commands: []*cli.Command{
				{
					Name: "sub",
					Action: func(c *cli.Context) {
						buf := c.NewBuffer()
						buf.SetStyle(cli.Bold)
						buf.WriteString(" BOLD TEXT ")
						buf.Reset()
						actual = buf.String()
					},
				},
			},
		}
		renderScreen(app, "demo sub")
		Expect(actual).To(Equal("\x1b[1m BOLD TEXT \x1b[0m"))
	})

	It("will be set when TERM=dumb", func() {
		var actual string

		app := &cli.App{
			Name: "demo",
			Action: cli.Pipeline(cli.AutodetectColor(), func(c *cli.Context) {
				buf := c.NewBuffer()
				buf.SetStyle(cli.Bold)
				buf.WriteString("BOLD TEXT")
				buf.Reset()
				actual = buf.String()
			}),
		}
		renderScreen(app, "demo")
		Expect(actual).To(Equal("BOLD TEXT"))
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
		Entry("display expression description (last-minute addition)",
			&cli.App{
				Args: cli.Args(
					"expr",
					&cli.Expression{
						Exprs: []*cli.Expr{},
					},
				),
				Before: func(c *cli.Context) error {
					arg, _ := c.LookupArg("expr")
					exprs := arg.Value.(*cli.Expression).Exprs
					arg.Value.(*cli.Expression).Exprs = append(exprs, &cli.Expr{Name: "cname", HelpText: "Gets the cname value"})
					return nil
				},
			},
			And(
				ContainSubstring("Expressions:"),
				ContainSubstring("-cname"),
				ContainSubstring("Gets the cname value"),
			)),

		Entry("hide expr",
			&cli.App{
				Args: cli.Args(
					"expr",
					&cli.Expression{
						Exprs: []*cli.Expr{
							{Name: "hidden", Options: cli.Hidden},
							{Name: "visible"},
						},
					},
				),
			},
			And(
				ContainSubstring("Expressions:"),
				ContainSubstring("-visible"),
				Not(ContainSubstring("-hidden")),
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

		Entry("display sorted flags",
			&cli.App{
				Options: cli.SortedFlags,
				Flags: []*cli.Flag{
					{Name: "zoo"},
					{Name: "due"},
				},
			},
			MatchRegexp(`(?s)--due.+--zoo`)),

		Entry("display sorted commands",
			&cli.App{
				Options: cli.SortedCommands,
				Commands: []*cli.Command{
					{Name: "z,"},
					{Name: "d,"},
				},
			},
			MatchRegexp(`(?s)d,.*z,`)),

		Entry("display sorted exprs",
			&cli.App{
				Args: []*cli.Arg{
					{
						Options: cli.SortedExprs,
						Value: &cli.Expression{
							Exprs: []*cli.Expr{
								{Name: "z,"},
								{Name: "d,"},
							},
						},
					},
				},
			},
			MatchRegexp(`(?s)-d,.*-z,`)),

		Entry("custom help part",
			&cli.App{
				Flags: []*cli.Flag{
					{Name: "z"},
				},

				// Must be done in Before so as to be done after built-in templates
				Before: cli.RegisterTemplate("Flag", `my custom synopsis`),
			},
			ContainSubstring("my custom synopsis")),
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
