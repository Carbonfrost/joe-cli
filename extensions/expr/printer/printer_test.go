// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	"github.com/Carbonfrost/joe-cli/extensions/expr/printer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/spf13/afero"
)

type item struct {
	Name string
	Size int
}

func (i item) String() string {
	return i.Name
}

var _ = Describe("Printer", func() {

	var (
		testFileSystem cli.FS
		stdout         *bytes.Buffer

		// newApp creates the minimal app which pushes the given values into
		// the expression pipeline
		newApp = func(p *printer.Printer, values ...any) *cli.App {
			stdout = new(bytes.Buffer)
			testFileSystem = wrapperFS{afero.NewMemMapFs()}

			return &cli.App{
				Name:   "app",
				Uses:   p,
				FS:     testFileSystem,
				Stdout: stdout,
				Args: []*cli.Arg{
					{
						Name: "path",
						NArg: 1,
					},
					{
						Name:  "expression",
						Value: new(expr.Expression),
					},
				},
				Action: func(c *cli.Context) error {
					return expr.FromContext(c, "expression").Evaluate(c, values...)
				},
			}
		}

		// run invokes the app with the given expression, which follows the
		// path arg as in "app . -print"
		run = func(app *cli.App, expression string) error {
			arguments, _ := cli.Split("app . " + expression)
			return app.RunContext(context.Background(), arguments)
		}

		contentsOf = func(name string) string {
			actual, err := fs.ReadFile(testFileSystem, name)
			Expect(err).NotTo(HaveOccurred())
			return string(actual)
		}
	)

	Describe("expression operators", func() {

		DescribeTable("standard output examples", func(args string, value any, expected string) {
			app := newApp(printer.New(), value)
			Expect(run(app, args)).NotTo(HaveOccurred())
			Expect(stdout.String()).To(Equal(expected))
		},
			Entry("print", "-print", "value", "value\n"),
			Entry("print0", "-print0", "value", "value\x00"),
			Entry("print uses default formatting", "-print", item{Name: "f", Size: 2}, "f\n"),
			Entry("printf", "-printf %(name)", item{Name: "f", Size: 2}, "f"),
			Entry("printf with format", "-printf '%(size:03d)'", item{Name: "f", Size: 2}, "002"),
			Entry("printf with literal", "-printf '%(name)=%(size)%(newline)'", item{Name: "f", Size: 2}, "f=2\n"),
			Entry("printf expands time values", "-printf %(year)-%(month)",
				time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC), "2026-July"),
		)

		DescribeTable("file examples", func(args string, value any, expected string) {
			app := newApp(printer.New(), value)
			Expect(run(app, args)).NotTo(HaveOccurred())
			Expect(contentsOf("out.txt")).To(Equal(expected))
			Expect(stdout.String()).To(BeEmpty())
		},
			Entry("fprint", "-fprint out.txt", "value", "value\n"),
			Entry("fprint0", "-fprint0 out.txt", "value", "value\x00"),
			Entry("fprintf", "-fprintf out.txt %(name)", item{Name: "f", Size: 2}, "f"),
			Entry("fprintf with format", "-fprintf out.txt '%(size:03d)'", item{Name: "f"}, "000"),
		)

		It("yields the value to the next operator in the pipeline", func() {
			var actual []any
			app := newApp(printer.New(), "first", "second")
			app.Args[1].Uses = expr.AddExpr(&expr.Expr{
				Name: "capture",
				Evaluate: func(v any) {
					actual = append(actual, v)
				},
			})

			Expect(run(app, "-print -capture")).NotTo(HaveOccurred())
			Expect(actual).To(Equal([]any{"first", "second"}))
			Expect(stdout.String()).To(Equal("first\nsecond\n"))
		})

		It("appends to the file when it is named again", func() {
			app := newApp(printer.New(), "first", "second")
			Expect(run(app, "-fprint out.txt -fprint out.txt")).NotTo(HaveOccurred())
			Expect(contentsOf("out.txt")).To(Equal("first\nfirst\nsecond\nsecond\n"))
		})

		It("truncates the file when it is first named", func() {
			app := newApp(printer.New(), "value")
			existing, err := testFileSystem.Create("out.txt")
			Expect(err).NotTo(HaveOccurred())
			existing.(io.Writer).Write([]byte("previous contents"))
			existing.Close()

			Expect(run(app, "-fprint out.txt")).NotTo(HaveOccurred())
			Expect(contentsOf("out.txt")).To(Equal("value\n"))
		})

		It("reports an error when the file cannot be opened", func() {
			app := newApp(printer.New(printer.WithFS(readOnlyFS{})), "value")
			Expect(run(app, "-fprint out.txt")).To(MatchError(fs.ErrPermission))
		})

		It("provides the synopsis of each operator", func() {
			app := newApp(printer.New())
			_, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			Expect(synopsesOf(app)).To(Equal([]string{
				"-fprint FILE",
				"-fprint0 FILE",
				"-fprintf FILE PATTERN",
				"-print",
				"-print0",
				"-printf PATTERN",
			}))
		})

		It("annotates each operator with the source of this package", func() {
			app := newApp(printer.New())
			_, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			name, value := printer.SourceAnnotation()
			Expect(name).To(Equal("Source"))
			Expect(value).To(Equal("github.com/Carbonfrost/joe-cli/extensions/expr/printer"))

			for _, e := range exprsOf(app) {
				actual, ok := e.LookupData(name)
				Expect(ok).To(BeTrue(), "expected %s to be annotated", e.Name)
				Expect(actual).To(Equal(value), e.Name)
			}
		})
	})

	Describe("AddExprs", func() {

		It("adds the operators to the arg which defines the expression", func() {
			// Only register the printer as a context service from the app so
			// that the operators come from the arg itself
			p := printer.New()
			p.Apply(printer.WithAction(printer.ContextValue(p)))

			stdout = new(bytes.Buffer)
			app := &cli.App{
				Name:   "app",
				Uses:   p,
				Stdout: stdout,
				Args: []*cli.Arg{
					{
						Name: "path",
						NArg: 1,
					},
					{
						Name:  "expression",
						Value: new(expr.Expression),
						Uses:  printer.AddExprs(),
					},
				},
				Action: func(c *cli.Context) error {
					return expr.FromContext(c, "expression").Evaluate(c, "value")
				},
			}

			Expect(run(app, "-print")).NotTo(HaveOccurred())
			Expect(stdout.String()).To(Equal("value\n"))
		})

		It("adds the operators to the expression of a subcommand", func() {
			stdout = new(bytes.Buffer)
			app := &cli.App{
				Name:   "app",
				Uses:   printer.New(),
				Stdout: stdout,
				Commands: []*cli.Command{
					{
						Name: "sub",
						Args: []*cli.Arg{
							{
								Name: "path",
								NArg: 1,
							},
							{
								Name:  "expression",
								Value: new(expr.Expression),
							},
						},
						Action: func(c *cli.Context) error {
							return expr.FromContext(c, "expression").Evaluate(c, "value")
						},
					},
				},
			}

			arguments, _ := cli.Split("app sub . -print")
			Expect(app.RunContext(context.Background(), arguments)).NotTo(HaveOccurred())
			Expect(stdout.String()).To(Equal("value\n"))
		})
	})

	Describe("options", func() {

		It("WithPatternOptions controls how patterns are parsed", func() {
			app := newApp(
				printer.New(printer.WithPatternOptions(expander.WithDelimiters("${", "}"))),
				item{Name: "f"},
			)

			Expect(run(app, "-printf '${name} %(name)'")).NotTo(HaveOccurred())
			Expect(stdout.String()).To(Equal("f %(name)"))
		})

		It("WithExpanderFactory controls how values are expanded", func() {
			factory := func(_ context.Context, v any) expander.Interface {
				return expander.Map{"custom": v}
			}
			app := newApp(printer.New(printer.WithExpanderFactory(factory)), "value")

			Expect(run(app, "-printf %(custom)")).NotTo(HaveOccurred())
			Expect(stdout.String()).To(Equal("value"))
		})

		It("WithFS controls how file names are interpreted", func() {
			other := wrapperFS{afero.NewMemMapFs()}
			app := newApp(printer.New(printer.WithFS(other)), "value")

			Expect(run(app, "-fprint out.txt")).NotTo(HaveOccurred())

			actual, err := fs.ReadFile(other, "out.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal("value\n"))

			_, err = fs.ReadFile(testFileSystem, "out.txt")
			Expect(err).To(HaveOccurred())
		})

		It("WithAction replaces the default action", func() {
			app := newApp(printer.New(printer.WithAction(nil)), "value")
			Expect(run(app, "-print")).To(MatchError(ContainSubstring("unknown expression")))
		})
	})

	Describe("Printer", func() {

		Describe("Compile", func() {

			It("caches compiled patterns", func() {
				p := printer.New()
				Expect(p.Compile("%(name)")).To(BeIdenticalTo(p.Compile("%(name)")))
			})

			It("uses the pattern options", func() {
				p := printer.New(printer.WithPatternOptions(expander.WithDelimiters("${", "}")))
				Expect(p.Compile("${name}").String()).To(Equal("${name}"))
			})
		})

		Describe("Expander", func() {

			DescribeTable("examples", func(value any, key string, expected types.GomegaMatcher) {
				p := printer.New()
				Expect(p.Expander(context.Background(), value).Expand(key)).To(expected)
			},
				Entry("nil", nil, "any", BeNil()),
				Entry("expander", expander.Map{"key": "v"}, "key", Equal("v")),
				Entry("time", time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC), "year", Equal(2026)),
				Entry("reflection", item{Name: "f"}, "name", Equal("f")),
				Entry("reflection, unknown", item{Name: "f"}, "unknown", BeNil()),
			)
		})

		Describe("Close", func() {

			It("re-creates the file when it is named again", func() {
				p := printer.New()
				app := newApp(p, "value")
				Expect(run(app, "-fprint out.txt")).NotTo(HaveOccurred())

				// The default action closed the file in the After timing, so
				// running again truncates rather than appends
				app = newApp(p, "other")
				Expect(run(app, "-fprint out.txt")).NotTo(HaveOccurred())
				Expect(contentsOf("out.txt")).To(Equal("other\n"))
			})
		})
	})

	Describe("FromContext", func() {

		It("obtains the printer from the context", func() {
			var actual *printer.Printer
			p := printer.New()
			app := newApp(p, "value")
			app.Action = func(c *cli.Context) {
				actual = printer.FromContext(c)
			}

			Expect(run(app, "")).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(p))
		})

		It("panics when the printer is not in the context", func() {
			Expect(func() {
				printer.FromContext(context.Background())
			}).To(PanicWith(MatchError(ContainSubstring("not present in context"))))
		})
	})
})

func exprsOf(app *cli.App) []*expr.Expr {
	return app.Args[1].Value.(*expr.Expression).Exprs
}

func synopsesOf(app *cli.App) []string {
	res := []string{}
	for _, e := range exprsOf(app) {
		res = append(res, e.Synopsis())
	}
	return res
}

type wrapperFS struct {
	afero.Fs
}

func (w wrapperFS) Create(name string) (fs.File, error) {
	return w.Fs.Create(name)
}

func (w wrapperFS) Open(name string) (fs.File, error) {
	return w.Fs.Open(name)
}

func (w wrapperFS) OpenContext(_ context.Context, name string) (fs.File, error) {
	return w.Fs.Open(name)
}

func (w wrapperFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return w.Fs.OpenFile(name, flag, perm)
}

type readOnlyFS struct {
	cli.FS
}

func (readOnlyFS) OpenFile(string, int, fs.FileMode) (fs.File, error) {
	return nil, fs.ErrPermission
}
