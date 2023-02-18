package cli_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"math/big"
	"net"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
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
			app.RunContext(context.TODO(), args)
		})

		It("executes action on setting flag", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("contains args in captured context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Args()).To(Equal([]string{"-f", "value"}))
		})

		It("provides properly initialized context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Name()).To(Equal("-f"))
			Expect(captured.Path().String()).To(Equal("app -f"))
		})

		It("contains the value in the context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Value("")).To(Equal("value"))
		})

		It("contains the correct Occurrences count", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Occurrences("")).To(Equal(1))
		})

		It("obtains context path", func() {
			captured := act.ExecuteArgsForCall(0)
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
				captured := act.ExecuteArgsForCall(0)
				Expect(captured.Name()).To(Equal("-f"))
				Expect(captured.Path().String()).To(Equal("app -f"))
			})

			It("contains the value in the context", func() {
				captured := act.ExecuteArgsForCall(0)
				Expect(captured.Value("")).To(Equal("value"))
			})

			It("contains the correct Occurrences count", func() {
				captured := act.ExecuteArgsForCall(0)
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
				captured := act.ExecuteArgsForCall(0)
				Expect(captured.Value("")).To(Equal("baz")) // winner due to being last
			})

			It("contains the correct Occurrences count", func() {
				captured := act.ExecuteArgsForCall(0)
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
					captured := act.ExecuteArgsForCall(0)
					Expect(captured.Name()).To(Equal("-f"))
					Expect(captured.Path().String()).To(Equal("app sub -f"))
				})

				It("contains the value in the context", func() {
					captured := act.ExecuteArgsForCall(0)
					Expect(captured.Value("")).To(Equal("value"))
				})
			})
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

		err := app.RunContext(context.TODO(), []string{"app", "--uses"})

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

		err := app.RunContext(context.TODO(), []string{"app"})

		// In particular, we expect --do-not-show to be hidden
		Expect(err).NotTo(HaveOccurred())
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
			app.RunContext(context.TODO(), args)
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
				beforeFlag.ExecuteStub = func(c *cli.Context) error {
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
			app.RunContext(context.TODO(), arguments)
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
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("not supported"))
		})
	})

	Context("when the value is Optional", func() {
		DescribeTable("examples", func(flag interface{}, args string, expected interface{}) {
			var actual interface{}
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

			arguments, _ := cli.Split(args)
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("bool", cli.Bool(), "app -s", true),
			Entry("float32", cli.Float32(), "app -s", float32(1.0)),
			Entry("float64", cli.Float64(), "app -s", float64(1.0)),
			Entry("int", cli.Int(), "app -s", 1),
			Entry("int64", cli.Int64(), "app -s", int64(1)),
			Entry("int32", cli.Int32(), "app -s", int32(1)),
			Entry("int16", cli.Int16(), "app -s", int16(1)),
			Entry("int8", cli.Int8(), "app -s", int8(1)),
			Entry("uint64", cli.UInt64(), "app -s", uint64(1)),
			Entry("uint32", cli.UInt32(), "app -s", uint32(1)),
			Entry("uint16", cli.UInt16(), "app -s", uint16(1)),
			Entry("uint8", cli.UInt8(), "app -s", uint8(1)),
			Entry("IP", cli.IP(), "app -s", net.ParseIP("127.0.0.1")),
			Entry("Duration", cli.Duration(), "app -s", time.Second),

			Entry("long bool", cli.Bool(), "app --show", true),
			Entry("long float32", cli.Float32(), "app --show", float32(1.0)),
			Entry("long float64", cli.Float64(), "app --show", float64(1.0)),
			Entry("long int", cli.Int(), "app --show", 1),
			Entry("long uint64", cli.UInt64(), "app --show", uint64(1)),
			Entry("long IP", cli.IP(), "app --show", net.ParseIP("127.0.0.1")),
			Entry("long Duration", cli.Duration(), "app --show", time.Second),
		)

		DescribeTable("OptionalValue examples", func(flag interface{}, args string, expected interface{}) {
			var actual interface{}
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "show",
						Aliases: []string{"s"},
						Value:   flag,
						Uses:    cli.OptionalValue(expected),
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("show")
				},
			}

			arguments, _ := cli.Split(args)
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("Regexp", cli.Regexp(), "app -s", regexp.MustCompile("hello")),
			Entry("BigInt", cli.BigInt(), "app -s", big.NewInt(2)),
			Entry("BigFloat", cli.BigFloat(), "app -s", big.NewFloat(3)),
			Entry("URL", cli.URL(), "app -s", unwrap(url.Parse("https://hello.example"))),
			Entry("List", cli.List(), "app -s", []string{"OK"}),
			Entry("Map", cli.Map(), "app -s", map[string]string{"A": "B"}),
		)

		It("following is treated as argument", func() {
			var actual, args interface{}
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
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(float64(1.0)))
			Expect(args).To(Equal("following"))
		})

		It("run-in is treated as argument on short option", func() {
			var actual interface{}
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
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(float64(2.0)))
		})

		It("equals is treated as argument on long option", func() {
			var actual interface{}
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
			err := app.RunContext(context.TODO(), arguments)
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
			app.RunContext(context.TODO(), arguments)
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
			_ = app.RunContext(context.TODO(), arguments)
			Expect(*t).To(Equal("tls1.2"))
		})
	})

	Context("when a NonPersistent flag", func() {
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
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("unknown option: --nope"))
			Expect(p).To(Equal(1600)) // unchanged
		})
	})

	Describe("Synopsis", func() {

		DescribeTable("examples",
			func(f *cli.Flag, expected string) {
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
					Data: map[string]interface{}{
						"_Synopsis": cli.NewFlagSynopsis("[no-]reason"),
					},
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
		err := app.RunContext(context.TODO(), args)
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
			"ManualText",
			cli.ManualText("abc"),
			MatchFields(IgnoreExtras, Fields{"ManualText": Equal("abc")}),
		),
		Entry(
			"Description",
			cli.Description("abc"),
			MatchFields(IgnoreExtras, Fields{"Description": Equal("abc")}),
		),
	)
})

type temperature string

func (t *temperature) Set(s string) error {
	switch s {
	case "F":
		*t = "Fahrenheit"
	case "C":
		*t = "Celsius"
	default:
		return errors.New("not supported")
	}
	return nil
}

func (t *temperature) String() string {
	return string(*t)
}

func (*temperature) Synopsis() string {
	return "{Fahrenheit|Celsius}"
}

type boolFlag string

func (*boolFlag) Set(s string) error {
	return nil
}

func (*boolFlag) IsBoolFlag() bool {
	return true
}

func (*boolFlag) String() string {
	return ""
}

var _ flag.Value = (*temperature)(nil)
var _ flag.Value = (*boolFlag)(nil)
