package cli_test

import (
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
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

			Context("set within the subcommand", func() {
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

	Context("when environment variables are set", func() {
		var (
			actual    string
			arguments string
		)

		BeforeEach(func() {
			arguments = "app "
		})

		JustBeforeEach(func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "f",
						EnvVars: []string{"_GOCLI_F"},
						Value:   &actual,
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
			Expect(err.Error()).To(Equal("not supported"))
		})
	})

	Context("when the value is Optional", func() {
		DescribeTable("examples", func(flag interface{}, expected interface{}) {
			var actual interface{}
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "s",
						Value:   flag,
						Options: cli.Optional,
					},
				},
				Action: func(c *cli.Context) {
					actual = c.Value("s")
				},
			}

			arguments, _ := cli.Split("app -s")
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("bool", cli.Bool(), true),
			Entry("float32", cli.Float32(), float32(1.0)),
			Entry("float64", cli.Float64(), float64(1.0)),
			Entry("int", cli.Int(), 1),
			Entry("uint64", cli.UInt64(), uint64(1)),
			Entry("IP", cli.IP(), net.ParseIP("127.0.0.1")),
			Entry("Duration", cli.Duration(), time.Second),
		)
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
		)

	})

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

var _ flag.Value = new(temperature)
