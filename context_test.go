package cli_test

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
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
			_ = app.RunContext(context.Background(), args)

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
			err := app.RunContext(context.Background(), args)
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
			_ = app.RunContext(context.Background(), args)

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
			_ = app.RunContext(context.Background(), args)

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.Value("f")).To(Equal([]string{"s", "r", "o"}))
		})

		It("contains value inherited in value target context", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name:  "a",
						Value: cli.List(),
						Uses:  cli.ProvideValueInitializer("", "<name>", cli.At(cli.ActionTiming, act)),
						NArg:  -1,
					},
				},
				Flags: []*cli.Flag{
					{Name: "f", Value: cli.Int()},
				},
			}

			args, _ := cli.Split("app -f 60 s")
			_ = app.RunContext(context.Background(), args)

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.Value("f")).To(Equal(60))
		})

		Describe("accessors", func() {
			DescribeTable("examples",
				func(v any, lookup func(*cli.Context) any, expected types.GomegaMatcher) {
					act := new(joeclifakes.FakeAction)
					app := &cli.App{
						Flags: []*cli.Flag{
							{Name: "a", Value: v},
						},
						Action: act,
					}

					args, _ := cli.Split("app")
					_ = app.RunContext(context.Background(), args)

					lk := cli.FromContext(act.ExecuteArgsForCall(0))
					Expect(lookup(lk)).To(expected)
				},
				Entry(
					"bool",
					cli.Bool(),
					func(lk *cli.Context) any { return lk.Bool("a") },
					Equal(false),
				),
				Entry(
					"File",
					&cli.File{},
					func(lk *cli.Context) any { return lk.File("a") },
					// Due to middleware setting up FS, we can only check type
					BeAssignableToTypeOf(&cli.File{}),
				),
				Entry(
					"FileSet",
					&cli.FileSet{},
					func(lk *cli.Context) any { return lk.FileSet("a") },
					// Due to middleware setting up FS, we can only check type
					BeAssignableToTypeOf(&cli.FileSet{}),
				),
				Entry(
					"Float32",
					cli.Float32(),
					func(lk *cli.Context) any { return lk.Float32("a") },
					Equal(float32(0)),
				),
				Entry(
					"Float64",
					cli.Float64(),
					func(lk *cli.Context) any { return lk.Float64("a") },
					Equal(float64(0)),
				),
				Entry(
					"Int",
					cli.Int(),
					func(lk *cli.Context) any { return lk.Int("a") },
					Equal(int(0)),
				),
				Entry(
					"Int16",
					cli.Int16(),
					func(lk *cli.Context) any { return lk.Int16("a") },
					Equal(int16(0)),
				),
				Entry(
					"Int32",
					cli.Int32(),
					func(lk *cli.Context) any { return lk.Int32("a") },
					Equal(int32(0)),
				),
				Entry(
					"Int64",
					cli.Int64(),
					func(lk *cli.Context) any { return lk.Int64("a") },
					Equal(int64(0)),
				),
				Entry(
					"Int8",
					cli.Int8(),
					func(lk *cli.Context) any { return lk.Int8("a") },
					Equal(int8(0)),
				),
				Entry(
					"Duration",
					cli.Duration(),
					func(lk *cli.Context) any { return lk.Duration("a") },
					Equal(time.Duration(0)),
				),
				Entry(
					"List",
					cli.List(),
					func(lk *cli.Context) any { return lk.List("a") },
					BeAssignableToTypeOf([]string{}),
				),
				Entry(
					"Map",
					cli.Map(),
					func(lk *cli.Context) any { return lk.Map("a") },
					BeAssignableToTypeOf(map[string]string{}),
				),
				Entry(
					"NameValue",
					&cli.NameValue{},
					func(lk *cli.Context) any { return lk.NameValue("a") },
					BeAssignableToTypeOf(&cli.NameValue{}),
				),
				Entry(
					"NameValues",
					cli.NameValues(),
					func(lk *cli.Context) any { return lk.NameValues("a") },
					BeAssignableToTypeOf(make([]*cli.NameValue, 0)),
				),
				Entry(
					"String",
					cli.String(),
					func(lk *cli.Context) any { return lk.String("a") },
					Equal(""),
				),
				Entry(
					"Uint",
					cli.Uint(),
					func(lk *cli.Context) any { return lk.Uint("a") },
					Equal(uint(0)),
				),
				Entry(
					"Uint16",
					cli.Uint16(),
					func(lk *cli.Context) any { return lk.Uint16("a") },
					Equal(uint16(0)),
				),
				Entry(
					"Uint32",
					cli.Uint32(),
					func(lk *cli.Context) any { return lk.Uint32("a") },
					Equal(uint32(0)),
				),
				Entry(
					"Uint64",
					cli.Uint64(),
					func(lk *cli.Context) any { return lk.Uint64("a") },
					Equal(uint64(0)),
				),
				Entry(
					"Uint8",
					cli.Uint8(),
					func(lk *cli.Context) any { return lk.Uint8("a") },
					Equal(uint8(0)),
				),
				Entry(
					"URL",
					cli.URL(),
					func(lk *cli.Context) any { return lk.URL("a") },
					BeAssignableToTypeOf(&url.URL{}),
				),
				Entry(
					"Regexp",
					cli.Regexp(),
					func(lk *cli.Context) any { return lk.Regexp("a") },
					BeAssignableToTypeOf(&regexp.Regexp{}),
				),
				Entry(
					"IP",
					cli.IP(),
					func(lk *cli.Context) any { return lk.IP("a") },
					BeAssignableToTypeOf(net.IP{}),
				),
				Entry(
					"BigFloat",
					cli.BigFloat(),
					func(lk *cli.Context) any { return lk.BigFloat("a") },
					BeAssignableToTypeOf(&big.Float{}),
				),
				Entry(
					"BigInt",
					cli.BigInt(),
					func(lk *cli.Context) any { return lk.BigInt("a") },
					BeAssignableToTypeOf(&big.Int{}),
				),
				Entry(
					"Value auto dereference",
					&hasDereference{v: &big.Int{}},
					func(lk *cli.Context) any { return lk.Value("a") },
					BeAssignableToTypeOf(&big.Int{}),
				),
				Entry(
					"Value auto dereference (typed)",
					&hasTypedDereference[*big.Int]{v: &big.Int{}},
					func(lk *cli.Context) any { return lk.Value("a") },
					BeAssignableToTypeOf(&big.Int{}),
				),
				Entry(
					"Value auto dereference (via flag.Getter)",
					&hasGetter{v: &big.Int{}},
					func(lk *cli.Context) any { return lk.Value("a") },
					BeAssignableToTypeOf(&big.Int{}),
				),
			)
		})
	})

	Describe("BindingLookup", func() {
		It("contains names of bindings that were used", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name: "a",
					},
				},
				Flags: []*cli.Flag{
					{
						Name: "f",
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app s")
			_ = app.RunContext(context.Background(), args)

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.BindingLookup().BindingNames()).To(Equal([]string{"a"}))
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
			_ = app.RunContext(context.Background(), args)

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
			_ = app.RunContext(context.Background(), args)

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.Raw("f")).To(Equal([]string{"-f", "dom"}))
			Expect(capturedContext.RawOccurrences("f")).To(Equal([]string{"dom"}))
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
			_ = app.RunContext(context.Background(), args)

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.Raw("")).To(Equal([]string{"-f", "sub"}))
			Expect(capturedContext.RawOccurrences("")).To(Equal([]string{"sub"}))
		})

		It("contains flag value from self context in Before", func() {
			// Addresses a bug: For a flag with zero occurrences, make sure that
			// RawOccurrences("") in the Before returns the correct value
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:   "e",
						Value:  cli.String(),
						Before: act,
					},
				},
			}

			args, _ := cli.Split("app")
			_ = app.RunContext(context.Background(), args)

			capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(capturedContext.Raw("")).To(Equal([]string{}))
			Expect(capturedContext.RawOccurrences("")).To(Equal([]string{}))
		})

		DescribeTable("examples",
			func(flag *cli.Flag, arguments string, raw, rawOccurrences []string, bindings [][]string) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Flags: []*cli.Flag{
						flag,
					},
					Action: act,
				}

				args, _ := cli.Split(arguments)
				_ = app.RunContext(context.Background(), args)

				capturedContext := cli.FromContext(act.ExecuteArgsForCall(0))
				Expect(capturedContext.Raw("f")).To(Equal(raw))
				Expect(capturedContext.RawOccurrences("f")).To(Equal(rawOccurrences))
				Expect(capturedContext.BindingLookup().Bindings("f")).To(Equal(bindings))
			},
			Entry(
				"bool flags",
				&cli.Flag{Name: "f", Value: cli.Bool()},
				"app -f",
				[]string{"-f", ""},
				[]string{""},
				[][]string{{"-f", ""}},
			),
			Entry(
				"multiple bool calls",
				&cli.Flag{Name: "f", Value: cli.Bool()},
				"app -f -f -f",
				[]string{"-f", "", "-f", "", "-f", ""},
				[]string{"", "", ""},
				[][]string{{"-f", ""}, {"-f", ""}, {"-f", ""}},
			),
			Entry(
				"string with quotes",
				&cli.Flag{Name: "f", Value: cli.String()},
				`app -f "text has spaces" -f ""`,
				[]string{"-f", "text has spaces", "-f", ""},
				[]string{"text has spaces", ""},
				[][]string{{"-f", "text has spaces"}, {"-f", ""}},
			),
			Entry(
				"name-value arg counter semantics",
				&cli.Flag{Name: "f", Value: new(cli.NameValue)},
				`app -f hello space`,
				[]string{"-f", "hello", "space"},
				[]string{"hello", "space"},
				[][]string{{"-f", "hello", "space"}},
			),
			Entry(
				"name-value arg counter semantics (long flag)",
				&cli.Flag{Name: "f", Value: new(cli.NameValue)},
				`app --f hello space`,
				[]string{"-f", "hello", "space"},
				[]string{"hello", "space"},
				[][]string{{"-f", "hello", "space"}},
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
				[][]string{{"--alias", ""}},
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
				[][]string{{"--alias", "9m32s"}},
			),
			Entry(
				"multiple instances long with alias",
				&cli.Flag{
					Name:    "f",
					Aliases: []string{"alias"},
					Value:   cli.Duration(),
				},
				"app --alias=9m32s -f 5m00s",
				[]string{"--alias", "9m32s", "-f", "5m00s"},
				[]string{"9m32s", "5m00s"},
				[][]string{{"--alias", "9m32s"}, {"-f", "5m00s"}},
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
			_ = app.RunContext(context.Background(), args)

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
			_ = app.RunContext(context.Background(), []string{"app"})
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
			_ = app.RunContext(context.Background(), []string{"app"})
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

			_ = app.RunContext(context.Background(), []string{"app"})
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

			_ = app.RunContext(context.Background(), []string{"app"})
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

				err := app.RunContext(context.Background(), []string{"app"})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(cli.ErrTimingTooLate))
			},
			Entry("AddFlag", cli.AddFlag(&cli.Flag{})),
			Entry("AddCommand", cli.AddCommand(&cli.Command{})),
			Entry("AddArg", cli.AddArg(&cli.Arg{})),
			Entry("AddAlias", cli.AddAlias("x")),
			Entry("RemoveAlias", cli.RemoveAlias("x")),
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

			_ = app.RunContext(context.Background(), nil)
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

		Context("when ErrSkipCommand", func() {

			BeforeEach(func() {
				walker = func(cmd *cli.Context) error {
					walkHelper(cmd)
					if cmd.Name() == "c" {
						return cli.ErrSkipCommand
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
				var actualID any
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
							case *expr.Expr:
								actualID = a.Data["id"]
							default:
								Fail(fmt.Sprintf("unexpected type %T", a))
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
								{
									Name: "arg3",
									Value: &expr.Expression{
										Exprs: []*expr.Expr{
											{
												Name: "expr",
												Uses: cli.Data("id", "4"),
											},
										},
									},
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

				_ = app.RunContext(context.Background(), []string{"app"})
				Expect(actualID).To(Equal(id))
			},
			Entry("empty self", []string{}, "0"),
			Entry("app name", []string{"app"}, "0"),
			Entry("sub-command", []string{"app", "sub"}, "1"),
			Entry("arg", []string{"app", "sub", "<arg>"}, "2"),
			Entry("expr", []string{"app", "sub", "<arg3>", "<-expr>"}, "4"),
		)
	})

	Describe("Path", func() {
		Context("when the app has a name", func() {
			DescribeTable("examples", func(v Fields) {
				actual := new(struct {
					App, Flag, Arg string
				})
				app := &cli.App{
					Name: "myname",
					Uses: func(c *cli.Context) {
						actual.App = c.Path().String()
					},
					Flags: []*cli.Flag{
						{
							Name: "flag",
							Uses: func(c *cli.Context) {
								actual.Flag = c.Path().String()
							},
						},
					},
					Args: []*cli.Arg{
						{
							Name: "arg",
							Uses: func(c *cli.Context) {
								actual.Arg = c.Path().String()
							},
						},
					},
				}

				// It's not relevant what the name of the app was in the Execute args
				_ = app.RunContext(context.Background(), []string{"app"})
				Expect(actual).To(PointTo(MatchFields(IgnoreExtras, v)))
			},
				Entry("app", Fields{"App": Equal("myname")}),
				Entry("flag", Fields{"Flag": Equal("myname --flag")}),
				Entry("arg", Fields{"Arg": Equal("myname <arg>")}),
			)
		})

		It("is nil for nil context", func() {
			actual := (*cli.Context)(nil).Path()
			Expect(actual).To(BeNil())
		})

		Context("when the app is unnamed", func() {

			var nameComesFromProcess = func() string {
				if runtime.GOOS == "windows" {
					return "joe-cli.test.exe"
				}
				return "joe-cli.test"
			}()

			DescribeTable("examples", func(v Fields) {
				actual := new(struct {
					Uses, Before, Action, After string
				})
				app := &cli.App{
					Uses: func(c *cli.Context) {
						actual.Uses = c.Path().String()
					},
					Before: func(c *cli.Context) {
						actual.Before = c.Path().String()
					},
					Action: func(c *cli.Context) {
						actual.Action = c.Path().String()
					},
					After: func(c *cli.Context) {
						actual.After = c.Path().String()
					},
				}

				// It's not relevant what the name of the app was in the Execute args
				_ = app.RunContext(context.Background(), []string{"called"})
				Expect(actual).To(PointTo(MatchFields(IgnoreExtras, v)))
			},
				Entry("Before", Fields{"Before": Equal(nameComesFromProcess)}),
				Entry("Action", Fields{"Action": Equal(nameComesFromProcess)}),
				Entry("After", Fields{"After": Equal(nameComesFromProcess)}),
				Entry("Uses", Fields{"Uses": Equal(nameComesFromProcess)}),
			)
		})
	})

	Describe("ContextOf", func() {
		It("obtains context for subtargets", func() {
			var (
				actualFlag    *cli.Context
				actualArg     *cli.Context
				actualCommand *cli.Context
				app           *cli.App
			)
			app = &cli.App{
				Action: func(c context.Context) {
					actualFlag = c.(*cli.Context).ContextOf(app.Flags[0])
					actualArg = c.(*cli.Context).ContextOf(app.Args[0])
					actualCommand = c.(*cli.Context).ContextOf(app.Commands[0])
				},
				Flags: []*cli.Flag{
					{Name: "f"},
				},
				Args: []*cli.Arg{
					{Name: "a"},
				},
				Commands: []*cli.Command{
					{Name: "c"},
				},
			}

			_ = app.RunContext(context.Background(), []string{"app"})
			Expect(actualFlag.Target()).To(Equal(app.Flags[0]))
			Expect(actualCommand.Target()).To(Equal(app.Commands[0]))
			Expect(actualArg.Target()).To(Equal(app.Args[0]))
		})

		Describe("resolving from a command", func() {

			DescribeTable("examples", func(name any, expected types.GomegaMatcher) {
				var actual any
				app := &cli.App{
					Name: "theApp",
					Action: func(c context.Context) {
						actual = c.(*cli.Context).ContextOf(name).Target()
					},
					Flags: []*cli.Flag{
						{Name: "flag", Aliases: []string{"f"}},
					},
					Args: []*cli.Arg{
						{Name: "a"},
					},
				}
				_ = app.RunContext(context.Background(), []string{"app"})
				Expect(actual).To(expected)
			},
				Entry("name of flag", "flag", WithTransform(flagName, Equal("flag"))),
				Entry("name of arg", "a", WithTransform(argName, Equal("a"))),
				Entry("rune", 'f', WithTransform(flagName, Equal("flag"))),
				Entry("index", 0, WithTransform(argName, Equal("a"))),
				Entry("self", "", WithTransform(commandName, Equal("theApp"))),
			)

		})

		Describe("resolving from an option", func() {

			DescribeTable("examples", func(name any, expected types.GomegaMatcher) {
				var actual any
				app := &cli.App{
					Name: "theApp",
					Flags: []*cli.Flag{
						{Name: "flag", Aliases: []string{"f"}},
						{
							Name: "self",
							Action: func(c context.Context) {
								actual = c.(*cli.Context).ContextOf(name).Target()
							},
						},
					},
					Args: []*cli.Arg{
						{Name: "a"},
					},
				}
				_ = app.RunContext(context.Background(), []string{"app", "--self", "v"})
				Expect(actual).To(expected)
			},
				Entry("name of flag", "flag", WithTransform(flagName, Equal("flag"))),
				Entry("name of arg", "a", WithTransform(argName, Equal("a"))),
				Entry("rune", 'f', WithTransform(flagName, Equal("flag"))),
				Entry("index", 0, WithTransform(argName, Equal("a"))),
				Entry("self", "", WithTransform(flagName, Equal("self"))),
			)
		})
	})

	Describe("Target", func() {
		It("obtains target from the context", func() {
			var (
				actualFlag    any
				actualArg     any
				actualCommand any
				actualExpr    any
			)
			app := &cli.App{
				Action: func() {},
				Flags: []*cli.Flag{
					{
						Name: "f",
						Uses: func(c *cli.Context) {
							actualFlag = c.Target()
						},
					},
				},
				Args: []*cli.Arg{
					{
						Name: "a",
						Uses: func(c *cli.Context) {
							actualArg = c.Target()
						},
					},
					{
						Name: "expression",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "e",
									Uses: func(c *cli.Context) {
										actualExpr = c.Target()
									},
								},
							},
						},
					},
				},
				Commands: []*cli.Command{
					{
						Name: "c",
						Uses: func(c *cli.Context) {
							actualCommand = c.Target()
						},
					},
				},
			}

			_ = app.RunContext(context.Background(), []string{"app", "c"})
			Expect(actualFlag).To(Equal(app.Flags[0]))
			Expect(actualCommand).To(Equal(app.Commands[0]))
			Expect(actualArg).To(Equal(app.Args[0]))
			Expect(actualExpr).To(Equal(app.Args[1].Value.(*expr.Expression).Exprs[0]))
		})

		DescribeTable("examples", func(name any, expected types.GomegaMatcher) {
			var actual any
			app := &cli.App{
				Name: "theApp",
				Action: func(c context.Context) {
					actual = c.(*cli.Context).ContextOf(name).Target()
				},
				Flags: []*cli.Flag{
					{Name: "flag", Aliases: []string{"f"}},
				},
				Args: []*cli.Arg{
					{Name: "a"},
				},
			}
			_ = app.RunContext(context.Background(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name of flag", "flag", WithTransform(flagName, Equal("flag"))),
			Entry("name of arg", "a", WithTransform(argName, Equal("a"))),
			Entry("rune", 'f', WithTransform(flagName, Equal("flag"))),
			Entry("index", 0, WithTransform(argName, Equal("a"))),
			Entry("self", "", WithTransform(commandName, Equal("theApp"))),
		)
	})

	Describe("LookupFlag", func() {
		DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
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
			_ = app.RunContext(context.Background(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name", "flag", WithTransform(flagName, Equal("flag"))),
			Entry("rune", 'f', WithTransform(flagName, Equal("flag"))),
			Entry("rune alias", 'g', WithTransform(flagName, Equal("g"))),
			Entry("Flag", &cli.Flag{Name: "flag"}, WithTransform(flagName, Equal("flag"))),
			Entry("Flag typed nil", (*cli.Flag)(nil), BeNil()),
		)
	})

	Describe("LookupArg", func() {
		DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
			var actual *cli.Arg
			app := &cli.App{
				Action: func(c context.Context) {
					actual, _ = c.(*cli.Context).LookupArg(v)
				},
				Args: []*cli.Arg{
					{Name: "arg"},
				},
			}
			_ = app.RunContext(context.Background(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name", "arg", WithTransform(argName, Equal("arg"))),
			Entry("index", 0, WithTransform(argName, Equal("arg"))),
			Entry("Arg", &cli.Arg{Name: "arg"}, WithTransform(argName, Equal("arg"))),
			Entry("Arg typed nil", (*cli.Arg)(nil), BeNil()),
		)

		It("returns false on out of range arg index", func() {
			var (
				actual *cli.Arg
				ok     bool
			)
			app := &cli.App{
				Uses: func(c context.Context) {
					actual, ok = c.(*cli.Context).LookupArg(0)
				},
			}
			_, _ = app.Initialize(context.Background())
			Expect(actual).To(BeNil())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("LookupValueTarget", func() {

		var myValue = new(struct{})

		DescribeTable("examples", func(v string, expected types.GomegaMatcher) {
			var actual any
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Uses: cli.ProvideValueInitializer(myValue, "name"),
						Action: func(c *cli.Context) {
							actual, _ = c.LookupValueTarget(v)
						},
					},
				},
			}
			_ = app.RunContext(context.Background(), []string{"app", "_"})
			Expect(actual).To(expected)
		},
			Entry("name", "name", BeIdenticalTo(myValue)),
		)
	})

	Describe("LookupCommand", func() {
		DescribeTable("examples", func(v any, expected types.GomegaMatcher) {
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
			_ = app.RunContext(context.Background(), []string{"app"})
			Expect(actual).To(expected)
		},
			Entry("name", "cmd", WithTransform(commandName, Equal("cmd"))),
			Entry("empty", "", WithTransform(commandName, Equal("app"))),
			Entry("Command", &cli.Command{Name: "cmd"}, WithTransform(commandName, Equal("cmd"))),
			Entry("Command typed nil", (*cli.Command)(nil), BeNil()),
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
				value := &haveArgs{
					Args: cli.Args("x", new(string), "y", new(string), "z", new(string)),
				}
				return &cli.App{
					Args: []*cli.Arg{
						{
							Name: "a",
							Uses: cli.ProvideValueInitializer(value, "me", act),
						},
					},
				}
			}, "x,y,z"),
		)

		It("can add set local args via convention", func() {
			value := &haveArgs{
				Args: cli.Args("x", new(string), "y", new(string), "z", new(string)),
			}
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name: "a",
						Uses: cli.Pipeline(
							cli.ProvideValueInitializer(value, "me", cli.AddArgs(cli.Args("o", new(int))...)),
						),
					},
				},
			}
			app.Initialize(context.Background())
			Expect(value.Args).To(HaveLen(4))
		})
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
				Expect(func() { cli.FromContext(context.Background()) }).To(Panic())
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

func flagName(v any) any {
	return v.(*cli.Flag).Name
}

func argName(v any) any {
	return v.(*cli.Arg).Name
}

func commandName(v any) any {
	return v.(*cli.Command).Name
}

func parseBigInt(s string) *big.Int {
	v := new(big.Int)
	if _, ok := v.SetString(s, 10); ok {
		return v
	}
	return nil
}
func parseBigFloat(s string) *big.Float {
	v, _, _ := big.ParseFloat(s, 10, 53, big.ToZero)
	return v
}

type haveArgs struct {
	Args []*cli.Arg
	Data map[string]any
}

func (a *haveArgs) LocalArgs() []*cli.Arg {
	return a.Args
}

func (a *haveArgs) SetLocalArgs(args []*cli.Arg) error {
	a.Args = args
	return nil
}

func (a *haveArgs) SetData(k string, v any) {
	if a.Data == nil {
		a.Data = map[string]any{}
	}
	a.Data[k] = v
}

func (*haveArgs) Set(string) error {
	return nil
}

func (*haveArgs) String() string {
	return ""
}
