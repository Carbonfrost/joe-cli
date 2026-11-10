// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"bytes"
	"context"
	"math/big"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/joe-clifakes"
	"github.com/Carbonfrost/joe-cli/internal/privatekey"
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Flag", func() {

	Describe("Action", func() {
		var (
			act       *joeclifakes.FakeAction
			app       *cli.App
			arguments = "app -f value"
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			app = &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:    "f",
						Aliases: []string{"alias"},
						Action:  act,
					},
				},
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split(arguments)
			app.RunContext(context.Background(), args...)
		})

		It("executes action on setting flag", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("contains args in captured context", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Args()).To(Equal([]string{"-f", "value"}))
		})

		It("provides properly initialized context", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Name()).To(Equal("-f"))
			Expect(captured.Path().String()).To(Equal("app -f"))
		})

		It("contains the value in the context", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Value("")).To(Equal("value"))
		})

		It("contains the correct Occurrences count", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Occurrences("")).To(Equal(1))
		})

		It("obtains context path", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Path().IsFlag()).To(BeTrue())
			Expect(captured.Path().Last()).To(Equal("-f"))
			Expect(captured.Path().String()).To(Equal("app -f"))
		})

		Context("when using the alias", func() {

			BeforeEach(func() {
				arguments = "app --alias value"
			})

			It("executes action on setting flag via its alias", func() {
				Expect(act.ExecuteCallCount()).To(Equal(1))
			})

			It("provides properly initialized context", func() {
				// The name is still -f despite using the alias
				captured := cli.FromContext(act.ExecuteArgsForCall(0))
				Expect(captured.Name()).To(Equal("-f"))
				Expect(captured.Path().String()).To(Equal("app -f"))
			})

			It("contains the value in the context", func() {
				captured := cli.FromContext(act.ExecuteArgsForCall(0))
				Expect(captured.Value("")).To(Equal("value"))
			})

			It("contains the correct Occurrences count", func() {
				captured := cli.FromContext(act.ExecuteArgsForCall(0))
				Expect(captured.Occurrences("")).To(Equal(1))
			})
		})

		Context("when using name and alias", func() {

			BeforeEach(func() {
				arguments = "app -f foo --alias bar -f baz"
			})

			It("executes action on setting flag via its alias", func() {
				Expect(act.ExecuteCallCount()).To(Equal(1))
			})

			It("contains the value in the context", func() {
				captured := cli.FromContext(act.ExecuteArgsForCall(0))
				Expect(captured.Value("")).To(Equal("baz")) // winner due to being last
			})

			It("contains the correct Occurrences count", func() {
				captured := cli.FromContext(act.ExecuteArgsForCall(0))
				Expect(captured.Occurrences("")).To(Equal(3))
			})
		})

		Context("for a persistent flag", func() {
			BeforeEach(func() {
				act = new(joeclifakes.FakeAction)
				app = &cli.App{
					Name: "app",
					Commands: []*cli.Command{
						{
							Name: "sub",
						},
					},
					Flags: []*cli.Flag{
						{
							Name:   "f",
							Action: act,
						},
					},
				}
			})

			Context("set within the sub-command", func() {
				BeforeEach(func() {
					arguments = "app sub -f value"
				})

				It("executes action on setting flag", func() {
					Expect(act.ExecuteCallCount()).To(Equal(1))
				})

				It("provides properly initialized context", func() {
					captured := cli.FromContext(act.ExecuteArgsForCall(0))
					Expect(captured.Name()).To(Equal("-f"))
					Expect(captured.Path().String()).To(Equal("app sub -f"))
				})

				It("contains the value in the context", func() {
					captured := cli.FromContext(act.ExecuteArgsForCall(0))
					Expect(captured.Value("")).To(Equal("value"))
				})
			})

			It("an error to re-use persistent flag names", func() {
				app := &cli.App{
					Name: "app",
					Commands: []*cli.Command{
						{
							Name: "sub",
							Flags: []*cli.Flag{
								{Name: "f"},
							},
						},
					},
					Flags: []*cli.Flag{
						{Name: "f"},
					},
				}
				_, err := app.Initialize(context.Background())
				Expect(err).To(MatchError(ContainSubstring(`internal error, at "app sub" (initial timing):`)))
				Expect(err).To(MatchError(ContainSubstring(`duplicate name used: "f" (persistent from app)`)))
			})
		})
	})

	Describe("Value", func() {

		It("generates an internal error if the value is not supported", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: "not supported value",
					},
				},
				Action: act,
			}

			_, err := app.Initialize(context.Background())
			Expect(err).To(MatchError(`internal error, at "app -f" (initial timing): unsupported flag type: string`))
		})
	})

	It("can set and define name and value by initializer", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Uses: func(c *cli.Context) {
						f := c.Flag()
						f.Name = "uses"
						f.Value = new(bool)
						f.Action = act
					},
				},
			},
		}

		err := app.RunContext(context.Background(), []string{"app", "--uses"}...)

		// In particular, we expect --uses to be available and not cause usage
		// error
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))

		Expect(app.Flags[0].Name).To(Equal("uses"))
	})

	It("can set additional options by initializer", func() {
		var capture bytes.Buffer
		defer disableConsoleColor()()

		app := &cli.App{
			Name:   "app",
			Stderr: &capture,
			Flags: []*cli.Flag{
				{
					Name: "do-not-show",
					Uses: func(c *cli.Context) {
						c.Flag().Options |= cli.Hidden
					},
				},
			},
			Action: cli.DisplayHelpScreen(),
		}

		err := app.RunContext(context.Background(), "app")

		// In particular, we expect --do-not-show to be hidden
		Expect(err).To(Succeed())
		Expect(capture.String()).NotTo(ContainSubstring("--do-not-show"))
	})

	Context("when environment variables are set", func() {
		var (
			actual     string
			arguments  string
			beforeFlag *joeclifakes.FakeAction
		)

		BeforeEach(func() {
			actual = ""
			arguments = "app"
			beforeFlag = new(joeclifakes.FakeAction)
		})

		JustBeforeEach(func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "f",
						EnvVars: []string{"_GOCLI_F"},
						Value:   &actual,
						Before:  beforeFlag,
					},
				},
			}

			os.Setenv("_GOCLI_F", "environment value")
			args, _ := cli.Split(arguments)
			app.RunContext(context.Background(), args...)
		})

		It("sets up value from environment", func() {
			Expect(actual).To(Equal("environment value"))
		})

		Context("when option also set", func() {
			BeforeEach(func() {
				arguments = "app -f 'option text'"
			})

			It("sets up value from option", func() {
				Expect(actual).To(Equal("option text"))
			})
		})

		Context("when accessed in the Before pipeline", func() {
			BeforeEach(func() {
				arguments = "app"
				beforeFlag.ExecuteStub = func(ctx context.Context) error {
					c := cli.FromContext(ctx)
					Expect(c.Value("f")).To(Equal("environment value"))
					return nil
				}
			})

			It("sets up value from option", func() {
				context := beforeFlag.ExecuteArgsForCall(0)
				Expect(context.Value("f")).To(Equal("environment value"))
			})
		})
	})

	Context("when a custom Value is used", func() {

		It("applies the conversion", func() {
			t := new(temperature)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "s",
						Value: t,
					},
				},
			}

			arguments, _ := cli.Split("app -sC")
			app.RunContext(context.Background(), arguments...)
			Expect(*t).To(Equal(temperature("Celsius")))
		})

		It("propagates the conversion error", func() {
			t := new(temperature)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "s",
						Value: t,
					},
				},
			}

			arguments, _ := cli.Split("app -sK")
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("not supported"))
		})
	})

	Context("when the value is Optional", func() {
		DescribeTable("examples", func(flag any, expected any) {
			var actual any
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "show",
						Aliases: []string{"s"},
						Value:   flag,
						Options: cli.Optional,
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("show")
				},
			}

			arguments, _ := cli.Split("app -s")
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("bool", cli.Bool(), true),
			Entry("float32", cli.Float32(), float32(1.0)),
			Entry("float64", cli.Float64(), float64(1.0)),
			Entry("int", cli.Int(), 1),
			Entry("int64", cli.Int64(), int64(1)),
			Entry("int32", cli.Int32(), int32(1)),
			Entry("int16", cli.Int16(), int16(1)),
			Entry("int8", cli.Int8(), int8(1)),
			Entry("uint64", cli.Uint64(), uint64(1)),
			Entry("uint32", cli.Uint32(), uint32(1)),
			Entry("uint16", cli.Uint16(), uint16(1)),
			Entry("uint8", cli.Uint8(), uint8(1)),
			Entry("IP", cli.IP(), net.ParseIP("127.0.0.1")),
			Entry("Duration", cli.Duration(), time.Second),

			Entry("long bool", cli.Bool(), true),
			Entry("long float32", cli.Float32(), float32(1.0)),
			Entry("long float64", cli.Float64(), float64(1.0)),
			Entry("long int", cli.Int(), 1),
			Entry("long uint64", cli.Uint64(), uint64(1)),
			Entry("long IP", cli.IP(), net.ParseIP("127.0.0.1")),
			Entry("long Duration", cli.Duration(), time.Second),
			Entry("string", new("x"), ""),
			Entry("[]string", new([]string{"x"}), []string{}),
			Entry("[]byte", new([]byte{2}), []byte{}),

			Entry("resettable Value", new(resettableValue), &resettableValue{resets: 1}),
		)

		DescribeTable("errors", func(flag any) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "show",
						Value:   flag,
						Options: cli.Optional,
					},
				},
				Action: func() {},
				Name:   "app",
			}

			_, err := app.Initialize(context.Background())
			Expect(err).To(MatchError(ContainSubstring(`internal error, at "app --show" (initial timing):`)))
			Expect(err).To(MatchError(MatchRegexp(`invalid type .+ for Optional flag`)))
		},
			Entry("Value", new(joeclifakes.FakeValue)),
		)

		DescribeTable("OptionalValue examples", func(flag any, args string, expected any) {
			var actual any
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "show",
						Aliases: []string{"s"},
						Value:   flag,
						Uses:    cli.OptionalValue(expected),
						Options: cli.No,
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("show")
				},
			}

			arguments, _ := cli.Split(args)
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("Regexp", cli.Regexp(), "app -s", regexp.MustCompile("hello")),
			Entry("BigInt", cli.BigInt(), "app -s", big.NewInt(2)),
			Entry("BigFloat", cli.BigFloat(), "app -s", big.NewFloat(3)),
			Entry("URL", cli.URL(), "app -s", unwrap(url.Parse("https://hello.example"))),
			Entry("List", cli.List(), "app -s", []string{"OK"}),
			Entry("Map", cli.Map(), "app -s", map[string]string{"A": "B"}),
			Entry("Bytes", cli.Bytes(), "app -s", []byte("ab")),

			Entry("trailing equal sign", cli.Bytes(), "app --show=", []byte("ab")), // Interpreted as resetting value
			Entry("Only No will reset", cli.Bytes(), "app --no-show", []byte{}),
		)

		It("following is treated as argument", func() {
			var actual, args any
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "s",
						Value:   cli.Float64(),
						Options: cli.Optional,
					},
				},
				Args: []*cli.Arg{
					{
						Name: "a",
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("s")
					args = c.Value("a")
				},
			}

			arguments, _ := cli.Split("app -s following")
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(float64(1.0)))
			Expect(args).To(Equal("following"))
		})

		It("run-in is treated as argument on short option", func() {
			var actual any
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "s",
						Value:   cli.Float64(),
						Options: cli.Optional,
					},
				},
				Args: []*cli.Arg{
					{
						Name: "a",
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("s")
				},
			}

			arguments, _ := cli.Split("app -s2.0")
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(float64(2.0)))
		})

		It("equals is treated as argument on long option", func() {
			var actual any
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "show",
						Value:   cli.Float64(),
						Options: cli.Optional,
					},
				},
				Args: []*cli.Arg{
					{
						Name: "a",
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("show")
				},
			}

			arguments, _ := cli.Split("app --show=2.0")
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(float64(2.0)))
		})
	})

	Context("when the value is OptionalValue", func() {
		It("applies the value when specified", func() {
			t := new(temperature)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "s",
						Value: t,
						Uses:  cli.OptionalValue("Fahrenheit"),
					},
				},
			}

			arguments, _ := cli.Split("app -sC")
			app.RunContext(context.Background(), arguments...)
			Expect(*t).To(Equal(temperature("Celsius")))
		})

		It("applies the optional value when not specified", func() {
			t := new(string)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "s",
						Value: t,
						Uses:  cli.OptionalValue("tls1.2"),
					},
				},
			}

			arguments, _ := cli.Split("app -s")
			_ = app.RunContext(context.Background(), arguments...)
			Expect(*t).To(Equal("tls1.2"))
		})
	})

	Context("when a NonPersistent flag", func() {
		It("does not act again in the sub-command", func() {
			var acted int
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "once",
						Value:   cli.Bool(),
						Options: cli.NonPersistent,
						Action:  func() { acted++ },
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Action: func(c *cli.Context) error { return nil },
					},
				},
			}

			arguments, _ := cli.Split("app --once sub")
			Expect(app.RunContext(context.Background(), arguments...)).To(Succeed())
			Expect(acted).To(Equal(1))
		})

		It("is not among the persistent flags of the sub-command", func() {
			var persistent []string
			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "local", Value: cli.Bool(), Options: cli.NonPersistent},
					{Name: "global", Value: cli.Bool()},
				},
				Commands: []*cli.Command{
					{
						Name: "sub",
						Action: func(c *cli.Context) error {
							for _, f := range c.PersistentFlags() {
								persistent = append(persistent, f.Name)
							}
							return nil
						},
					},
				},
			}

			arguments, _ := cli.Split("app sub")
			Expect(app.RunContext(context.Background(), arguments...)).To(Succeed())
			Expect(persistent).To(ContainElement("global"))
			Expect(persistent).NotTo(ContainElement("local"))
		})

		It("is a usage error to use", func() {
			p := 1600
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "nope",
						Value:   &p,
						Options: cli.NonPersistent,
					},
				},
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			}

			arguments, _ := cli.Split("app sub --nope 19")
			err := app.RunContext(context.Background(), arguments...)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("unknown option: --nope"))
			Expect(p).To(Equal(1600)) // unchanged
		})
	})

	Describe("Synopsis", func() {

		DescribeTable("examples",
			func(f *cli.Flag, expected string) {
				cli.Initialized(f)
				Expect(f.Synopsis()).To(Equal(expected))
			},
			Entry(
				"bool flag no placeholders",
				&cli.Flag{
					Name:  "o",
					Value: cli.Bool(),
				},
				"-o",
			),
			Entry(
				"IsBoolFlag convention",
				&cli.Flag{
					Name:  "o",
					Value: new(boolFlag),
				},
				"-o",
			),
			Entry(
				"int flag no placeholders",
				&cli.Flag{
					Name:  "o",
					Value: cli.Int(),
				},
				"-o NUMBER",
			),
			Entry(
				"String flag no placeholders",
				&cli.Flag{
					Name:  "o",
					Value: cli.String(),
				},
				"-o STRING",
			),
			Entry(
				"long flag with placeholder",
				&cli.Flag{
					Name:     "otown",
					HelpText: "{USE}",
					Value:    cli.Int(),
				},
				"--otown=USE",
			),
			Entry(
				"short flag with placeholder",
				&cli.Flag{
					Name:     "o",
					HelpText: "{USE}",
					Value:    cli.Int(),
				},
				"-o USE",
			),
			Entry(
				"aliases flag with placeholder",
				&cli.Flag{
					Name:     "otown",
					Aliases:  []string{"o", "other", "u"},
					HelpText: "{USE}",
					Value:    cli.Int(),
				},
				"-o, --otown=USE",
			),
			Entry(
				"map",
				&cli.Flag{
					Name:  "m",
					Value: cli.Map(),
				},
				"-m NAME=VALUE",
			),
			Entry(
				"file",
				&cli.Flag{
					Name:  "f",
					Value: &cli.File{},
				},
				"-f FILE",
			),
			Entry(
				"file set",
				&cli.Flag{
					Name:  "f",
					Value: &cli.FileSet{},
				},
				"-f FILES",
			),
			Entry(
				"URL",
				&cli.Flag{
					Name:  "f",
					Value: cli.URL(),
				},
				"-f URL",
			),
			Entry(
				"IP",
				&cli.Flag{
					Name:  "f",
					Value: cli.IP(),
				},
				"-f IP",
			),
			Entry(
				"Regexp",
				&cli.Flag{
					Name:  "f",
					Value: cli.Regexp(),
				},
				"-f PATTERN",
			),
			Entry(
				"Synopsis provider",
				&cli.Flag{
					Name:  "f",
					Value: new(temperature),
				},
				"-f {Fahrenheit|Celsius}",
			),
			Entry(
				"Synopsis data",
				&cli.Flag{
					Name: "reason",
					Uses: cli.Data(privatekey.Synopsis, &synopsis.Flag{
						Long:    "[no-]reason",
						Primary: "[no-]reason",
						Names:   []string{"--[no-]reason"},
						Value:   &synopsis.Value{},
					}),
					Value: cli.Bool(),
				},
				"--[no-]reason",
			),
			Entry(
				"Synopsis via UsageText",
				&cli.Flag{
					Name:      "f",
					UsageText: "Usage",
				},
				"-f Usage",
			),
		)

		Describe("optional values", func() {
			DescribeTable("examples",
				func(f *cli.Flag, expected string) {
					app := &cli.App{
						Name:   "app",
						Flags:  []*cli.Flag{f},
						Action: func() {},
					}
					_, err := app.Initialize(context.Background())
					Expect(err).NotTo(HaveOccurred())
					Expect(f.Synopsis()).To(Equal(expected))
				},
				Entry(
					"OptionalValue nominal",
					&cli.Flag{
						Name:    "secure",
						Aliases: []string{"s"},
						Value:   cli.String(),
						Uses:    cli.OptionalValue("TLS1.2"),
					},
					"-s, --secure[=TLS1.2]",
				),
				Entry(
					"OptionalValue (short flag only)",
					&cli.Flag{
						Name:  "s",
						Value: cli.String(),
						Uses:  cli.OptionalValue("TLS1.2"),
					},
					"-s[TLS1.2]",
				),
				Entry(
					"Duration",
					&cli.Flag{
						Name:  "timeout",
						Value: cli.Duration(),
						Uses:  cli.OptionalValue(30 * time.Second),
					},
					"--timeout[=30s]",
				),
				Entry(
					"List",
					&cli.Flag{
						Name:  "tag",
						Value: cli.List(),
						Uses:  cli.OptionalValue([]string{"latest", "stable"}),
					},
					"--tag[=latest,stable]",
				),
				Entry(
					"OptionalValue (user authored placeholder)",
					&cli.Flag{
						Name:     "secure",
						HelpText: "Use {VERSION} of TLS",
						Value:    cli.String(),
						Uses:     cli.OptionalValue("TLS1.2"),
					},
					"--secure[=VERSION]",
				),
				Entry(
					"OptionalValue (value synopsis convention)",
					&cli.Flag{
						Name:  "temp",
						Value: new(temperature),
						Uses:  cli.OptionalValue(temperature("Fahrenheit")),
					},
					"--temp[={Fahrenheit|Celsius}]",
				),
				Entry(
					"Optional nominal",
					&cli.Flag{
						Name:    "level",
						Value:   cli.Int(),
						Options: cli.Optional,
					},
					"--level[=NUMBER]",
				),
				Entry(
					"Optional (boolean flag)",
					&cli.Flag{
						Name:    "verbose",
						Value:   cli.Bool(),
						Options: cli.Optional,
					},
					"--verbose",
				),
			)
		})

	})

	Describe("Names", func() {

		DescribeTable("examples", func(f *cli.Flag, long, short string, names []string) {
			Expect(f.LongName()).To(Equal(long))
			Expect(f.ShortName()).To(Equal(short))
			Expect(f.Names()).To(Equal(names))
		},
			Entry("long only", &cli.Flag{Name: "flag"}, "--flag", "", []string{"--flag"}),
			Entry("short only", &cli.Flag{Name: "f"}, "--f", "-f", []string{"-f"}),
			Entry("long in alias", &cli.Flag{Name: "f", Aliases: []string{"g", "flag"}}, "--flag", "-f", []string{"-f", "-g", "--flag"}),
			Entry("short in alias", &cli.Flag{Name: "flag", Aliases: []string{"rag", "f"}}, "--flag", "-f", []string{"--flag", "--rag", "-f"}),
		)
	})

	DescribeTable("initializers", func(act cli.Action, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "a",
			Flags: []*cli.Flag{
				{
					Uses: act,
					Name: "a",
				},
			},
		}

		args, _ := cli.Split("app -a s")
		err := app.RunContext(context.Background(), args...)
		Expect(err).NotTo(HaveOccurred())
		cmd, _ := app.Command("")
		Expect(cmd.Flags[0]).To(PointTo(expected))
	},
		Entry(
			"Category",
			cli.Category("abc"),
			MatchFields(IgnoreExtras, Fields{"Category": Equal("abc")}),
		),
		Entry(
			"Alias",
			cli.Alias("abc"),
			MatchFields(IgnoreExtras, Fields{"Aliases": Equal([]string{"abc"})}),
		),
		Entry(
			"ManualText",
			cli.ManualText("abc"),
			MatchFields(IgnoreExtras, Fields{"ManualText": Equal("abc")}),
		),
		Entry(
			"HelpText",
			cli.HelpText("abc"),
			MatchFields(IgnoreExtras, Fields{"HelpText": Equal("abc")}),
		),
		Entry(
			"UsageText",
			cli.UsageText("abc"),
			MatchFields(IgnoreExtras, Fields{"UsageText": Equal("abc")}),
		),
		Entry(
			"Description",
			cli.Description("abc"),
			MatchFields(IgnoreExtras, Fields{"Description": Equal("abc")}),
		),
		Entry(
			"SetCompletion",
			cli.SetCompletion(cli.ValueCompletion("ok")),
			MatchFields(IgnoreExtras, Fields{"Completion": Not(BeNil())}),
		),
	)

	Describe("visibility", func() {
		DescribeTable("examples", func(flag *cli.Flag, finder string, visibleExpected bool) {
			var found *cli.Flag
			app := &cli.App{
				Flags: []*cli.Flag{flag},
				Action: func(c *cli.Context) {
					parent, _ := c.FindTarget(strings.Fields("app " + finder))
					found = parent.Target().(*cli.Flag)
				},
				Name: "app",
			}
			args, _ := cli.Split("app")
			err := app.RunContext(context.Background(), args...)
			Expect(err).NotTo(HaveOccurred())
			Expect(cli.IsVisible(found)).To(Equal(visibleExpected))
		},
			Entry("nominal", &cli.Flag{Name: "visible"}, "-visible", true),
			Entry("visible", &cli.Flag{Name: "visible", Options: cli.Visible}, "-visible", true),
			Entry("implicitly hidden by name", &cli.Flag{Name: "_hidden"}, "-_hidden", false),
			Entry("disable implicitly hidden behavior (self)", &cli.Flag{Name: "_hidden", Options: cli.DisableAutoVisibility}, "-_hidden", true),
			Entry("explicitly made visible implicitly hidden behavior", &cli.Flag{Name: "_hidden", Options: cli.Visible}, "-_hidden", true),
			Entry("hidden wins over visible", &cli.Flag{Name: "hidden", Options: cli.Visible | cli.Hidden}, "-hidden", false),
		)
	})

})

