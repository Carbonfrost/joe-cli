package cli_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("middleware", func() {

	var (
		captured *cli.Context
		before   cli.ActionHandler
	)
	JustBeforeEach(func() {
		act := new(joeclifakes.FakeActionHandler)
		app := &cli.App{
			Name:   "app",
			Before: before,
			Action: act,
		}
		app.RunContext(context.TODO(), []string{"app"})
		captured = act.ExecuteArgsForCall(0)
	})

	Context("ContextValue", func() {
		BeforeEach(func() {
			before = cli.ContextValue("mykey", "context value")
		})

		It("ContextValue can set and retrieve context value", func() {
			Expect(captured.Context.Value("mykey")).To(BeIdenticalTo("context value"))
		})

	})

})
