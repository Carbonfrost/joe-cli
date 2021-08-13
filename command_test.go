package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {

	Describe("actions", func() {

		var (
			act       *joeclifakes.FakeActionHandler
			beforeAct *joeclifakes.FakeActionHandler

			app *cli.App
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeActionHandler)
			beforeAct = new(joeclifakes.FakeActionHandler)

			app = &cli.App{
				Commands: []*cli.Command{
					{
						Name:   "c",
						Action: act,
						Before: beforeAct,
					},
				},
			}

			args, _ := cli.Split("app c")
			app.RunContext(nil, args)
		})

		It("executes action on executing sub-command", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("executes before action on executing sub-command", func() {
			Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
		})
	})

})
