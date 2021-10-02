package cli_test

import (
	"os"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Arg", func() {

	Describe("Action", func() {
		var (
			act       *joeclifakes.FakeAction
			app       *cli.App
			arguments = "app f"
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			app = &cli.App{
				Name: "app",
				Args: []*cli.Arg{
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

		It("executes action on setting Arg", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("contains args in captured context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Args()).To(Equal([]string{"f"}))
		})

		It("provides properly initialized context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Name()).To(Equal("<f>"))
			Expect(captured.Path().String()).To(Equal("app <f>"))
		})
	})

	Describe("NArg", func() {

		DescribeTable(
			"inferred type",
			func(count int, expected interface{}) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: count,
						},
					},
					Action: act,
				}
				arguments := "f"
				if count > 0 {
					arguments = strings.Repeat(" g", count)
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(nil, args)
				Expect(err).NotTo(HaveOccurred())

				captured := act.ExecuteArgsForCall(0)
				Expect(captured.LookupArg("f").Value).To(BeAssignableToTypeOf(expected))
			},
			Entry("list when 0", 0, cli.String()),
			Entry("string when 1", 1, cli.String()),
			Entry("list when 2", 2, cli.List()),
			Entry("list when -2", -2, cli.List()),
		)

		DescribeTable(
			"arg parsing",
			func(count int, arguments string, match types.GomegaMatcher) {
				items := []string{}
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name:  "f",
							NArg:  count,
							Value: &items,
						},
					},
					Flags: []*cli.Flag{
						{
							Name:  "f",
							Value: cli.Bool(),
						},
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(nil, args)
				Expect(err).NotTo(HaveOccurred())
				Expect(items).To(match)
			},
			Entry("exactly 1", 1, "one", Equal([]string{"one"})),
			Entry("exactly 3", 3, "one two three", Equal([]string{"one", "two", "three"})),
			Entry("all values even flags", -1, "one -f two -f", Equal([]string{"one", "-f", "two", "-f"})),
			Entry("values stop on flags", -2, "one two -f", Equal([]string{"one", "two"})),
			Entry("optional empty", 0, "", Equal([]string{})),
		)

		DescribeTable(
			"errors",
			func(count int, arguments string, match types.GomegaMatcher) {
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: count,
						},
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(nil, args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(match)
			},
			Entry("missing when 1", 1, "", Equal("expected argument")),
			Entry("too few by 1", 2, "a", Equal("expected 2 arguments")),
			Entry("too many by 1", 1, "a b", Equal(`unexpected argument "b"`)),
		)
	})

	Describe("Synopsis", func() {

		DescribeTable("examples",
			func(f *cli.Arg, expected string) {
				Expect(f.Synopsis()).To(Equal(expected))
			},
			Entry(
				"bool arg",
				&cli.Arg{
					Name:  "arg",
					Value: cli.Bool(),
				},
				"<arg>",
			),
			Entry(
				"repeat arg",
				&cli.Arg{
					Name:  "arg",
					NArg:  -1,
					Value: cli.Bool(),
				},
				"<arg>...",
			),
		)

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
				Args: []*cli.Arg{
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

		Context("when value also set", func() {
			BeforeEach(func() {
				arguments = "app 'option text'"
			})

			It("sets up value from option", func() {
				Expect(actual).To(Equal("option text"))
			})
		})
	})
})