var _ = Describe("PseudoCommand with added commands", func() {

	var (
		ran []string
		run func(flagUses cli.Action, args string) error
	)

	record := func(name string) *cli.Command {
		return &cli.Command{
			Name:   name,
			Action: func() { ran = append(ran, name) },
		}
	}

	BeforeEach(func() {
		ran = nil
		run = func(flagUses cli.Action, args string) error {
			app := &cli.App{
				Name:   "app",
				Flags:  []*cli.Flag{{Uses: flagUses}},
				Action: func() { ran = append(ran, "app") },
			}
			arguments, _ := cli.Split(args)
			return app.RunContext(context.Background(), arguments...)
		}
	})

	It("adds commands with AddCommand", func() {
		uses := cli.Pipeline(
			cli.PseudoCommand(),
			cli.AddCommand(record("one")),
			cli.AddCommand(record("two")),
		)
		Expect(run(uses, "app -g two")).To(Succeed())
		Expect(ran).To(Equal([]string{"two"}))
	})

	It("adds commands with AddCommands", func() {
		uses := cli.Pipeline(
			cli.PseudoCommand(),
			cli.AddCommands(record("one"), record("two")),
		)
		Expect(run(uses, "app -g one")).To(Succeed())
		Expect(ran).To(Equal([]string{"one"}))
	})

	It("adds commands with an initializer", func() {
		uses := cli.Pipeline(
			cli.PseudoCommand(),
			cli.AddCommand(nil, cli.Prototype{Name: "late", Action: func() { ran = append(ran, "late") }}),
			cli.AddCommand(record("one")),
		)
		Expect(run(uses, "app -g late")).To(Succeed())
		Expect(ran).To(Equal([]string{"late"}))
	})

	It("adds to the commands given to PseudoCommand", func() {
		uses := cli.Pipeline(
			cli.PseudoCommand(record("one"), record("two")),
			cli.AddCommand(record("three")),
		)
		Expect(run(uses, "app -g three")).To(Succeed())
		Expect(ran).To(Equal([]string{"three"}))
	})

	Describe("defaults of the flag", func() {

		It("are copied from the command when only one is added", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(),
				cli.AddCommand(nil, cli.Prototype{
					Name:     "one",
					Aliases:  []string{"o"},
					HelpText: "Run one",
					Action:   func() { ran = append(ran, "one") },
				}),
			)
			Expect(run(uses, "app --one")).To(Succeed())
			Expect(run(uses, "app -o")).To(Succeed())
			Expect(ran).To(Equal([]string{"one", "one"}))
			Expect(run(uses, "app -g one")).NotTo(Succeed())
		})

		It("are copied from the remaining command when others are removed", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(record("one"), record("two")),
				cli.RemoveCommand("one"),
			)
			Expect(run(uses, "app --two")).To(Succeed())
			Expect(ran).To(Equal([]string{"two"}))
		})

		It("are conventional when several commands are added", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(record("one")),
				cli.AddCommand(record("two")),
			)
			app := &cli.App{Name: "app", Flags: []*cli.Flag{{Uses: uses}}}
			ctx, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			f, ok := cli.FromContext(ctx).Command().Flag("global")
			Expect(ok).To(BeTrue())
			Expect(f.Aliases).To(ConsistOf("g"))
			Expect(f.HelpText).To(Equal("Run an operation"))
		})

		It("are not applied over values set by the flag", func() {
			app := &cli.App{Name: "app", Flags: []*cli.Flag{{
				Name:     "operation",
				Aliases:  []string{"o"},
				HelpText: "Mine",
				Uses: cli.Pipeline(
					cli.PseudoCommand(),
					cli.AddCommand(nil, cli.Prototype{Name: "one", HelpText: "Theirs"}),
				),
			}}}
			ctx, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			f, ok := cli.FromContext(ctx).Command().Flag("operation")
			Expect(ok).To(BeTrue())
			Expect(f.Aliases).To(ConsistOf("o"))
			Expect(f.HelpText).To(Equal("Mine"))
		})

		It("run customizations that match the new name", func() {
			app := &cli.App{
				Name: "app",
				Uses: cli.Customize("--one", cli.Data("matched", "yes")),
				Flags: []*cli.Flag{{
					Uses: cli.Pipeline(cli.PseudoCommand(), cli.AddCommand(record("one"))),
				}},
			}
			ctx, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			f, ok := cli.FromContext(ctx).Command().Flag("one")
			Expect(ok).To(BeTrue())
			Expect(f.Data).To(HaveKeyWithValue("matched", "yes"))
		})
	})

	It("does not add to the sub-commands of the command", func() {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{{
				Uses: cli.Pipeline(cli.PseudoCommand(), cli.AddCommand(record("one"))),
			}},
		}
		Expect(app.RunContext(context.Background(), "app", "one")).NotTo(Succeed())
		Expect(ran).To(BeEmpty())

		ctx, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())
		_, ok := cli.FromContext(ctx).Command().Command("one")
		Expect(ok).To(BeFalse())
	})

	It("can use the same action for more than one flag", func() {
		uses := cli.Pipeline(
			cli.PseudoCommand(),
			cli.AddCommand(record("one")),
			cli.AddCommand(record("two")),
		)
		Expect(run(uses, "app -g one")).To(Succeed())
		Expect(run(uses, "app -g two")).To(Succeed())
		Expect(ran).To(Equal([]string{"one", "two"}))
	})

	Describe("RemoveCommand", func() {

		It("removes by name", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(record("one"), record("two")),
				cli.RemoveCommand("one"),
			)
			Expect(run(uses, "app --two")).To(Succeed())
			Expect(ran).To(Equal([]string{"two"}))
		})

		It("removes by name from a selector", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(record("one"), record("two"), record("three")),
				cli.RemoveCommand("one"),
			)
			Expect(run(uses, "app -g two")).To(Succeed())
			Expect(ran).To(Equal([]string{"two"}))

			ran = nil
			Expect(run(uses, "app -g one")).To(MatchError(`"one" is not a command`))
		})

		It("removes by command", func() {
			one := record("one")
			uses := cli.Pipeline(
				cli.PseudoCommand(one, record("two"), record("three")),
				cli.RemoveCommand(one),
			)
			Expect(run(uses, "app -g one")).To(MatchError(`"one" is not a command`))
		})

		It("succeeds when the command doesn't exist", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(),
				cli.AddCommand(record("one")),
				cli.AddCommand(record("two")),
				cli.RemoveCommand("nope"),
			)
			Expect(run(uses, "app -g two")).To(Succeed())
		})
	})

	Describe("internal errors", func() {

		It("when no commands are added", func() {
			err := run(cli.PseudoCommand(), "app")
			Expect(err).To(MatchError(ContainSubstring(`internal error, at "app" (initial timing):`)))
			Expect(err).To(MatchError(ContainSubstring("pseudo command requires at least one command")))
		})

		It("when all commands are removed", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(record("one")),
				cli.RemoveCommand("one"),
			)
			Expect(run(uses, "app")).To(MatchError(ContainSubstring("pseudo command requires at least one command")))
		})

		It("when names are duplicated", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(),
				cli.AddCommand(record("one")),
				cli.AddCommand(record("one")),
			)
			err := run(uses, "app")
			Expect(err).To(MatchError(ContainSubstring(`internal error, at "app" (initial timing):`)))
			Expect(err).To(MatchError(ContainSubstring(`duplicate name used: "one"`)))
		})

		It("when an alias duplicates a name", func() {
			dup := record("two")
			dup.Aliases = []string{"one"}
			uses := cli.Pipeline(
				cli.PseudoCommand(record("one"), record("three")),
				cli.AddCommand(dup),
			)
			Expect(run(uses, "app")).To(MatchError(ContainSubstring(`duplicate name used: "one"`)))
		})

		It("when names are duplicated by commands given to PseudoCommand", func() {
			uses := cli.PseudoCommand(record("one"), record("one"))
			Expect(run(uses, "app")).To(MatchError(ContainSubstring(`duplicate name used: "one"`)))
		})

		It("when a command is unnamed", func() {
			uses := cli.Pipeline(
				cli.PseudoCommand(),
				cli.AddCommand(&cli.Command{}),
				cli.AddCommand(record("one")),
			)
			Expect(run(uses, "app")).To(MatchError(ContainSubstring("must have a name")))
		})
	})
})

