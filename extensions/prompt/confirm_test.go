// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompt_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/prompt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SetConfirmed", func() {

	It("defines conventional flag names", func() {
		var flag *cli.Flag
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Uses: prompt.SetConfirmed()},
			},
			Action: func(c *cli.Context) {
				flag, _ = c.LookupFlag("confirm")
			},
		}
		_ = app.RunContext(context.Background(), "app")

		Expect(flag).NotTo(BeNil())
		Expect(flag.Aliases).To(ConsistOf("yes", "y"))
		Expect(flag.HelpText).To(Equal("Assume yes to confirmation prompts"))
	})

	It("does not collide with in-use alias names", func() {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Uses: prompt.SetConfirmed()},
				{Name: "yes"},
				{Name: "y"},
			},
		}
		_, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})

	It("keeps a name specified on the flag", func() {
		var flag *cli.Flag
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Name: "force", Uses: prompt.SetConfirmed()},
			},
			Action: func(c *cli.Context) {
				flag, _ = c.LookupFlag("force")
			},
		}
		_ = app.RunContext(context.Background(), "app")

		Expect(flag).NotTo(BeNil())
	})

	DescribeTable("confirmed", func(arguments string, expected bool) {
		var actual bool
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Uses: prompt.SetConfirmed()},
			},
			Action: bind.SetPointer(&actual, bind.Context().Matches(prompt.Confirmed)),
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.Background(), args...)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	},
		Entry("not specified", "app", false),
		Entry("long", "app --confirm", true),
		Entry("yes alias", "app --yes", true),
		Entry("short alias", "app -y", true),
		Entry("explicit false", "app --confirm=false", false),
	)

	DescribeTable("confirmed in sub-command", func(arguments string, expected bool) {
		var actual bool
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Uses: prompt.SetConfirmed()},
			},
			Commands: []*cli.Command{
				{
					Name:   "sub",
					Action: bind.SetPointer(&actual, bind.Context().Matches(prompt.Confirmed)),
				},
			},
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.Background(), args...)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(expected))
	},
		Entry("not specified", "app sub", false),
		Entry("ancestor confirmed", "app -y sub", true),
	)
})
