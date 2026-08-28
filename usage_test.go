// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
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

	var example = struct {
		App struct{ Name string }
	}{
		App: struct{ Name string }{
			Name: "li",
		},
	}

	DescribeTable("examples", func(data any) {
		var captured bytes.Buffer
		app := &cli.App{
			Name:   "li",
			Uses:   cli.RegisterTemplate("T", "{{ .App.Name }}"),
			Action: cli.ExecuteTemplate("T", data),
			Stdout: &captured,
		}

		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).NotTo(HaveOccurred())
		Expect(captured.String()).To(Equal("li"))
	},
		Entry("nil to default", nil),
		Entry("thunk", func() any { return example }),
		Entry("context.Context thunk", func(context.Context) any { return example }),
		Entry("*Context thunk", func(*cli.Context) any { return example }),
	)

	It("panics on unknown func signature", func() {
		Expect(func() {
			new(cli.Context).ExecuteTemplate("_", func() int { return 0 })
		}).To(Panic())
	})

	DescribeTable("errors", func(data any) {
		app := &cli.App{
			Name:   "app",
			Uses:   cli.RegisterTemplate("T", "{{ .App.Name }}"),
			Action: cli.ExecuteTemplate("T", data),
			Stdout: io.Discard,
		}
		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(`execution error (template "T"): error returned`))
	},
		Entry("context.Context err thunk", func(context.Context) (any, error) { return nil, fmt.Errorf("error returned") }),
		Entry("Context err thunk", func(*cli.Context) (any, error) { return nil, fmt.Errorf("error returned") }),
		Entry("err thunk", func() (any, error) { return nil, fmt.Errorf("error returned") }),
	)

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
				buf := c.Stdout
				buf.SetStyle(cli.Bold)
				buf.WriteString(" BOLD TEXT ")
				buf.Reset()
			}),
		}
		actual = renderScreen(app, "demo")
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
						buf := c.Stdout
						buf.SetStyle(cli.Bold)
						buf.WriteString(" BOLD TEXT ")
						buf.Reset()
					},
				},
			},
		}
		actual = renderScreen(app, "demo sub")
		Expect(actual).To(Equal("\x1b[1m BOLD TEXT \x1b[0m"))
	})

	It("will be set when TERM=dumb", func() {
		var actual string

		app := &cli.App{
			Name: "demo",
			Action: cli.Pipeline(cli.AutodetectColor(), func(c *cli.Context) {
				buf := c.Stdout
				buf.SetStyle(cli.Bold)
				buf.WriteString("BOLD TEXT")
				buf.Reset()
			}),
		}
		actual = renderScreen(app, "demo")
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

	It("returns exit code 2", func() {
		app := &cli.App{
			Name:   "demo",
			Action: cli.DisplayHelpScreen(),
			Stderr: io.Discard,
		}
		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).To(beExitCode(2))
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
		Entry("display default text",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name:        "s",
						DefaultText: "easy",
					},
				},
			},
			ContainSubstring("(default: easy)")),
		Entry("display expression",
			&cli.App{
				Args: cli.Args("expression", &expr.Expression{}),
			},
			ContainSubstring("<expression>...")),
		Entry("display expression description",
			&cli.App{
				Args: cli.Args(
					"expr",
					&expr.Expression{
						Exprs: []*expr.Expr{
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
		Entry("display expression description with binding",
			&cli.App{
				Args: cli.Args(
					"expr",
					&expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name:     "cname",
								HelpText: "Gets the cname value",
								// Addresses a bug in Prototype{} where HelpText was getting
								// overwritten by the effect of the Prototype being used in expr.Evaluator's
								// initializer
								Uses: expr.BindEvaluator(func(int) expr.Evaluator { return nil }, bind.Int()),
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
					&expr.Expression{
						Exprs: []*expr.Expr{},
					},
				),
				Before: func(c *cli.Context) error {
					arg, _ := c.LookupArg("expr")
					exprs := arg.Value.(*expr.Expression).Exprs
					arg.Value.(*expr.Expression).Exprs = append(exprs, &expr.Expr{Name: "cname", HelpText: "Gets the cname value"})
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
					&expr.Expression{
						Exprs: []*expr.Expr{
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
					{Name: "z_"},
					{Name: "d_"},
				},
			},
			MatchRegexp(`(?s)d_.*z_`)),

		Entry("display sorted exprs",
			&cli.App{
				Args: []*cli.Arg{
					{
						Options: cli.SortedExprs,
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{Name: "z"},
								{Name: "d"},
							},
						},
					},
				},
			},
			MatchRegexp(`(?s)-d.*-z`)),

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

	DescribeTable("arg examples",
		func(arg *cli.Arg, expected types.GomegaMatcher) {
			app := &cli.App{
				Args: []*cli.Arg{
					arg,
				},
			}
			Expect(renderScreen(app, "app --help")).To(expected)
		},
		Entry("optional arg",
			&cli.Arg{
				UsageText: "usage",
				NArg:      cli.OptionalArg(regexp.MustCompile("-.+").MatchString)},
			ContainSubstring("[usage]"),
		),
	)

	DescribeTable("sub-command examples",
		func(app *cli.App, args string, expected types.GomegaMatcher) {
			Expect(renderScreen(app, args)).To(expected)
		},
		Entry("shows sub-command using help switch interspersed",
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
								Value: &expr.Expression{
									Exprs: []*expr.Expr{
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
		Entry("shows sub-command using help switch after it",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			},
			"app sub --help",
			ContainSubstring("usage: app sub")),
		Entry("shows sub-command with sub-commands using help switch",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
						Subcommands: []*cli.Command{
							{Name: "bar"},
							{Name: "baz"},
						},
					},
				},
			},
			"app sub --help",
			ContainSubstring("usage: app sub")),
		Entry("does not show hidden flags on sub-command",
			&cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:     "global",
						HelpText: "hidden persistent flag",
						Options:  cli.Hidden | cli.Exits,
					},
				},
				Commands: []*cli.Command{
					{
						Name: "sub",
						Flags: []*cli.Flag{
							{
								Name:     "h",
								HelpText: "hidden flag",
								Options:  cli.Hidden | cli.Exits,
							},
							{
								Name:     "v",
								HelpText: "visible flag",
							},
						},
					},
				},
			},
			"app sub --help",
			And(
				Not(ContainSubstring("hidden flag")),
				Not(ContainSubstring("hidden persistent flag")),
				ContainSubstring("visible flag"),
			)),
		Entry("does not show a persistent flag that the sub-command can't use",
			persistentInApp(),
			"app other --help",
			Not(ContainSubstring("narrowed persistent flag"))),
		Entry("shows a persistent flag that the sub-command can use",
			persistentInApp(),
			"app sub --help",
			ContainSubstring("narrowed persistent flag")),
	)

})

