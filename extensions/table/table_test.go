package table_test

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/table"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("template functions", func() {

	DescribeTable("examples", func(templ string, expected types.GomegaMatcher) {
		app := &cli.App{
			Name:   "demo",
			Uses:   table.Options{},
			Before: cli.RegisterTemplate("myTable", templ),
			Action: cli.RenderTemplate("myTable", func(_ *cli.Context) interface{} {
				return struct{ Data int }{1}
			}),
		}
		Expect(renderScreen(app, "app")).To(expected)
	},
		Entry(
			"porcelain",
			`{{- Table "Porcelain" -}}
				{{- Headers "First" "Last" -}}
				{{- Row -}}
				{{- Cell "George" -}}
				{{- Cell "Burdell" -}}
			 {{- EndTable -}}`,
			Equal("First\tLast\nGeorge\tBurdell\n"),
		),
	)

	DescribeTable("errors", func(templ string, expected types.GomegaMatcher) {
		app := &cli.App{
			Name:   "demo",
			Uses:   table.Options{},
			Before: cli.RegisterTemplate("myTable", templ),
			Action: cli.RenderTemplate("myTable", func(_ *cli.Context) interface{} {
				return struct{ Data int }{1}
			}),
			Stderr: io.Discard,
			Stdout: io.Discard,
		}
		err := app.RunContext(context.TODO(), []string{"app"})
		Expect(err).To(expected)
	},
		Entry("panics on Table with args", `{{ Table "a" "b" }}`, MatchError(ContainSubstring(`expects 0 or 1 arguments`))),
	)
})

func renderScreen(app *cli.App, args string) string {
	defer disableConsoleColor()()

	arguments, _ := cli.Split(args)
	var buffer bytes.Buffer
	app.Stderr = &buffer
	app.Stdout = &buffer
	err := app.RunContext(context.TODO(), arguments)
	Expect(err).NotTo(HaveOccurred())
	return buffer.String()
}

func disableConsoleColor() func() {
	os.Setenv("TERM", "dumb")
	return func() {
		os.Setenv("TERM", "0")
	}
}
