// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompt_test

import (
	"context"
	"encoding/json"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/prompt"
	"github.com/Carbonfrost/joe-cli/internal/promptfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FromContext", func() {

	It("obtains the prompter from the context", func() {
		var actual prompt.Prompter
		p := new(promptfakes.FakePrompter)
		app := &cli.App{
			Uses: prompt.ContextValue(p),
			Action: func(c *cli.Context) {
				actual = prompt.FromContext(c)
			},
		}

		Expect(app.RunContext(context.Background(), "app")).NotTo(HaveOccurred())
		Expect(actual).To(BeIdenticalTo(p))
	})

	It("panics when the prompter is not in the context", func() {
		Expect(func() {
			prompt.FromContext(context.Background())
		}).To(PanicWith(MatchError(ContainSubstring("not present in context"))))
	})
})

var _ = Describe("ContextFilter", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(val prompt.ContextFilter, expected string) {
			actual, _ := json.Marshal(val)
			Expect(string(actual)).To(Equal("\"" + expected + "\""))

			var o prompt.ContextFilter
			_ = json.Unmarshal(actual, &o)
			Expect(o).To(Equal(val))
			Expect(o.String()).To(Equal(expected))
		},
			Entry("Defines", prompt.Defines, "prompt.DEFINES"),
			Entry("Confirmed", prompt.Confirmed, "prompt.CONFIRMED"),
		)
	})

	Describe("Describe", func() {

		DescribeTable("examples", func(val prompt.ContextFilter, expected string) {
			actual := val.Describe()
			Expect(actual).To(Equal(expected))
		},
			Entry("Defines", prompt.Defines, "defined in joe-cli/prompt pkg"),
			Entry("Confirmed", prompt.Confirmed, "confirmed"),
		)
	})

	Describe("Defines", func() {
		It("defines on confirm flag", func() {
			actual := map[string]bool{}
			app := &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{Uses: prompt.SetConfirmed()},
				},
				Action: func(c *cli.Context) {
					for _, flag := range c.Flags() {
						actual[flag.Name] = c.ContextOf(flag).Matches(prompt.Defines)
					}
				},
			}
			_ = app.RunContext(context.Background(), nil...)

			Expect(actual).To(Equal(map[string]bool{
				"confirm":        true,
				"zsh-completion": false,
				"help":           false,
				"version":        false,
			}))
		})
	})
})

var _ = Describe("WithAutoConfirmation", func() {

	It("automatically confirms when the Confirmed filter matches", func() {
		inner := new(promptfakes.FakePrompter)
		inner.ConfirmReturns(false, nil)

		var actual bool
		var err error
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Uses: prompt.SetConfirmed()},
			},
			Uses: prompt.ContextValue(prompt.WithAutoConfirmation(inner)),
			Action: func(c context.Context) {
				actual, err = prompt.FromContext(c).Confirm(c, "proceed?", false)
			},
		}
		_ = app.RunContext(context.Background(), "app", "--yes")

		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(BeTrue())
		Expect(inner.ConfirmCallCount()).To(Equal(0))
	})

	It("delegates Confirm when the Confirmed filter does not match", func() {
		inner := new(promptfakes.FakePrompter)
		inner.ConfirmReturns(true, nil)

		var actual bool
		var err error
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Uses: prompt.SetConfirmed()},
			},
			Uses: prompt.ContextValue(prompt.WithAutoConfirmation(inner)),
			Action: func(c context.Context) {
				actual, err = prompt.FromContext(c).Confirm(c, "proceed?", false)
			},
		}
		_ = app.RunContext(context.Background(), "app")

		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(BeTrue())
		Expect(inner.ConfirmCallCount()).To(Equal(1))

		_, p, d := inner.ConfirmArgsForCall(0)
		Expect(p).To(Equal("proceed?"))
		Expect(d).To(BeFalse())
	})

	It("delegates other methods unconditionally", func() {
		inner := new(promptfakes.FakePrompter)
		inner.InputReturns("value", nil)

		w := prompt.WithAutoConfirmation(inner)
		actual, err := w.Input(context.Background(), "name?", "default")

		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal("value"))
		Expect(inner.InputCallCount()).To(Equal(1))
	})
})