var _ = Describe("PseudoCommand", func() {

	Context("when a singleton", func() {

		var (
			app *cli.App

			// Records what happened, as "name arg..." for the operation which ran
			ran       []string
			appAction bool
		)

		run := func(args string) error {
			arguments, _ := cli.Split(args)
			return app.RunContext(context.Background(), arguments...)
		}

		BeforeEach(func() {
			ran = nil
			appAction = false
			app = &cli.App{
				Name: "dns-sd",
				Flags: []*cli.Flag{
					{Name: "verbose", Aliases: []string{"v"}, Value: cli.Bool()},
					{
						Name:    "enumerate",
						Aliases: []string{"E"},
						Uses: cli.PseudoCommand(&cli.Command{
							HelpText: "Enumerate registration domains",
							Action: func(c *cli.Context) error {
								ran = append(ran, "enumerate", c.Name())
								if c.Bool("verbose") {
									ran = append(ran, "verbose")
								}
								return nil
							},
						}),
					},
					{
						Name:    "browse",
						Aliases: []string{"B"},
						Uses: cli.PseudoCommand(&cli.Command{
							HelpText: "Browse for services",
							Flags: []*cli.Flag{
								{Name: "timeout", Aliases: []string{"t"}, Value: cli.Int()},
							},
							Args: []*cli.Arg{
								{Name: "type", NArg: 1},
								{Name: "domain", NArg: 0},
							},
							Action: func(c *cli.Context) error {
								ran = append(ran, "browse", c.String("type"), c.String("domain"))
								if c.Int("timeout") > 0 {
									ran = append(ran, "timeout")
								}
								if c.Bool("verbose") {
									ran = append(ran, "verbose")
								}
								return nil
							},
						}),
					},
				},
				Action: func() { appAction = true },
			}
		})

		Describe("execution", func() {

			DescribeTable("examples",
				func(args string, expected ...string) {
					Expect(run(args)).To(Succeed())
					Expect(ran).To(Equal(expected))
					Expect(appAction).To(BeFalse(), "the action of the command is skipped")
				},
				Entry("short name", "dns-sd -E", "enumerate", "enumerate"),
				Entry("long name", "dns-sd --enumerate", "enumerate", "enumerate"),
				Entry("with args", "dns-sd -B _http._tcp local", "browse", "_http._tcp", "local"),
				Entry("with optional arg omitted", "dns-sd -B _http._tcp", "browse", "_http._tcp", ""),
				Entry("with flags of the pseudo command", "dns-sd -B -t 3 _http._tcp", "browse", "_http._tcp", "", "timeout"),
				Entry("with flags of the pseudo command amid args", "dns-sd -B --timeout=3 _http._tcp local", "browse", "_http._tcp", "local", "timeout"),
				Entry("persistent flag before", "dns-sd -v -E", "enumerate", "enumerate", "verbose"),
				Entry("persistent flag after", "dns-sd -E -v", "enumerate", "enumerate", "verbose"),
				Entry("persistent flag before args", "dns-sd -v -B _http._tcp", "browse", "_http._tcp", "", "verbose"),
				Entry("persistent flag amid args", "dns-sd -B -v _http._tcp", "browse", "_http._tcp", "", "verbose"),
			)

			It("runs the action of the command when no operation is named", func() {
				Expect(run("dns-sd -v")).To(Succeed())
				Expect(ran).To(BeEmpty())
				Expect(appAction).To(BeTrue())
			})

			It("is an error to omit required args of the pseudo command", func() {
				err := run("dns-sd -B")
				Expect(err).To(HaveOccurred())
				Expect(ran).To(BeEmpty())
			})

			It("takes the rest of the line, so another operation can't be named", func() {
				err := run("dns-sd -E -B x")
				Expect(err).To(HaveOccurred())
				Expect(ran).To(BeEmpty())
			})

			It("does not split on commas", func() {
				Expect(run("dns-sd -B a,b")).To(Succeed())
				Expect(ran).To(Equal([]string{"browse", "a,b", ""}))
			})
		})

		Describe("defaults from the pseudo command", func() {

			It("uses the name and aliases of the command", func() {
				var ran bool
				app := &cli.App{
					Name: "app",
					Flags: []*cli.Flag{
						{Uses: cli.PseudoCommand(&cli.Command{
							Name:     "lookup",
							Aliases:  []string{"L"},
							HelpText: "Look up a service",
							Action:   func() { ran = true },
						})},
					},
				}

				Expect(app.RunContext(context.Background(), "app", "-L")).To(Succeed())
				Expect(ran).To(BeTrue())

				ctx, _ := app.Initialize(context.Background())
				f, ok := cli.FromContext(ctx).Command().Flag("lookup")
				Expect(ok).To(BeTrue())
				Expect(f.HelpText).To(Equal("Look up a service"))
				Expect(f.Category).To(Equal("Operations"))
			})

			It("uses the name of the flag when the command has none", func() {
				var ran bool
				app := &cli.App{
					Name: "app",
					Flags: []*cli.Flag{
						{Name: "lookup", Uses: cli.PseudoCommand(&cli.Command{
							Action: func() { ran = true },
						})},
					},
				}

				Expect(app.RunContext(context.Background(), "app", "--lookup")).To(Succeed())
				Expect(ran).To(BeTrue())
			})

			It("panics without a command", func() {
				Expect(func() { cli.PseudoCommand(nil) }).To(Panic())
			})
		})

		Describe("help screen", func() {

			It("lists each operation as an alternative synopsis", func() {
				Expect(renderScreen(app, "dns-sd --help")).To(HavePrefix(
					"usage: dns-sd [--verbose] --enumerate\n" +
						"       dns-sd [--verbose] --browse [--timeout=NUMBER] <type> [<domain>]\n" +
						"       dns-sd {--help | --version}\n"))
			})

			It("lists the operations by category", func() {
				Expect(renderScreen(app, "dns-sd --help")).To(ContainSubstring(
					"Operations:\n" +
						"\n" +
						"  -E, --enumerate  Enumerate registration domains\n" +
						"  -B, --browse     Browse for services\n"))
			})

			Describe("of the pseudo command", func() {

				BeforeEach(func() {
					// Required args prevent the help flag from being used after them
					app.Flags[2].Uses = cli.PseudoCommand(&cli.Command{
						Name:  "browse",
						Flags: []*cli.Flag{{Name: "timeout", Aliases: []string{"t"}, Value: cli.Int()}},
						Args:  []*cli.Arg{{Name: "type"}},
					})
				})

				It("uses the flag to invoke the pseudo command", func() {
					Expect(renderScreen(app, "dns-sd -B --help")).To(HavePrefix(
						"usage: dns-sd --browse [--timeout=NUMBER] [<type>]\n"))
				})

				It("does not list operations as global options", func() {
					screen := renderScreen(app, "dns-sd -B --help")
					Expect(screen).To(ContainSubstring("--verbose"))
					Expect(screen).NotTo(ContainSubstring("--enumerate"))
				})
			})

			It("lists the generic synopsis when the command has args", func() {
				app.Args = []*cli.Arg{{Name: "file"}}

				Expect(renderScreen(app, "dns-sd --help")).To(HavePrefix(
					"usage: dns-sd [--verbose] --enumerate\n" +
						"       dns-sd [--verbose] --browse [--timeout=NUMBER] <type> [<domain>]\n" +
						"       dns-sd [--verbose] [<file>]\n" +
						"       dns-sd {--help | --version}\n"))
			})

			It("lists operations together with sub-commands when expanded", func() {
				app.Options = cli.ExpandSubcommandSynopses
				app.Commands = []*cli.Command{
					{Name: "list", Args: []*cli.Arg{{Name: "type", NArg: 1}}},
				}

				Expect(renderScreen(app, "dns-sd --help")).To(HavePrefix(
					"usage: dns-sd [--verbose] --enumerate\n" +
						"       dns-sd [--verbose] --browse [--timeout=NUMBER] <type> [<domain>]\n" +
						"       dns-sd [--verbose] list <type>\n" +
						"       dns-sd {--help | --version}\n"))
			})
		})

		Describe("Synopsis", func() {

			It("condenses the operations in a single synopsis", func() {
				ctx, _ := app.Initialize(context.Background())
				cmd := cli.FromContext(ctx).Command()

				Expect(cmd.Synopsis()).To(Equal("dns-sd {--help | --version} [--verbose] {--enumerate | --browse}"))
			})
		})

		Describe("completion", func() {

			DescribeTable("examples", func(arguments string, incomplete string, expected ...cli.CompletionItem) {
				ctx, err := app.Initialize(context.Background())
				Expect(err).NotTo(HaveOccurred())

				args, _ := cli.Split(arguments)
				items := cli.FromContext(ctx).Complete(args, incomplete)
				Expect(items).To(ContainElements(expected))
			},
				// The operation is not completed with = because it takes the rest of the line
				Entry("operations", "dns-sd", "--b",
					cli.CompletionItem{Value: "--browse", HelpText: "Browse for services"}),
				Entry("flags of the pseudo command", "dns-sd -B", "--ti",
					cli.CompletionItem{Value: "--timeout=", PreventSpaceAfter: true}),
				Entry("flags of the pseudo command after args", "dns-sd -B _http._tcp", "--ti",
					cli.CompletionItem{Value: "--timeout=", PreventSpaceAfter: true}),
			)

			It("delegates args to the pseudo command", func() {
				app.Flags[2].Uses = cli.PseudoCommand(&cli.Command{
					Name: "browse",
					Args: []*cli.Arg{
						{Name: "type", NArg: 1, Completion: cli.ValueCompletion("_http._tcp", "_ssh._tcp")},
					},
				})
				ctx, err := app.Initialize(context.Background())
				Expect(err).NotTo(HaveOccurred())

				args, _ := cli.Split("dns-sd -B")
				items := cli.FromContext(ctx).Complete(args, "_h")
				Expect(items).To(ContainElement(cli.CompletionItem{Value: "_http._tcp"}))
				Expect(items).NotTo(ContainElement(cli.CompletionItem{Value: "_ssh._tcp"}))
			})
		})

	})

	Context("when more than one", func() {

		var (
			app *cli.App

			// Records what happened, as "name arg..." for the operation which ran
			ran       []string
			appAction bool
		)

		run := func(args string) error {
			arguments, _ := cli.Split(args)
			return app.RunContext(context.Background(), arguments...)
		}

		BeforeEach(func() {
			ran = nil
			appAction = false
			app = &cli.App{
				Name: "dns-sd",
				Flags: []*cli.Flag{
					{Name: "verbose", Aliases: []string{"v"}, Value: cli.Bool()},
					{
						Uses: cli.PseudoCommand(
							&cli.Command{
								Name:     "enumerate",
								Aliases:  []string{"E"},
								HelpText: "Enumerate registration domains",
								Action: func(c *cli.Context) error {
									ran = append(ran, "enumerate")
									if c.Bool("verbose") {
										ran = append(ran, "verbose")
									}
									return nil
								},
							},
							&cli.Command{
								Name:     "browse",
								HelpText: "Browse for services",
								Flags: []*cli.Flag{
									{Name: "timeout", Aliases: []string{"t"}, Value: cli.Int()},
								},
								Args: []*cli.Arg{
									{Name: "type"},
									{Name: "domain"},
								},
								Action: func(c *cli.Context) error {
									ran = append(ran, "browse", c.String("type"), c.String("domain"))
									if c.Int("timeout") > 0 {
										ran = append(ran, "timeout")
									}
									if c.Bool("verbose") {
										ran = append(ran, "verbose")
									}
									return nil
								},
							},
						),
					},
				},
				Action: func() { appAction = true },
			}
		})

		Describe("execution", func() {

			DescribeTable("examples",
				func(args string, expected ...string) {
					Expect(run(args)).To(Succeed())
					Expect(ran).To(Equal(expected))
					Expect(appAction).To(BeFalse(), "the action of the command is skipped")
				},
				Entry("short name", "dns-sd -g enumerate", "enumerate"),
				Entry("long name", "dns-sd --global enumerate", "enumerate"),
				Entry("alias of the command", "dns-sd -g E", "enumerate"),
				Entry("with args", "dns-sd -g browse _http._tcp local", "browse", "_http._tcp", "local"),
				Entry("with flags of the command", "dns-sd -g browse --timeout=3 _http._tcp", "browse", "_http._tcp", "", "timeout"),
				Entry("persistent flag before", "dns-sd -v -g enumerate", "enumerate", "verbose"),
				Entry("persistent flag after the name", "dns-sd -g enumerate -v", "enumerate", "verbose"),
				Entry("persistent flag amid args", "dns-sd -g browse -v _http._tcp", "browse", "_http._tcp", "", "verbose"),
			)

			It("runs the action of the command when no operation is selected", func() {
				Expect(run("dns-sd -v")).To(Succeed())
				Expect(ran).To(BeEmpty())
				Expect(appAction).To(BeTrue())
			})

			It("is an error to name a command that doesn't exist", func() {
				err := run("dns-sd -g nope")
				Expect(err).To(MatchError(`"nope" is not a command`))
				Expect(ran).To(BeEmpty())
			})

			It("is an error to omit the name", func() {
				err := run("dns-sd -g")
				Expect(err).To(MatchError("expected argument"))
				Expect(ran).To(BeEmpty())
			})

			It("can attach the name to the flag only when no args follow", func() {
				Expect(run("dns-sd --global=enumerate")).To(Succeed())
				Expect(ran).To(Equal([]string{"enumerate"}))

				Expect(run("dns-sd --global=browse _http._tcp")).NotTo(Succeed())
			})
		})

		Describe("construction", func() {

			It("uses the name and alias of the conventional flag", func() {
				ctx, _ := app.Initialize(context.Background())
				f, ok := cli.FromContext(ctx).Command().Flag("global")
				Expect(ok).To(BeTrue())
				Expect(f.Aliases).To(ConsistOf("g"))
				Expect(f.Category).To(Equal("Operations"))
			})

			It("uses the name of the flag when the flag has one", func() {
				app.Flags[1].Name = "operation"
				app.Flags[1].Aliases = []string{"o"}

				Expect(run("dns-sd -o enumerate")).To(Succeed())
				Expect(ran).To(Equal([]string{"enumerate"}))
			})

			It("panics with a nil command", func() {
				Expect(func() { cli.PseudoCommand(nil) }).To(Panic())
			})

			It("can provide late initialization of command names", func() {
				var lookedUp bool
				app.Flags = []*cli.Flag{{
					Uses: cli.PseudoCommand(
						&cli.Command{
							Uses:   cli.Name("lookup"),
							Action: func() { lookedUp = true },
						},
						&cli.Command{
							Name: "M",
						}),
				}}

				Expect(run("dns-sd -g lookup")).To(Succeed())
				Expect(lookedUp).To(BeTrue())
			})

			It("can be used along with flags for individual commands", func() {
				var lookedUp bool
				app.Flags = append(app.Flags, &cli.Flag{
					Name:    "lookup",
					Aliases: []string{"L"},
					Uses: cli.PseudoCommand(&cli.Command{
						HelpText: "Look up a service",
						Action:   func() { lookedUp = true },
					}),
				})

				Expect(run("dns-sd -L")).To(Succeed())
				Expect(lookedUp).To(BeTrue())

				Expect(run("dns-sd -g enumerate")).To(Succeed())
				Expect(ran).To(Equal([]string{"enumerate"}))
			})
		})

		Describe("help screen", func() {

			It("lists each command as an alternative synopsis", func() {
				Expect(renderScreen(app, "dns-sd --help")).To(HavePrefix(
					"usage: dns-sd [--verbose] --global enumerate\n" +
						"       dns-sd [--verbose] --global browse [--timeout=NUMBER] [<type>][<domain>]\n" +
						"       dns-sd {--help | --version}\n"))
			})

			It("describes the commands as values of the flag", func() {
				Expect(renderScreen(app, "dns-sd --help")).To(ContainSubstring(
					"Operations:\n" +
						"\n" +
						"  -g, --global {browse|enumerate}  Run an operation\n" +
						"\n" +
						"         enumerate  Enumerate registration domains\n" +
						"         browse     Browse for services\n"))
			})

			It("uses the flag and name to invoke the command", func() {
				Expect(renderScreen(app, "dns-sd -g browse --help")).To(HavePrefix(
					"usage: dns-sd --global browse [--timeout=NUMBER] [<type>][<domain>]\n"))
			})

			It("does not list the flag as a global option of the command", func() {
				Expect(renderScreen(app, "dns-sd -g browse --help")).NotTo(ContainSubstring("--global {"))
			})
		})

		Describe("Synopsis", func() {

			It("condenses the selector in a single synopsis", func() {
				ctx, _ := app.Initialize(context.Background())
				cmd := cli.FromContext(ctx).Command()

				Expect(cmd.Synopsis()).To(Equal("dns-sd {--help | --version} [--verbose] {--global {browse|enumerate}}"))
			})
		})

		Describe("completion", func() {

			DescribeTable("examples", func(arguments string, incomplete string, expected ...cli.CompletionItem) {
				ctx, err := app.Initialize(context.Background())
				Expect(err).NotTo(HaveOccurred())

				args, _ := cli.Split(arguments)
				items := cli.FromContext(ctx).Complete(args, incomplete)
				Expect(items).To(ConsistOf(expected))
			},
				Entry("names", "dns-sd -g", "",
					cli.CompletionItem{Value: "enumerate", HelpText: "Enumerate registration domains"},
					cli.CompletionItem{Value: "E", HelpText: "Enumerate registration domains"},
					cli.CompletionItem{Value: "browse", HelpText: "Browse for services"}),
				Entry("names by prefix", "dns-sd --global", "br",
					cli.CompletionItem{Value: "browse", HelpText: "Browse for services"}),
				Entry("names with no match", "dns-sd -g", "x"),
				Entry("flags of the command", "dns-sd -g browse", "--ti",
					cli.CompletionItem{Value: "--timeout=", PreventSpaceAfter: true}),
				Entry("unknown command", "dns-sd -g nope", "-"),
			)
		})
	})

	Context("validation", func() {
		It("allows flag and arg within in it to see command", func() {
			var (
				flagSeesCommand bool
				argSeesCommand  bool
			)
			hasCommandAccess := func(b *bool) cli.ActionFunc {
				return func(c *cli.Context) error {
					*b = c.Command() != nil && c.Command().Name == "browse"
					return nil
				}
			}

			flagAction := hasCommandAccess(&flagSeesCommand)
			argAction := hasCommandAccess(&argSeesCommand)
			app := &cli.App{
				Name: "dns-sd",
				Flags: []*cli.Flag{
					{
						Name:    "browse",
						Aliases: []string{"B"},
						Uses: cli.PseudoCommand(&cli.Command{
							HelpText: "Browse for services",
							Flags: []*cli.Flag{
								{Name: "timeout", Aliases: []string{"t"}, Value: cli.Int(), Action: flagAction},
							},
							Args: []*cli.Arg{
								{Name: "type", NArg: 1, Action: argAction},
								{Name: "domain", NArg: 0},
							},
							Action: func(c *cli.Context) error {
								return nil
							},
						}),
					},
				},
				Action: func() {},
			}

			arguments, _ := cli.Split("dns-sd -B -t 30 _http._tcp local")
			app.RunContext(context.Background(), arguments...)
			Expect(flagSeesCommand).To(BeTrue())
			Expect(argSeesCommand).To(BeTrue())
		})

	})
})
