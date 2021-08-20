package cli_test

import (
	"bytes"

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
		renderHelpScreen = func(app *cli.App) string {
			var buffer bytes.Buffer
			app.Stderr = &buffer
			_ = app.RunContext(nil, []string{"app", "--help"})
			return buffer.String()
		}
	)

	DescribeTable("examples",
		func(app *cli.App, expected types.GomegaMatcher) {
			Expect(renderHelpScreen(app)).To(expected)
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
	)

})
