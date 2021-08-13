package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
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
	})
})
