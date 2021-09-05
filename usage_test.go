package cli_test

import (
	"bytes"
	"os"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("usage", func() {

	Describe("parse", func() {

		DescribeTable("extract placeholders",
			func(text string, expected []string) {
				Expect(cli.ParseUsage(text).Placeholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", []string{}),
			Entry("placeholder", "{PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("placeholder used twice", "{PLACEHOLDER} {PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("2 placeholders", "{A} {B}", []string{"A", "B"}),
			Entry("2 placeholders with indexes", "{1:A} {0:B}", []string{"B", "A"}),
		)

		DescribeTable("without placeholders text",
			func(text string, expected string) {
				Expect(cli.ParseUsage(text).WithoutPlaceholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", "Literal text"),
			Entry("placeholder", "Load configuration from {FILE}s", "Load configuration from FILEs"),
		)
	})
})

var _ = Describe("DisplayHelpScreen", func() {
	var (
		renderHelpScreen = func(app *cli.App, args string) string {
			defer disableConsoleColor()()

			arguments, _ := cli.Split(args)
			var buffer bytes.Buffer
			app.Stderr = &buffer
			_ = app.RunContext(nil, arguments)
			return buffer.String()
		}
	)

	It("is the default action for an app with sub-commands", func() {
		app := &cli.App{
			Name: "demo",
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderHelpScreen(app, "demo")).To(ContainSubstring("usage: demo"))
	})

	DescribeTable("examples",
		func(app *cli.App, expected types.GomegaMatcher) {
			Expect(renderHelpScreen(app, "app --help")).To(expected)
		},
		Entry("shows normal flags",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name: "normal",
					},
				},
			},
			ContainSubstring("--normal")),
		Entry("replace usage placeholders",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name:     "normal",
						HelpText: "Loads configuration from {FILE}s",
					},
				},
			},
			ContainSubstring("Loads configuration from FILEs")),
		Entry("does not show hidden flags",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "hidden",
						Options: cli.Hidden,
					},
				},
			},
			Not(ContainSubstring("--hidden"))),
		Entry("display action-like flags",
			&cli.App{},
			ContainSubstring("{--help | --version}")),
	)

	DescribeTable("sub-command examples",
		func(app *cli.App, args string, expected types.GomegaMatcher) {
			Expect(renderHelpScreen(app, args)).To(expected)
		},
		Entry("shows sub-command using help switch",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			},
			"app --help sub",
			ContainSubstring("usage: app sub")),
		Entry("show sub-command using help command",
			&cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			},
			"app help sub",
			ContainSubstring("usage: app sub")),
	)

})

func disableConsoleColor() func() {
	os.Setenv("NO_COLOR", "1")
	return func() {
		os.Setenv("NO_COLOR", "0")
	}
}
