package cli_test

import (
	"bytes"
	"context"
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

var _ = Describe("RenderTemplate", func() {
	It("uses the template and custom funcs", func() {
		app := &cli.App{
			Name: "demo",
			Before: cli.Pipeline(
				cli.RegisterTemplateFunc("CustomFunc", func() string {
					return "customFunc result"
				}),
				cli.RegisterTemplate("custom", "template {{ CustomFunc }} {{ .Data }}"),
			),
			Action: cli.RenderTemplate("custom", func(_ *cli.Context) interface{} {
				return struct{ Data int }{1}
			}),
		}
		Expect(renderScreen(app, "app")).To(ContainSubstring("template customFunc result 1"))
	})
})

var _ = Describe("PrintVersion", func() {
	It("uses the version template", func() {
		app := &cli.App{
			Name:    "demo",
			Version: "hello.5.0",
			Before:  cli.RegisterTemplate("version", "custom template {{ .App.Name }} {{ .App.Version }}"),
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderScreen(app, "app --version")).To(ContainSubstring("custom template demo hello.5.0"))
	})
})

var _ = Describe("DisplayHelpScreen", func() {
	It("is the default action for an app with sub-commands", func() {
		app := &cli.App{
			Name: "demo",
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderScreen(app, "demo")).To(ContainSubstring("usage: demo"))
	})

	It("uses the help template", func() {
		app := &cli.App{
			Name:   "demo",
			Before: cli.RegisterTemplate("help", "custom help template"),
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
		}
		Expect(renderScreen(app, "app help")).To(ContainSubstring("custom help template"))
	})

	DescribeTable("examples",
		func(app *cli.App, expected types.GomegaMatcher) {
			Expect(renderScreen(app, "app --help")).To(expected)
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
		Entry("display sub-command",
			&cli.App{
				Commands: []*cli.Command{
					{
						Name: "ok",
					},
				},
			},
			ContainSubstring("<command> [<args>]")),
		Entry("display expression",
			&cli.App{
				Exprs: []*cli.Expr{
					{
						Name: "expr",
					},
				},
			},
			ContainSubstring("<expression>...")),
	)

	DescribeTable("sub-command examples",
		func(app *cli.App, args string, expected types.GomegaMatcher) {
			Expect(renderScreen(app, args)).To(expected)
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
		Entry("display expression",
			&cli.App{
				Commands: []*cli.Command{
					{
						Name: "sub",

						Exprs: []*cli.Expr{
							{
								Name: "expr",
							},
						},
					},
				},
			},
			"app help sub",
			ContainSubstring("<expression>...")),
	)

})

func disableConsoleColor() func() {
	os.Setenv("NO_COLOR", "1")
	return func() {
		os.Setenv("NO_COLOR", "0")
	}
}

func renderScreen(app *cli.App, args string) string {
	defer disableConsoleColor()()

	arguments, _ := cli.Split(args)
	var buffer bytes.Buffer
	app.Stderr = &buffer
	app.Stdout = &buffer
	_ = app.RunContext(context.TODO(), arguments)
	return buffer.String()
}
