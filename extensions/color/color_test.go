package color_test

import (
	"bytes"
	"context"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/color"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Features", func() {

	DescribeTable("flag synopses", func(features color.Feature, expected string) {
		var actual string
		app := &cli.App{
			Name: "app",
			Uses: color.Options{
				Features: features,
			},
			Action: func(c *cli.Context) {
				actual = c.Command().Synopsis()
			},
		}
		_ = app.RunContext(context.Background(), []string{"app"})

		actual = strings.ReplaceAll(actual, "{--help | --version}", "")
		actual = strings.ReplaceAll(actual, "  ", " ")
		Expect(actual).To(Equal(expected))
	},
		Entry("boolean flag", color.FlagFeature, "app [--color]"),
		Entry("both boolean flags", color.FlagFeature|color.NoFlagFeature, "app [--[no-]color]"),
		Entry("both flags with mode", color.ModeFeature|color.FlagFeature|color.NoFlagFeature, "app [--color={auto|always|never}] [--no-color]"),
		Entry("mode", color.ModeFeature, "app [--color={auto|always|never}]"),
	)

	DescribeTable("set color", func(arguments string, resetColorCapableCallCount int, setColorCapable types.GomegaMatcher) {
		w := new(joeclifakes.FakeWriter)
		app := &cli.App{
			Name:   "app",
			Stdout: w,
			Uses:   color.Options{},
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())

		calls := make([]bool, w.SetColorCapableCallCount())
		for i := 0; i < w.SetColorCapableCallCount(); i++ {
			calls[i] = w.SetColorCapableArgsForCall(i)
		}
		Expect(w.ResetColorCapableCallCount()).To(Equal(resetColorCapableCallCount))
		Expect(calls).To(setColorCapable)
	},
		Entry("auto", "app --color=auto", 1, BeEmpty()),
		Entry("no value implies always", "app --color", 0, Equal([]bool{true})),
		Entry("always", "app --color=always", 0, Equal([]bool{true})),
		Entry("never", "app --color=never", 0, Equal([]bool{false})),
		Entry("no color", "app --no-color", 0, Equal([]bool{false})),
	)
})

var _ = Describe("Templates", func() {

	Describe("style and color printers", func() {

		DescribeTable("examples", func(tpl string, expected types.GomegaMatcher) {
			app := &cli.App{
				Name: "demo",
				Uses: color.Options{},

				Before: cli.Pipeline(
					cli.SetColor(true),
					cli.RegisterTemplate("custom", tpl),
				),
				Action: cli.RenderTemplate("custom", func(_ *cli.Context) interface{} {
					return struct {
						Data  string
						Int   int
						Items []string
					}{
						Data:  " string ",
						Int:   420,
						Items: []string{"A", "B"},
					}
				}),
			}
			Expect(renderScreen(app, "app")).To(expected)
		},
			Entry("pipe func", "{{ .Data | Bold }}", Equal("\x1b[1m string \x1b[0m")),
			Entry("direct func color", "{{ Red }} {{ .Int }} {{ ResetColor }}", Equal("\x1b[31m 420 \x1b[39m")),
			Entry("direct func style", "{{ Underline }} {{ .Int }} {{ Reset }}", Equal("\x1b[4m 420 \x1b[0m")),
			Entry("empty string", `{{ "" | Underline }}`, Equal("")),
			Entry("Color pipe func", `{{ .Data | Color "Red" }}`, Equal("\x1b[31m string \x1b[39m")),
			Entry("Color direct func", `{{ Color "Red" }} {{ .Int }} {{ ResetColor }}`, Equal("\x1b[31m 420 \x1b[39m")),
			Entry("Background pipe func", `{{ .Data | Background "Red" }}`, Equal("\x1b[41m string \x1b[49m")),
			Entry("Background direct func", `{{ Background "Red" }} {{ .Int }} {{ ResetColor }}`, Equal("\x1b[41m 420 \x1b[39m")),
			Entry("Style pipe func", `{{ .Data | Style "Bold" }}`, Equal("\x1b[1m string \x1b[0m")),
			Entry("Style direct func", `{{ Style "Bold" }} {{ .Int }} {{ ResetColor }}`, Equal("\x1b[1m 420 \x1b[39m")),
			Entry("Multiple styles", `{{ .Data | Style "Bold Underline" }}`, Equal("\x1b[1m\x1b[4m string \x1b[0m")),
			Entry("No styles", `{{ .Data | Style "" }}`, Equal(" string ")),
			Entry("invalid style", `{{ Style "Superscript" }} Style`, Equal("")),
			Entry("invalid styles", `{{ Style "Bold Superscript" }} Style`, Equal("")),
			Entry("empty style", `{{ Style "" }} Style`, Equal(" Style")),
			Entry("BoldFirst", `{{ .Items | BoldFirst | Join ", " }}`, Equal("\x1b[1mA\x1b[0m, B")),
		)

		It("disables color if stdout has no color", func() {
			app := &cli.App{
				Name: "demo",
				Uses: color.Options{},

				Before: cli.Pipeline(
					cli.RegisterTemplate("custom", "{{ .Data | Bold }}"),
				),
				Action: cli.RenderTemplate("custom", func(_ *cli.Context) interface{} {
					return struct{ Data string }{" BOLD TEXT "}
				}),
			}
			Expect(renderScreen(app, "app")).To(Equal(" BOLD TEXT "))
		})
	})
})

func renderScreen(app *cli.App, args string) string {
	arguments, _ := cli.Split(args)
	var buffer bytes.Buffer
	app.Stderr = &buffer
	app.Stdout = &buffer
	_ = app.RunContext(context.TODO(), arguments)
	return buffer.String()
}
