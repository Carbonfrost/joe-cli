package cli_test

import (
	"errors"
	"flag"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Flag", func() {

	Describe("Action", func() {
		var (
			act       *joeclifakes.FakeActionHandler
			app       *cli.App
			arguments = "app -f value"
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeActionHandler)
			app = &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:   "f",
						Action: act,
					},
				},
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split(arguments)
			app.RunContext(nil, args)
		})

		It("executes action on setting flag", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
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

		Context("for a persistent flag", func() {
			BeforeEach(func() {
				act = new(joeclifakes.FakeActionHandler)
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
			app.RunContext(nil, args)
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
			app.RunContext(nil, arguments)
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
			err := app.RunContext(nil, arguments)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("not supported"))
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

var _ flag.Value = new(temperature)
