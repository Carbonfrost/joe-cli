package cli_test

import (
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Arg", func() {

	Describe("Action", func() {
		var (
			act       *joeclifakes.FakeActionHandler
			app       *cli.App
			arguments = "app f"
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeActionHandler)
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

		It("provides properly initialized context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Name()).To(Equal("<f>"))
			Expect(captured.Path().String()).To(Equal("app <f>"))
		})
	})

	DescribeTable(
		"NArg",
		func(count int, expected interface{}) {
			act := new(joeclifakes.FakeActionHandler)
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
			args, _ := cli.Split("app f")
			app.RunContext(nil, args)

			captured := act.ExecuteArgsForCall(0)
			Expect(captured.LookupArg("f").Value).To(BeAssignableToTypeOf(expected))
		},
		Entry("list when 0", 0, cli.String()),
		Entry("string when 1", 1, cli.String()),
		Entry("list when 2", 2, cli.List()),
		Entry("list when -2", -2, cli.List()),
	)

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
