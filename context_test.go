package cli_test

import (
	"context"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context", func() {

	Describe("Value", func() {
		It("contains flag value at the app level", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app -f")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal(true))
		})

		It("contains flag value from inherited context", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.String(),
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Action: act,
					},
				},
			}

			args, _ := cli.Split("app -f dom sub")
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal("dom"))
		})

		It("contains flag value set using one of its aliases", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "f",
						Aliases: []string{"alias"},
						Value:   cli.Bool(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app --alias")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal(true))
		})

		It("contains arg value", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name:  "f",
						Value: cli.List(),
						NArg:  -1,
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app s r o")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal([]string{"s", "r", "o"}))
		})
	})

	Describe("Raw", func() {
		It("contains flag value at the app level", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app -f")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Raw("f")).To(Equal([]string{"-f", ""}))
		})

		It("contains flag value from inherited context", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.String(),
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Action: act,
					},
				},
			}

			args, _ := cli.Split("app -f dom sub")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Raw("f")).To(Equal([]string{"-f", "dom"}))
		})

		It("contains flag value from self context", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:   "f",
						Value:  cli.String(),
						Action: act,
					},
				},
			}

			args, _ := cli.Split("app -f sub")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Raw("")).To(Equal([]string{"-f", "sub"}))
		})

		DescribeTable("examples",
			func(flag *cli.Flag, arguments string, raw, rawOccurrences []string) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Flags: []*cli.Flag{
						flag,
					},
					Action: act,
				}

				args, _ := cli.Split(arguments)
				_ = app.RunContext(context.TODO(), args)

				capturedContext := act.ExecuteArgsForCall(0)
				Expect(capturedContext.Raw("f")).To(Equal(raw))
				Expect(capturedContext.RawOccurrences("f")).To(Equal(rawOccurrences))
			},
			Entry(
				"bool flags",
				&cli.Flag{Name: "f", Value: cli.Bool()},
				"app -f",
				[]string{"-f", ""},
				[]string{""},
			),
			Entry(
				"multiple bool calls",
				&cli.Flag{Name: "f", Value: cli.Bool()},
				"app -f -f -f",
				[]string{"-f", "", "-f", "", "-f", ""},
				[]string{"", "", ""},
			),
			Entry(
				"string with quotes",
				&cli.Flag{Name: "f", Value: cli.String()},
				`app -f "text has spaces" -f ""`,
				[]string{"-f", "text has spaces", "-f", ""},
				[]string{"text has spaces", ""},
			),
			Entry(
				"name-value arg counter semantics",
				&cli.Flag{Name: "f", Value: new(cli.NameValue)},
				`app -f hello space`,
				[]string{"-f", "hello", "space"},
				[]string{"hello", "space"},
			),
			Entry(
				"name-value arg counter semantics (long flag)",
				&cli.Flag{Name: "f", Value: new(cli.NameValue)},
				`app --f hello space`,
				[]string{"--f", "hello", "space"},
				[]string{"hello", "space"},
			),
			Entry(
				"alias flags",
				&cli.Flag{
					Name:    "f",
					Aliases: []string{"alias"},
					Value:   cli.Bool(),
				},
				"app --alias",
				[]string{"--alias", ""},
				[]string{""},
			),
			Entry(
				"long with equals",
				&cli.Flag{
					Name:    "f",
					Aliases: []string{"alias"},
					Value:   cli.Duration(),
				},
				"app --alias=9m32s",
				[]string{"--alias", "9m32s"},
				[]string{"9m32s"},
			),
		)

		It("contains arg value", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name:  "f",
						Value: cli.List(),
						NArg:  -1,
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app s r o")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Raw("f")).To(Equal([]string{"s", "r", "o"}))
		})
	})

	Describe("Before", func() {

		It("defers when set from initializer", func() {
			act := new(joeclifakes.FakeAction)
			act.ExecuteCalls(func(c *cli.Context) error {
				Expect(c.IsBefore()).To(BeTrue())
				return nil
			})
			app := &cli.App{
				Uses: func(c *cli.Context) {
					c.Before(act)

					Expect(act.ExecuteCallCount()).To(Equal(0))
				},
			}

			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("invokes immediately in the before context", func() {
			ctx := &cli.Context{}
			act := new(joeclifakes.FakeAction)
			cli.SetBeforeTiming(ctx)

			_ = ctx.Before(act)
			Expect(act.ExecuteCallCount()).To(Equal(1))
			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.IsBefore()).To(BeTrue())
		})

		DescribeTable("error when timing after",
			func(timing func(*cli.Context)) {
				act := new(joeclifakes.FakeAction)
				ctx := &cli.Context{}
				timing(ctx)

				err := ctx.Before(act)
				Expect(err).To(HaveOccurred())
			},
			Entry("action timing", cli.SetActionTiming),
			Entry("after timing", cli.SetAfterTiming),
		)
	})

	Describe("Walk", func() {

		var (
			walker   func(*cli.Context) error
			app      *cli.App
			paths    []string
			commands []string

			walkHelper = func(cmd *cli.Context) {
				// Don't worry about "help" and "version" in this test
				if cmd.Name() == "help" || cmd.Name() == "version" {
					return
				}

				commands = append(commands, cmd.Name())
				paths = append(paths, cmd.Path().String())
			}
		)

		BeforeEach(func() {
			walker = func(cmd *cli.Context) error {
				walkHelper(cmd)
				return nil
			}
		})

		JustBeforeEach(func() {
			commands = make([]string, 0, 5)
			paths = make([]string, 0, 5)
			app = &cli.App{
				Name: "_",
				Action: func(c *cli.Context) {
					c.Walk(walker)
				},
				Commands: []*cli.Command{
					{
						Name: "p",
						Subcommands: []*cli.Command{
							{
								Name: "c",
								Subcommands: []*cli.Command{
									{
										Name: "g",
									},
									{
										Name: "h",
									},
								},
							},
						},
					},
					{
						Name: "q",
					},
				},
			}

			_ = app.RunContext(context.TODO(), nil)
		})

		It("provides the expected traversal", func() {
			Expect(commands).To(Equal([]string{
				"_",
				"p",
				"c",
				"g",
				"h",
				"q",
			}))
			Expect(paths).To(Equal([]string{
				"_",
				"_ p",
				"_ p c",
				"_ p c g",
				"_ p c h",
				"_ q",
			}))

		})

		Context("when SkipCommand", func() {

			BeforeEach(func() {
				walker = func(cmd *cli.Context) error {
					walkHelper(cmd)
					if cmd.Name() == "c" {
						return cli.SkipCommand
					}
					return nil
				}
			})

			It("do skip sub-commands", func() {
				Expect(commands).To(Equal([]string{
					"_",
					"p",
					"c",
					"q",
				}))
			})
		})

	})

	Describe("FindTarget", func() {

		DescribeTable("examples",
			func(name []string, id string) {
				var actualID interface{}
				app := &cli.App{
					Action: func(c *cli.Context) {
						if actual, ok := c.FindTarget(cli.ContextPath(name)); ok {
							switch a := actual.Target().(type) {
							case *cli.Flag:
								actualID = a.Data["id"]
							case *cli.Arg:
								actualID = a.Data["id"]
							case *cli.Command:
								actualID = a.Data["id"]
							}
						}
					},
					Commands: []*cli.Command{
						{
							Name: "sub",
							Args: []*cli.Arg{
								{
									Name: "arg",
									Uses: cli.Data("id", "2"),
								},
								{
									Name: "arg2",
									Uses: cli.Data("id", "3"),
								},
							},
							Uses: cli.Data("id", "1"),
						},
					},
					Flags: []*cli.Flag{
						{
							Name:  "f",
							Value: new(bool),
						},
					},
					Uses: cli.Data("id", "0"),
					Name: "app",
				}

				_ = app.RunContext(context.TODO(), []string{"app"})
				Expect(actualID).To(Equal(id))
			},
			Entry("empty self", []string{}, "0"),
			Entry("app name", []string{"app"}, "0"),
			Entry("sub-command", []string{"app", "sub"}, "1"),
			Entry("arg", []string{"app", "sub", "<arg>"}, "2"),
		)
	})
})

