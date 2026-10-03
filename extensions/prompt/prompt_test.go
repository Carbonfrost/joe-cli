// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompt_test

import (
	"context"

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
