// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompt_test

import (
	"context"
	"errors"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/prompt"
	"github.com/Carbonfrost/joe-cli/internal/promptfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("ActionBinder", func() {

	var p *promptfakes.FakePrompter

	BeforeEach(func() {
		p = new(promptfakes.FakePrompter)
		p.InputReturns("input", nil)
		p.PasswordReturns("password", nil)
		p.ConfirmReturns(true, nil)
		p.SelectReturns(1, nil)
		p.MultiSelectReturns([]int{0, 2}, nil)
		p.EditReturns("edit", nil)
	})

	Context("as a binder", func() {

		DescribeTable("examples",
			func(binder func(*any) any, expected any) {
				var actual any
				app := &cli.App{
					Uses:   prompt.ContextValue(p),
					Action: binder(&actual),
				}
				Expect(app.RunContext(context.Background(), "app")).To(Succeed())
				Expect(actual).To(Equal(expected))
			},
			Entry("Input", capture(prompt.Input("Name", "")), "input"),
			Entry("Password", capture(prompt.Password("Password")), "password"),
			Entry("Confirm", capture(prompt.Confirm("OK?", false)), true),
			Entry("Select", capture(prompt.Select("Pick", "a", []string{"a", "b"})), 1),
			Entry("MultiSelect", capture(prompt.MultiSelect("Pick", nil, []string{"a", "b", "c"})), []int{0, 2}),
			Entry("Edit", capture(prompt.Edit("Text", "", true)), "edit"),
			Entry("Select Index", capture(prompt.Select("Pick", "a", []string{"a", "b"}).Index()), 1),
			Entry("Select Value", capture(prompt.Select("Pick", "a", []string{"a", "b"}).Value()), "b"),
			Entry("MultiSelect Index", capture(prompt.MultiSelect("Pick", nil, []string{"a", "b", "c"}).Indexes()), []int{0, 2}),
			Entry("MultiSelect Value", capture(prompt.MultiSelect("Pick", nil, []string{"a", "b", "c"}).Values()), []string{"a", "c"}),
		)

		It("returns an error when no prompter is in the context", func() {
			app := &cli.App{
				Action: bind.Call(func(string) error { return nil }, prompt.Input("Name", "")),
			}
			Expect(app.RunContext(context.Background(), "app")).To(
				MatchError(ContainSubstring("not present in context")))
		})

	})

	Context("as an action on a flag", func() {

		DescribeTable("sets the value of the flag",
			func(action cli.Action, expected any) {
				app := &cli.App{
					Uses: prompt.ContextValue(p),
					Flags: []*cli.Flag{
						{Name: "f", Uses: action},
					},
				}
				Expect(app.RunContext(context.Background(), "app", "-f", "x")).To(Succeed())
				Expect(app.Flags[0].Value).To(PointTo(Equal(expected)))
			},
			Entry("Input", prompt.Input("Name", ""), "input"),
			Entry("Password", prompt.Password("Password"), "password"),
			Entry("Edit", prompt.Edit("Text", "", true), "edit"),
			Entry("Select", prompt.Select("Pick", "a", []string{"a", "b"}), "b"),
		)

		It("sets the value of a Confirm flag", func() {
			app := &cli.App{
				Uses: prompt.ContextValue(p),
				Flags: []*cli.Flag{
					{Name: "f", Uses: prompt.Confirm("OK?", false)},
				},
			}
			Expect(app.RunContext(context.Background(), "app", "-f")).To(Succeed())
			Expect(app.Flags[0].Value).To(PointTo(BeTrue()))
		})

		It("sets the value of a MultiSelect flag", func() {
			app := &cli.App{
				Uses: prompt.ContextValue(p),
				Flags: []*cli.Flag{
					{Name: "f", Uses: prompt.MultiSelect("Pick", nil, []string{"a", "b", "c"})},
				},
			}
			Expect(app.RunContext(context.Background(), "app", "-f", "x")).To(Succeed())
			Expect(app.Flags[0].Value).To(PointTo(Equal([]string{"a", "c"})))
		})

		It("schedules the prompt for the Action timing from Before", func() {
			app := &cli.App{
				Uses: prompt.ContextValue(p),
				Flags: []*cli.Flag{
					{Name: "f", Before: prompt.Input("Name", "")},
				},
			}
			Expect(app.RunContext(context.Background(), "app", "-f", "x")).To(Succeed())
			Expect(p.InputCallCount()).To(Equal(1))
			Expect(app.Flags[0].Value).To(PointTo(Equal("input")))
		})

		It("does not prompt when the flag is not set", func() {
			app := &cli.App{
				Uses: prompt.ContextValue(p),
				Flags: []*cli.Flag{
					{Name: "f", Uses: prompt.Input("Name", "")},
				},
			}
			Expect(app.RunContext(context.Background(), "app")).To(Succeed())
			Expect(p.InputCallCount()).To(Equal(0))
		})

		It("propagates the error from the prompter", func() {
			p.InputReturns("", errors.New("prompt failed"))
			app := &cli.App{
				Uses: prompt.ContextValue(p),
				Flags: []*cli.Flag{
					{Name: "f", Uses: prompt.Input("Name", "")},
				},
			}
			Expect(app.RunContext(context.Background(), "app", "-f", "x")).To(
				MatchError(ContainSubstring("prompt failed")))
		})

		It("returns an error when no prompter is in the context", func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "f", Uses: prompt.Input("Name", "")},
				},
			}
			Expect(app.RunContext(context.Background(), "app", "-f", "x")).To(
				MatchError(ContainSubstring("not present in context")))
		})
	})

	Context("in ImplicitValueTiming", func() {

		It("displays the prompt immediately and sets the value", func() {
			app := &cli.App{
				Uses: prompt.ContextValue(p),
				Flags: []*cli.Flag{
					{Name: "f", Uses: cli.Implicitly(prompt.Input("Name", ""))},
				},
			}
			Expect(app.RunContext(context.Background(), "app")).To(Succeed())
			Expect(p.InputCallCount()).To(Equal(1))
			Expect(app.Flags[0].Value).To(PointTo(Equal("input")))
		})

		It("does nothing when no prompter is in the context", func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "f", Uses: cli.Implicitly(prompt.Input("Name", ""))},
				},
			}
			Expect(app.RunContext(context.Background(), "app")).To(Succeed())
		})

		It("does nothing when value is already set", func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "f", Uses: cli.Implicitly(prompt.Input("Name", ""))},
				},
			}
			Expect(app.RunContext(context.Background(), "app", "-f", "Gary")).To(Succeed())
			Expect(p.InputCallCount()).To(Equal(0))
		})
	})

	Context("as an action on a command", func() {

		It("displays the prompt in the Action timing", func() {
			app := &cli.App{
				Uses: cli.Pipeline(prompt.ContextValue(p), prompt.Confirm("OK?", false)),
			}
			Expect(app.RunContext(context.Background(), "app")).To(Succeed())
			Expect(p.ConfirmCallCount()).To(Equal(1))
		})

		It("returns an error when no prompter is in the context", func() {
			app := &cli.App{
				Uses: prompt.Confirm("OK?", false),
			}
			Expect(app.RunContext(context.Background(), "app")).To(
				MatchError(ContainSubstring("not present in context")))
		})
	})
})

func capture[T any](binder bind.Binder[T]) func(*any) any {
	return func(actual *any) any {
		return bind.Call(func(v T) error {
			*actual = v
			return nil
		}, binder)
	}
}