var _ = Describe("HelpTopic", func() {

	DescribeTable("registration examples",
		func(app *cli.App) {
			Expect(renderScreen(app, "app --help")).To(And(
				ContainSubstring("Additional help topics:"),
				ContainSubstring("environment"),
				ContainSubstring("Environment variables"),
			))
		},
		Entry("from the app Uses pipeline",
			&cli.App{
				Name: "app",
				Uses: environmentTopic(),
			}),
		Entry("from a sub-command Uses pipeline",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
						Uses: environmentTopic(),
					},
				},
			}),
		Entry("from a flag Uses pipeline",
			&cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name: "f",
						Uses: environmentTopic(),
					},
				},
			}),
		Entry("from an arg Uses pipeline",
			&cli.App{
				Name: "app",
				Args: []*cli.Arg{
					{
						Name: "a",
						Uses: environmentTopic(),
					},
				},
			}),
	)

	It("does not display the heading when there are no topics", func() {
		app := &cli.App{
			Name: "app",
		}
		Expect(renderScreen(app, "app --help")).NotTo(ContainSubstring("Additional help topics:"))
	})

	It("is not displayed on the help screen of a sub-command", func() {
		app := &cli.App{
			Name: "app",
			Uses: environmentTopic(),
			Commands: []*cli.Command{
				{Name: "sub"},
			},
		}
		Expect(renderScreen(app, "app help sub")).NotTo(ContainSubstring("Additional help topics:"))
	})

	It("replaces a topic which was registered with the same name", func() {
		app := &cli.App{
			Name: "app",
			Uses: cli.Pipeline(
				environmentTopic(),
				cli.HelpTopic{
					Name:        "environment",
					HelpText:    "Newer help text",
					Description: "Newer description",
				},
			),
		}
		Expect(renderScreen(app, "app --help")).To(And(
			ContainSubstring("Newer help text"),
			Not(ContainSubstring("Environment variables that are used")),
		))
	})

	Describe("help sub-command", func() {

		It("displays the contents of the topic", func() {
			app := &cli.App{
				Name:     "app",
				Uses:     environmentTopic(),
				Commands: []*cli.Command{{Name: "sub"}},
			}
			Expect(renderScreen(app, "app help environment")).To(
				Equal("PAGER is used to page output\n\n"))
		})

		It("uses the HelpTopic template", func() {
			app := &cli.App{
				Name:     "app",
				Uses:     environmentTopic(),
				Commands: []*cli.Command{{Name: "sub"}},

				// Must be done in Before so as to be ready after built-in templates
				Before: cli.RegisterTemplate("HelpTopic", "custom topic template {{ .Name }}"),
			}
			Expect(renderScreen(app, "app help environment")).To(
				ContainSubstring("custom topic template environment"))
		})

		It("prefers a sub-command which has the same name", func() {
			app := &cli.App{
				Name: "app",
				Uses: cli.HelpTopic{
					Name:        "sub",
					Description: "topic description",
				},
				Commands: []*cli.Command{
					{Name: "sub", HelpText: "the sub-command"},
				},
			}
			Expect(renderScreen(app, "app help sub")).To(And(
				ContainSubstring("usage: app sub"),
				Not(ContainSubstring("topic description")),
			))
		})

		It("is an error when the topic doesn't exist", func() {
			app := &cli.App{
				Name:     "app",
				Uses:     environmentTopic(),
				Commands: []*cli.Command{{Name: "sub"}},
				Stderr:   io.Discard,
			}
			err := app.RunContext(context.Background(), []string{"app", "help", "unknown"})
			Expect(err).To(MatchError(ContainSubstring(`"unknown" is not a command`)))
		})
	})

	Describe("help flag", func() {
		It("does not display the topic", func() {
			app := &cli.App{
				Name:   "app",
				Uses:   environmentTopic(),
				Stderr: io.Discard,
				Stdout: io.Discard,
			}
			Expect(renderScreen(app, "app --help environment")).NotTo(
				ContainSubstring("PAGER is used to page output"))
		})
	})
})

