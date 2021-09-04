package cli_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("middleware", func() {

	var (
		captured *cli.Context
		before   cli.ActionHandler
		flags    []*cli.Flag
	)
	JustBeforeEach(func() {
		act := new(joeclifakes.FakeActionHandler)
		app := &cli.App{
			Name:   "app",
			Before: before,
			Action: act,
			Flags:  flags,
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

	Context("SetValue", func() {
		BeforeEach(func() {
			flags = []*cli.Flag{
				{
					Name:   "int",
					Value:  cli.Int(),
					Before: cli.SetValue(420),
				},
			}
		})

		It("can set and retrieve value", func() {
			Expect(captured.Value("int")).To(Equal(420))
		})
	})

})

var _ = Describe("events", func() {
	DescribeTable("execution order of events",
		func(arguments string, expected types.GomegaMatcher) {
			result := make([]string, 0)
			event := func(name string) cli.ActionHandler {
				return cli.Action(func() {
					result = append(result, name)
				})
			}
			app := &cli.App{
				Before: event("before app"),
				Action: event("app"),
				After:  event("after app"),
				Flags: []*cli.Flag{
					{
						Name:   "global",
						Value:  cli.Bool(),
						Before: event("before --global"),
						Action: event("--global"),
						After:  event("after --global"),
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Before: event("before sub"),
						Action: event("sub"),
						After:  event("after sub"),
						Flags: []*cli.Flag{
							{
								Name:   "local",
								Value:  cli.Bool(),
								Before: event("before --local"),
								Action: event("--local"),
								After:  event("after --local"),
							},
						},
						Subcommands: []*cli.Command{
							{
								Name:   "dom",
								Before: event("before dom"),
								After:  event("after dom"),
								Action: event("dom"),
								Flags: []*cli.Flag{
									{
										Name:   "nest",
										Value:  cli.Bool(),
										Before: event("before --nest"),
										Action: event("--nest"),
									},
								},
								Args: []*cli.Arg{
									{
										Name:   "a",
										Before: event("before a"),
										Action: event("a"),
									},
								},
							},
						},
					},
				},
			}
			args, _ := cli.Split(arguments)
			err := app.RunContext(nil, args)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(expected)
		},
		Entry(
			"persistent flags always run before subcommand flags",
			"app sub --local --global", // despite being used after, before --global is run first
			ContainElements("before --global", "before --local"),
		),
		Entry(
			"sub-command call",
			"app sub",
			And(
				ContainElements("before app", "before sub"),
				ContainElements("after sub", "after app"),
			),
		),
		Entry(
			"nested command persistent flag is called",
			"app sub --global ",
			ContainElements("before --global", "--global"),
		),
	)
})
