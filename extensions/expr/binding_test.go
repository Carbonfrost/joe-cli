// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr_test

import (
	"context"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type caseInsensitiveBinding struct {
	expr.Binding
	resolved []string
}

func (b *caseInsensitiveBinding) ResolveAlias(arg string) (string, bool) {
	b.resolved = append(b.resolved, arg)
	return b.Binding.ResolveAlias(strings.ToLower(arg))
}

// helloBinding resolves -hello to an item which doesn't support bindings
type helloBinding struct {
	expr.Binding
	hello expr.BindingEvaluator
}

func (b *helloBinding) ResolveAlias(arg string) (string, bool) {
	if arg == "-hello" {
		return "hello", true
	}
	return b.Binding.ResolveAlias(arg)
}

func (b *helloBinding) LookupExpr(name string) (expr.BindingEvaluator, bool) {
	if name == "hello" {
		return b.hello, true
	}
	return b.Binding.LookupExpr(name)
}

type nothingToParse struct {
	expr.BindingEvaluator
	applied int
}

func (*nothingToParse) Binding() cli.Binding { return nil }
func (n *nothingToParse) Apply(*cli.BindingResult) error {
	n.applied++
	return nil
}

var _ = Describe("Binding", func() {

	Describe("NewBinding", func() {

		It("resolves names and aliases to the canonical name", func() {
			b := expr.NewBinding([]*expr.Expr{{Name: "name", Aliases: []string{"n"}}}, 0)

			for _, arg := range []string{"-name", "-n"} {
				name, ok := b.ResolveAlias(arg)
				Expect(ok).To(BeTrue())
				Expect(name).To(Equal("name"))
			}
		})

		It("requires the dash to resolve an expression operator", func() {
			b := expr.NewBinding([]*expr.Expr{{Name: "name"}}, 0)

			for _, arg := range []string{"name", "-unknown", "-", ""} {
				_, ok := b.ResolveAlias(arg)
				Expect(ok).To(BeFalse(), arg)
			}
		})

		It("looks up by canonical name only", func() {
			b := expr.NewBinding([]*expr.Expr{{Name: "name", Aliases: []string{"n"}}}, 0)

			ev, ok := b.LookupExpr("name")
			Expect(ok).To(BeTrue())
			Expect(ev.Expr().Name).To(Equal("name"))
			Expect(ev).To(Satisfy(func(v any) bool { _, ok := v.(expr.BindingState); return ok }))

			_, ok = b.LookupExpr("n")
			Expect(ok).To(BeFalse())
		})

		It("creates new state for each lookup", func() {
			b := expr.NewBinding([]*expr.Expr{{Name: "name"}}, 0)

			s1, _ := b.LookupExpr("name")
			s2, _ := b.LookupExpr("name")
			Expect(s1).NotTo(BeIdenticalTo(s2))
		})

		It("state provides the binding for the args", func() {
			b := expr.NewBinding([]*expr.Expr{{Name: "name", Args: cli.Args("a", cli.String())}}, 0)

			ev, _ := b.LookupExpr("name")
			state := ev.(expr.BindingState)

			Expect(state.Binding().PositionalArgNames()).To(Equal([]string{"a"}))
		})

		It("resolves operators only with ParseOperators", func() {
			without := expr.NewBinding(nil, 0)
			with := expr.NewBinding(nil, expr.ParseOperators)

			_, ok := without.ResolveAlias("-and")
			Expect(ok).To(BeFalse())

			for _, arg := range []string{"-and", "-a"} {
				name, ok := with.ResolveAlias(arg)
				Expect(ok).To(BeTrue())

				ev, ok := with.LookupExpr(name)
				Expect(ok).To(BeTrue())
				Expect(ev).To(Equal(expr.And))
			}
		})

		It("operators don't support bindings", func() {
			b := expr.NewBinding(nil, expr.ParseOperators)

			ev, _ := b.LookupExpr("(")
			Expect(ev).NotTo(Satisfy(func(v any) bool { _, ok := v.(expr.BindingState); return ok }))
		})

		It("identifies the numeric expression operator", func() {
			number := &expr.Expr{Name: "number"}
			number.SetNumeric(true)

			Expect(expr.NewBinding([]*expr.Expr{{Name: "other"}}, 0).Numeric()).To(BeEmpty())
			Expect(expr.NewBinding([]*expr.Expr{number}, 0).Numeric()).To(Equal("number"))
		})
	})

	Describe("Expression.Binding", func() {

		newApp := func(e *expr.Expression) *cli.App {
			return &cli.App{
				Name: "app",
				Action: func(c *cli.Context) {
					expr.FromContext(c, "e").Evaluate(c, "")
				},
				Args: []*cli.Arg{
					{
						Name: "f",
						NArg: -2,
					},
					{
						Name:  "e",
						Value: e,
					},
				},
			}
		}

		It("uses the binding to look up expression operators", func() {
			values := map[string]any{}
			name := &expr.Expr{
				Name: "name",
				Args: []*cli.Arg{{Value: new(string)}},
				Evaluate: func(c *cli.Context) {
					values["name"] = c.Value(0)
				},
			}
			binding := &caseInsensitiveBinding{
				Binding: expr.NewBinding([]*expr.Expr{name}, 0),
			}
			app := newApp(&expr.Expression{
				Exprs:   []*expr.Expr{name},
				Binding: binding,
			})

			args, _ := cli.Split("app arg -NAME value")
			err := app.RunContext(context.Background(), args...)

			Expect(err).NotTo(HaveOccurred())
			Expect(binding.resolved).To(Equal([]string{"-NAME"}))
			Expect(values).To(HaveKeyWithValue("name", "value"))
		})

		It("reports unknown expression operators from the binding", func() {
			name := &expr.Expr{Name: "name"}
			app := newApp(&expr.Expression{
				Exprs:   []*expr.Expr{name},
				Binding: expr.NewBinding(nil, 0),
			})

			args, _ := cli.Split("app arg -name")
			err := app.RunContext(context.Background(), args...)

			Expect(err).To(MatchError("unknown expression: -name"))
		})

		It("places items which don't support bindings in the pipeline", func() {
			values := map[string]any{}
			hello := expr.NewBindingEvaluator(expr.EvaluatorOf(func() {
				values["hello"] = true
			}))
			binding := &helloBinding{
				Binding: expr.NewBinding(nil, 0),
				hello:   hello,
			}
			app := newApp(&expr.Expression{Binding: binding})

			args, _ := cli.Split("app arg -hello")
			err := app.RunContext(context.Background(), args...)

			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveKeyWithValue("hello", true))
		})

		It("applies ParseAllowInlineValues to the binding", func() {
			values := map[string]any{}
			name := &expr.Expr{
				Name:    "name",
				Aliases: []string{"N"},
				Args:    []*cli.Arg{{Value: new(string)}},
				Evaluate: func(c *cli.Context) {
					values["name"] = c.Value(0)
				},
			}
			app := newApp(&expr.Expression{
				Exprs:   []*expr.Expr{name},
				Binding: expr.NewBinding([]*expr.Expr{name}, 0),
			})
			app.Args[1].Options = expr.ParseAllowInlineValues

			args, _ := cli.Split("app arg -Nvalue")
			err := app.RunContext(context.Background(), args...)

			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveKeyWithValue("name", "value"))
		})

		It("reports unknown inline names rather than failing", func() {
			name := &expr.Expr{Name: "name", Aliases: []string{"N"}}
			app := newApp(&expr.Expression{Exprs: []*expr.Expr{name}})
			app.Args[1].Options = expr.ParseAllowInlineValues

			args, _ := cli.Split("app arg -Zvalue")
			err := app.RunContext(context.Background(), args...)

			Expect(err).To(MatchError("unknown expression: -Zvalue"))
		})

		It("sees names that change during initialization", func() {
			values := map[string]any{}
			number := &expr.Expr{
				Name:    "number",
				Args:    []*cli.Arg{{Value: new(int)}},
				Options: cli.Numeric,
				Evaluate: func(c *cli.Context) {
					values["number"] = c.Value(0)
				},
			}

			// Options such as Numeric are applied after the binding is created
			app := newApp(&expr.Expression{
				Exprs:   []*expr.Expr{number},
				Binding: expr.NewBinding([]*expr.Expr{number}, 0),
			})

			args, _ := cli.Split("app arg -12")
			err := app.RunContext(context.Background(), args...)

			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveKeyWithValue("number", 12))
		})

		Describe("operators", func() {
			var items []expr.BindingEvaluator

			compiler := func(i []expr.BindingEvaluator) (expr.Evaluator, error) {
				items = i
				return expr.AlwaysTrue, nil
			}

			BeforeEach(func() {
				items = nil
			})

			It("parses operators from the binding", func() {
				name := &expr.Expr{Name: "name"}
				app := newApp(&expr.Expression{
					Exprs:    []*expr.Expr{name},
					Binding:  expr.NewBinding([]*expr.Expr{name}, expr.ParseOperators),
					Compiler: compiler,
				})

				args, _ := cli.Split("app arg -name -or ( -name )")
				err := app.RunContext(context.Background(), args...)

				Expect(err).NotTo(HaveOccurred())
				Expect(items).To(HaveLen(5))
				Expect(items[1]).To(Equal(expr.Or))
				Expect(items[2]).To(Equal(expr.LParen))
				Expect(items[4]).To(Equal(expr.RParen))
			})

			It("does not parse an operator alias as an inline value", func() {
				name := &expr.Expr{Name: "name", Aliases: []string{"N"}}
				app := newApp(&expr.Expression{
					Exprs:    []*expr.Expr{name},
					Binding:  expr.NewBinding([]*expr.Expr{name}, expr.ParseOperators),
					Compiler: compiler,
				})
				app.Args[1].Options = expr.ParseAllowInlineValues

				args, _ := cli.Split("app arg -a=1")
				err := app.RunContext(context.Background(), args...)

				Expect(err).To(MatchError("unknown expression: -a=1"))
			})
		})

		It("does not parse or apply a state which has nothing to parse", func() {
			state := &nothingToParse{BindingEvaluator: expr.NewBindingEvaluator(expr.Invariant(true))}
			binding := &helloBinding{
				Binding: expr.NewBinding(nil, 0),
				hello:   state,
			}
			app := newApp(&expr.Expression{Binding: binding})

			args, _ := cli.Split("app arg -hello")
			err := app.RunContext(context.Background(), args...)

			Expect(err).NotTo(HaveOccurred())
			Expect(state.applied).To(BeZero())
		})

		It("is copied by Clone", func() {
			binding := expr.NewBinding(nil, 0)
			e := &expr.Expression{Binding: binding}

			Expect(e.Clone().Binding).To(BeIdenticalTo(binding))
		})
	})
})