var _ = Describe("ListHelpTopics", func() {

	DescribeTable("examples",
		func(action func() cli.Action, expected types.GomegaMatcher) {
			app := &cli.App{
				Name: "app",
				Uses: cli.Pipeline(
					environmentTopic(),
					cli.HelpTopic{
						Name:     "tutorial",
						HelpText: "A gentle introduction",
					},
				),
				Action: action(),
			}
			Expect(renderScreen(app, "app")).To(expected)
		},
		Entry("lists each topic", cli.ListHelpTopics, And(
			ContainSubstring("environment"),
			ContainSubstring("Environment variables that are used"),
			ContainSubstring("tutorial"),
			ContainSubstring("A gentle introduction"),
		)),
		Entry("does not display the heading", cli.ListHelpTopics,
			Not(ContainSubstring("Additional help topics:"))),
	)

	It("uses the HelpTopicListing template", func() {
		app := &cli.App{
			Name: "app",
			Uses: environmentTopic(),

			// Must be done in Before so as to be done after built-in templates
			Before: cli.RegisterTemplate("HelpTopicListing", `{{ range . }}custom listing {{ .Name }}{{ end }}`),
			Action: cli.ListHelpTopics(),
		}
		Expect(renderScreen(app, "app")).To(ContainSubstring("custom listing environment"))
	})

	It("can be used within a Uses pipeline", func() {
		app := &cli.App{
			Name: "app",
			Uses: cli.Pipeline(
				environmentTopic(),
				cli.ListHelpTopics(),
			),
		}
		Expect(renderScreen(app, "app")).To(ContainSubstring("Environment variables that are used"))
	})
})

func environmentTopic() cli.HelpTopic {
	return cli.HelpTopic{
		Name:        "environment",
		HelpText:    "Environment variables that are used",
		Description: "PAGER is used to page output",
	}
}

// persistentInApp defines --global at the root, but narrowed to only apply to sub
func persistentInApp() *cli.App {
	return &cli.App{
		Name: "app",
		Flags: []*cli.Flag{
			{
				Name:     "global",
				Value:    cli.Bool(),
				HelpText: "narrowed persistent flag",
				Uses:     cli.PersistentIn(cli.PatternFilter("app sub")),
			},
		},
		Commands: []*cli.Command{
			{Name: "sub"},
			{Name: "other"},
		},
	}
}

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
	_ = app.RunContext(context.Background(), arguments)
	return buffer.String()
}
