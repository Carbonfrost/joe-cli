package cli_test

import (
	"context"
	"github.com/Carbonfrost/joe-cli"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("IsMatch", func() {

	var (
		timingStrings = map[cli.Timing]string{
			cli.InitialTiming: "i",
			cli.BeforeTiming:  "b",
			cli.ActionTiming:  "c",
			cli.AfterTiming:   "a",
		}

		res []string

		appendName cli.ActionFunc = func(c *cli.Context) error {
			res = append(res, c.Name())
			return nil
		}

		appendTiming cli.ActionFunc = func(c *cli.Context) error {
			res = append(res, timingStrings[c.Timing()])
			return nil
		}

		targetApp = func(mode cli.ContextFilter) (string, *cli.App) {
			return "p c -f a", &cli.App{
				Name: "p",
				Commands: []*cli.Command{
					{
						Name:   "c",
						Before: cli.IfMatch(mode, appendName),
						Flags: []*cli.Flag{
							{
								Name:   "f",
								Value:  new(bool),
								Before: cli.IfMatch(mode, appendName),
							},
						},
						Args: []*cli.Arg{
							{
								Name:   "a",
								Before: cli.IfMatch(mode, appendName),
							},
						},
					},
				},
				Uses: cli.IfMatch(mode, appendName),
			}
		}

		timingApp = func(mode cli.ContextFilter) (string, *cli.App) {
			return "p", &cli.App{
				Uses:   cli.IfMatch(mode, appendTiming),
				Before: cli.IfMatch(mode, appendTiming),
				After:  cli.IfMatch(mode, appendTiming),
				Action: cli.IfMatch(mode, appendTiming),
			}
		}
	)

	JustBeforeEach(func() {
		res = nil
	})

	DescribeTable("examples", func(createApp func(cli.ContextFilter) (string, *cli.App), m cli.ContextFilter, expected types.GomegaMatcher) {
		arguments, app := createApp(m)

		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(expected)
	},
		Entry("AnyFlag", targetApp, cli.AnyFlag, Equal([]string{"-f"})),
		Entry("AnyArg", targetApp, cli.AnyArg, Equal([]string{"<a>"})),
		Entry("Anything", targetApp, cli.Anything, ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("HasValue", targetApp, cli.HasValue, Equal([]string{"-f", "<a>"})),
		Entry("RootCommand", targetApp, cli.RootCommand, Equal([]string{"p"})),
		Entry("Seen", targetApp, cli.Seen, ConsistOf([]string{"-f", "<a>"})),
		Entry("Initial", timingApp, cli.InitialTiming, Equal([]string{"i"})),
		Entry("Before", timingApp, cli.BeforeTiming, Equal([]string{"b"})),
		Entry("After", timingApp, cli.AfterTiming, Equal([]string{"a"})),
		Entry("Action", timingApp, cli.ActionTiming, Equal([]string{"c"})),
		Entry("combination", targetApp, cli.AnyFlag|cli.Seen, Equal([]string{"-f"})),
		Entry("nil matches everything", targetApp, nil, ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("thunk", targetApp, cli.ContextFilterFunc(func(c *cli.Context) bool { return false }), BeEmpty()),
		Entry("nil thunk matches everything", targetApp, cli.ContextFilterFunc(nil), ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("pattern", targetApp, cli.PatternFilter("c -f"), Equal([]string{"-f"})),
		Entry("empty matches everything", targetApp, cli.PatternFilter(""), Equal([]string{"p", "c", "-f", "<a>"})),
	)
})
