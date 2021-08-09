package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Action", func() {

	// TODO Detect flag being set and invoke action
	XIt("executes action on setting flag", func() {
		act := new(joeclifakes.FakeActionHandler)
		app := &cli.App{
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:   "f",
					Action: act,
				},
			},
		}

		args, _ := cli.Split("app -f value")
		app.RunContext(nil, args)
		Expect(act.ExecuteCallCount()).To(Equal(1))
	})

})
