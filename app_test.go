package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("App", func() {

	Describe("actions", func() {

		var (
			act       *joeclifakes.FakeActionHandler
			beforeAct *joeclifakes.FakeActionHandler
			arguments string

			app *cli.App
		)

		JustBeforeEach(func() {
			act = new(joeclifakes.FakeActionHandler)
			beforeAct = new(joeclifakes.FakeActionHandler)

			app = &cli.App{
				Commands: []*cli.Command{
					{
						Name: "c",
					},
				},
				Flags: []*cli.Flag{
					{
						Name: "f",
					},
				},
				Action: act,
				Before: beforeAct,
			}

			args, _ := cli.Split(arguments)
			app.RunContext(nil, args)
		})

		Context("when executing a sub-command", func() {
			BeforeEach(func() {
				arguments = "cli c"
			})

			It("does not execute action on executing sub-command", func() {
				Expect(act.ExecuteCallCount()).To(Equal(0))
			})

			It("executes before action on executing sub-command", func() {
				Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
			})
		})

		Context("when setting a flag", func() {
			BeforeEach(func() {
				arguments = "cli"
			})
			It("executes before action on executing self", func() {
				Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
			})
		})

	})

})
