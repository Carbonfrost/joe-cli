package cli_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Complete", func() {
	DescribeTable("examples", func(arguments string, incomplete string, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "app",

			// No-op actions are set in order to prevent displaying the help screen.
			// The call to ctx.Complete below triggers the flow for completion
			// but does not handle all of the pipeline actions which would prevent
			// the help screen from being displayed
			Commands: []*cli.Command{
				{
					Name:     "sub",
					HelpText: "sub help text",
					Subcommands: []*cli.Command{
						{Name: "grand"},
					},
					Action: func() {},
				},
				{Name: "par"},
			},
			Args: []*cli.Arg{
				{Name: "a", Value: new(cli.File), NArg: 1},
			},
			Flags: []*cli.Flag{
				{Name: "flag", Aliases: []string{"f"}, Value: new(bool)},
				{Name: "long", Aliases: []string{"l"}, HelpText: "has help text", Completion: cli.CompletionValues("a")},
			},
			Action: func() {},
		}

		args, _ := cli.Split(arguments)
		ctx, err := app.Initialize(context.TODO())
		Expect(err).NotTo(HaveOccurred())
		Expect(cli.FromContext(ctx).Complete(args, incomplete)).To(expected)
	},
		Entry("no matches", "app", "--fr", WithTransform(ignoringDefaults, BeEmpty())),
		Entry("all options", "app", "-", WithTransform(ignoringDefaults, ConsistOf([]cli.CompletionItem{
			{Value: "--flag"},
			{Value: "--long", HelpText: "has help text"},
			{Value: "-l", HelpText: "has help text"},
			{Value: "-f"},
		}))),

		// For flags with values, only the name is suggested in the case multiple are available
		Entry("long options", "app", "--", WithTransform(ignoringDefaults, ConsistOf([]cli.CompletionItem{
			{Value: "--flag"},
			{Value: "--long", HelpText: "has help text"}, // Only the name is suggested
		}))),

		// For flags with values, if it is the only remaining match, append the equal sign.
		Entry("long option with value", "app", "--l", WithTransform(ignoringDefaults, ConsistOf([]cli.CompletionItem{
			{Value: "--long=", HelpText: "has help text", PreventSpaceAfter: true},
		}))),

		Entry("long option space after", "app --long", "", WithTransform(ignoringDefaults, ConsistOf([]cli.CompletionItem{
			{Value: "a"},
		}))),

		Entry("long option space after", "app --long", "a", WithTransform(ignoringDefaults, ConsistOf([]cli.CompletionItem{
			{Value: "a"},
		}))),

		Entry("long option (partial)", "app", "--f", WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "--flag"},
		}))),
		Entry("file arg delegate", "app", "", Equal([]cli.CompletionItem{
			{Value: "", Type: cli.FileCompletionType},
		})),
		Entry("sub-command", "app file_specified", "", ContainElements([]cli.CompletionItem{
			{Value: "sub", HelpText: "sub help text"},
			{Value: "par"},
		})),
		Entry("sub-command (partial)", "app file_specified", "su", Equal([]cli.CompletionItem{
			{Value: "sub", HelpText: "sub help text"},
		})),
		Entry("sub-sub-command", "app file_specified sub", "", Equal([]cli.CompletionItem{
			{Value: "grand"},
		})),
		Entry("ignore incorrect arg name", "app --something", "--f", WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "--flag"},
		}))),
	)

	var (
		rosesAndViolets = cli.CompletionValues("roses", "violets")
	)

	DescribeTable("flag examples", func(arguments string, incomplete string, completion cli.Completion, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Name: "flag", Aliases: []string{"f"}, Completion: completion, Value: new(string)},
			},
			Action: func() {},
		}

		args, _ := cli.Split(arguments)
		ctx, err := app.Initialize(context.TODO())
		Expect(err).NotTo(HaveOccurred())
		Expect(cli.FromContext(ctx).Complete(args, incomplete)).To(expected)
	},
		Entry("completion all values", "app", "--flag", rosesAndViolets, WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "--flag=roses"},
			{Value: "--flag=violets"},
		}))),
		Entry("completion value prefix", "app", "--flag=r", rosesAndViolets, WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "--flag=roses"},
		}))),
		Entry("completion value space", "app --flag", "", rosesAndViolets, WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "roses"},
			{Value: "violets"},
		}))),

		Entry("short all values", "app -f", "", rosesAndViolets, WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "roses"},
			{Value: "violets"},
		}))),
		Entry("short value prefix", "app -f", "r", rosesAndViolets, WithTransform(ignoringDefaults, Equal([]cli.CompletionItem{
			{Value: "roses"},
		}))),
	)
})

func ignoringDefaults(v interface{}) interface{} {
	// Remove --help and --version to simplify test
	c := v.([]cli.CompletionItem)
	res := make([]cli.CompletionItem, 0, len(c))
	for _, item := range c {
		if item.Value == "--help" || item.Value == "-h" || item.Value == "--version" {
			continue
		}
		res = append(res, item)
	}
	return res
}