var _ = Describe("ContextPath", func() {

	DescribeTable("Match",
		func(pattern string, path string) {
			p := cli.ContextPath(strings.Fields(path))
			Expect(p.Match(pattern)).To(BeTrue())
		},
		Entry("simple", "app", "app"),
		Entry("simple command", "sub", "app sub"),
		Entry("nested command", "sub", "app app sub"),
		Entry("simple flag", "--flag", "app --flag"),
		Entry("nested flag", "--flag", "app app sub --flag"),
		Entry("flag one dash", "-flag", "app --flag"),
		Entry("expression", "<-expr>", "app <expr> <-expr>"),
		Entry("any command", "*", "app"),
		Entry("any sub-command", "*", "app sub"),
		Entry("any flag", "-", "app --flag"),
		Entry("any arg", "<>", "app <arg>"),
		Entry("any expr", "<->", "app <-expr>"),
		Entry("sub path", "sub cmd", "app sub cmd"),
	)

	DescribeTable("Match counterexamples",
		func(pattern string, path string) {
			p := cli.ContextPath(strings.Fields(path))
			Expect(p.Match(pattern)).To(BeFalse())
		},
		Entry("* doesn't match flag", "*", "app --flag"),
		Entry("* doesn't match arg", "*", "app <arg>"),
		Entry("* doesn't match expr", "*", "app <-expr>"),
		Entry("<> doesn't match expr", "*", "app <-expr>"),
		Entry("flag doesn't match sub-command", "-", "app sub"),
	)
})
