package cli_test

import (
	"context"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

				capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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
				[]string{"-f", "hello", "space"},
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

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.Raw("f")).To(Equal([]string{"<f>", "s", "<f>", "r", "<f>", "o"}))
			Expect(capturedContext.RawOccurrences("f")).To(Equal([]string{"s", "r", "o"}))
		})
	})

	Describe("SetValue", func() {
		DescribeTable("examples", func(value any, v any) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "flag",
						Aliases: []string{"f"},
						Value:   value,
						Before: func(c *cli.Context) {
							Expect(func() { c.SetValue(v) }).NotTo(Panic())
						},
					},
				},
			}
			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(app.Flags[0].Value).To(PointTo(Equal(v)))
		},
			Entry("bool", cli.Bool(), true),
			Entry("Float32", cli.Float32(), float32(2.0)),
			Entry("Float64", cli.Float64(), float64(2.0)),
			Entry("Int", cli.Int(), int(16)),
			Entry("Int16", cli.Int16(), int16(16)),
			Entry("Int32", cli.Int32(), int32(16)),
			Entry("Int64", cli.Int64(), int64(16)),
			Entry("Int8", cli.Int8(), int8(16)),
			Entry("List", cli.List(), []string{"text", "plus"}),
			Entry("Map", cli.Map(), map[string]string{"key": "value"}),
			Entry("String", cli.String(), "text"),
			Entry("Uint", cli.Uint(), uint(19)),
			Entry("Uint16", cli.Uint16(), uint16(19)),
			Entry("Uint32", cli.Uint32(), uint32(19)),
			Entry("Uint64", cli.Uint64(), uint64(19)),
			Entry("Uint8", cli.Uint8(), uint8(19)),
			Entry("URL", cli.URL(), unwrap(url.Parse("https://localhost"))),
			Entry("Regexp", cli.Regexp(), regexp.MustCompile("blc")),
			Entry("IP", cli.IP(), net.ParseIP("127.0.0.1")),
			Entry("BigFloat", cli.BigFloat(), parseBigFloat("201.12")),
			Entry("BigInt", cli.BigInt(), parseBigInt("200")),
			Entry("Bytes", cli.Bytes(), []byte{4, 2}),
		)

		DescribeTable("errors", func(value any, v any) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "flag",
						Aliases: []string{"f"},
						Value:   value,
						Before: func(c *cli.Context) {
							Expect(func() { c.SetValue(v) }).To(Panic())
						},
					},
				},
			}
			_ = app.RunContext(context.TODO(), []string{"app"})
		},
			Entry("not supported TextMarshaler", new(textMarshaler), textMarshaler("")),
			Entry("not supported Value", new(customValue), &customValue{}),
		)
	})

	Describe("Before", func() {

		It("defers when set from initializer", func() {
			act := new(joeclifakes.FakeAction)
			act.ExecuteCalls(func(ctx context.Context) error {
				c := cli.FromContext(ctx)
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
			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
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

	Describe("Use", func() {

		It("invokes the action during the initializer", func() {
			act := new(joeclifakes.FakeAction)
			act.ExecuteCalls(func(ctx context.Context) error {
				c := cli.FromContext(ctx)
				Expect(c.IsInitializing()).To(BeTrue())
				return nil
			})
			app := &cli.App{
				Uses: func(c *cli.Context) {
					c.Use(act)

					Expect(act.ExecuteCallCount()).To(Equal(1))
				},
			}

			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		DescribeTable("error when timing after",
			func(timing func(*cli.Context)) {
				act := new(joeclifakes.FakeAction)
				ctx := &cli.Context{}
				timing(ctx)

				err := ctx.Use(act)
				Expect(err).To(HaveOccurred())
			},
			Entry("before timing", cli.SetBeforeTiming),
			Entry("action timing", cli.SetActionTiming),
			Entry("after timing", cli.SetAfterTiming),
		)

		DescribeTable("example actions that require Uses timing",
			func(act cli.Action) {
				app := &cli.App{
					Before: act,
				}

				err := app.RunContext(context.TODO(), []string{"app"})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(cli.ErrTimingTooLate))
			},
			Entry("AddFlag", cli.AddFlag(&cli.Flag{})),
			Entry("AddCommand", cli.AddCommand(&cli.Command{})),
			Entry("AddArg", cli.AddArg(&cli.Arg{})),
			Entry("RemoveArg", cli.RemoveArg("x")),
			Entry("PreventSetup", cli.PreventSetup),
			Entry("RemoveFlag", cli.RemoveFlag(nil)),
			Entry("RemoveCommand", cli.RemoveCommand(nil)),
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

	Describe("LookupFlag", func() {
		DescribeTable("examples", func(v interface{}, expected types.GomegaMatcher) {
			var actual *cli.Flag
			app := &cli.App{
				Action: func(c context.Context) {
					actual, _ = c.(*cli.Context).LookupFlag(v)
				},
				Flags: []*cli.Flag{
					{Name: "flag", Aliases: []string{"f"}},
					{Name: "g"},
				},
			}
			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name", "flag", WithTransform(flagName, Equal("flag"))),
			Entry("rune", 'f', WithTransform(flagName, Equal("flag"))),
			Entry("rune alias", 'g', WithTransform(flagName, Equal("g"))),
			Entry("Flag", &cli.Flag{Name: "flag"}, WithTransform(flagName, Equal("flag"))),
		)
	})

	Describe("LookupArg", func() {
		DescribeTable("examples", func(v interface{}, expected types.GomegaMatcher) {
			var actual *cli.Arg
			app := &cli.App{
				Action: func(c context.Context) {
					actual, _ = c.(*cli.Context).LookupArg(v)
				},
				Args: []*cli.Arg{
					{Name: "arg"},
				},
			}
			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name", "arg", WithTransform(argName, Equal("arg"))),
			Entry("index", 0, WithTransform(argName, Equal("arg"))),
			Entry("Arg", &cli.Arg{Name: "arg"}, WithTransform(argName, Equal("arg"))),
		)
	})

	Describe("LookupCommand", func() {
		DescribeTable("examples", func(v interface{}, expected types.GomegaMatcher) {
			var actual *cli.Command
			app := &cli.App{
				Name: "app",
				Action: func(c context.Context) {
					actual, _ = c.(*cli.Context).LookupCommand(v)
				},
				Commands: []*cli.Command{
					{Name: "cmd"},
				},
			}
			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name", "cmd", WithTransform(commandName, Equal("cmd"))),
			Entry("empty", "", WithTransform(commandName, Equal("app"))),
			Entry("Command", &cli.Command{Name: "cmd"}, WithTransform(commandName, Equal("cmd"))),
		)
	})

	Describe("Flags", func() {
		var (
			flags = func(names ...string) []*cli.Flag {
				res := make([]*cli.Flag, len(names))
				for i := range names {
					res[i] = &cli.Flag{Name: names[i]}
				}
				return res
			}
			names = func(f []*cli.Flag) string {
				if f == nil {
					return "<nil>"
				}
				res := make([]string, 0, len(f))
				for i := range f {
					// Don't include built-ins for the sake of this test
					name := f[i].Name
					if name == "help" || name == "version" || name == "zsh-completion" {
						continue
					}
					res = append(res, name)
				}
				return strings.Join(res, ",")
			}
		)
		DescribeTable("examples", func(factory func(cli.Action) *cli.App, flags, persistentFlags, localFlags string) {
			act := new(joeclifakes.FakeAction)
			app := factory(act)
			app.Initialize(context.Background())

			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(names(captured.Flags())).To(Equal(flags))
			Expect(names(captured.PersistentFlags())).To(Equal(persistentFlags))
			Expect(names(captured.LocalFlags())).To(Equal(localFlags))
		},
			Entry("app", func(act cli.Action) *cli.App {
				return &cli.App{
					Uses:  act,
					Flags: flags("f"),
				}
			}, "f", "<nil>", "f"),
			Entry("sub-command", func(act cli.Action) *cli.App {
				return &cli.App{
					Flags: flags("a", "b"),
					Commands: []*cli.Command{
						{
							Name:  "c",
							Flags: flags("d"),
							Uses:  act,
						},
					},
				}
			}, "a,b,d", "a,b", "d"),
			Entry("flag", func(act cli.Action) *cli.App {
				return &cli.App{
					Flags: []*cli.Flag{
						{Name: "a", Uses: act},
					},
				}
			}, "<nil>", "<nil>", "<nil>"),
			Entry("arg", func(act cli.Action) *cli.App {
				return &cli.App{
					Args: []*cli.Arg{
						{Name: "a", Uses: act},
					},
				}
			}, "<nil>", "<nil>", "<nil>"),
		)
	})

	Describe("LocalArgs", func() {
		var (
			names = func(f []*cli.Arg) string {
				if f == nil {
					return "<nil>"
				}
				res := make([]string, 0, len(f))
				for i := range f {
					// Don't include built-ins for the sake of this test
					name := f[i].Name
					res = append(res, name)
				}
				return strings.Join(res, ",")
			}
		)
		DescribeTable("examples", func(factory func(cli.Action) *cli.App, localArgs string) {
			act := new(joeclifakes.FakeAction)
			app := factory(act)
			app.Initialize(context.Background())

			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(names(captured.LocalArgs())).To(Equal(localArgs))
		},
			Entry("app", func(act cli.Action) *cli.App {
				return &cli.App{
					Uses: act,
					Args: cli.Args("a", new(string)),
				}
			}, "a"),
			Entry("sub-command", func(act cli.Action) *cli.App {
				return &cli.App{
					Commands: []*cli.Command{
						{
							Name: "c",
							Args: cli.Args("d", new(string)),
							Uses: act,
						},
					},
				}
			}, "d"),
			Entry("flag", func(act cli.Action) *cli.App {
				return &cli.App{
					Flags: []*cli.Flag{
						{Name: "a", Uses: act},
					},
				}
			}, "<nil>"),
			Entry("arg", func(act cli.Action) *cli.App {
				return &cli.App{
					Args: []*cli.Arg{
						{Name: "a", Uses: act},
					},
				}
			}, "<nil>"),
			Entry("value providing the convention", func(act cli.Action) *cli.App {
				return &cli.App{
					Args: []*cli.Arg{
						{Name: "a", Uses: cli.ProvideValueInitializer(&haveArgs{}, "me", act)},
					},
				}
			}, "x,y,z"),
		)
	})
})

var _ = Describe("FromContext", func() {
	var pass func()
	DescribeTable("examples", func(action any) {
		var called bool
		pass = func() {
			called = true
		}
		app := &cli.App{
			Action: action,
		}
		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeTrue(), "pass() must be called")
	},
		Entry(
			"trivial conversion",
			func(c *cli.Context) error {
				Expect(cli.FromContext(c)).To(BeIdenticalTo(c))
				pass()
				return nil
			},
		),
		Entry(
			"indirect conversion",
			func(c context.Context) error {
				Expect(cli.FromContext(c)).NotTo(BeNil())
				pass()
				return nil
			},
		),
		Entry(
			"wrapped with value",
			func(c context.Context) error {
				Expect(cli.FromContext(context.WithValue(c, privateKey("someKey"), ""))).NotTo(BeNil())
				pass()
				return nil
			},
		),
		Entry(
			"panic if missing",
			func() error {
				Expect(func() { cli.FromContext(context.TODO()) }).To(Panic())
				pass()
				return nil
			},
		),
	)
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
		Entry("empty matches cmd", "", "app"),
		Entry("empty matches flag", "", "-f"),
		Entry("empty matches expr", "", "<-expr>"),
	)

	DescribeTable("Match counterexamples",
		func(pattern string, path string) {
			p := cli.ContextPath(strings.Fields(path))
			Expect(p.Match(pattern)).To(BeFalse())
		},
		Entry("* doesn't match flag", "*", "app --flag"),
		Entry("* doesn't match arg", "*", "app <arg>"),
		Entry("* doesn't match expr", "*", "app <-expr>"),
		Entry("<> doesn't match expr", "<>", "app <-expr>"),
		Entry("flag doesn't match sub-command", "-", "app sub"),
		Entry("different sub-command", "app sub -", "app child -f"),
	)
})

func flagName(v interface{}) interface{} {
	return v.(*cli.Flag).Name
}

func argName(v interface{}) interface{} {
	return v.(*cli.Arg).Name
}

func commandName(v interface{}) interface{} {
	return v.(*cli.Command).Name
}

func parseBigInt(s string) *big.Int {
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); ok {
	}
	return v
}
func parseBigFloat(s string) *big.Float {
	v, _, _ := big.ParseFloat(s, 10, 53, big.ToZero)
	return v
}

type haveArgs struct{}

func (*haveArgs) LocalArgs() []*cli.Arg {
	return cli.Args("x", "", "y", "", "z", "")
}
