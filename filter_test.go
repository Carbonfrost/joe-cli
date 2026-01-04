// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Carbonfrost/joe-cli"
	joeclifakes "github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Assert", func() {
	DescribeTable("examples", func(app *cli.App, expected types.GomegaMatcher) {
		args, _ := cli.Split("app")
		err := app.RunContext(context.Background(), args)

		Expect(err).To(HaveOccurred())
		Expect(err).To(expected)
	},
		Entry(
			"initial timing",
			&cli.App{Action: cli.Assert(cli.InitialTiming, nil)},
			MatchError(ContainSubstring("context must be initial timing"))),
		Entry(
			"before timing",
			&cli.App{Action: cli.Assert(cli.BeforeTiming, nil)},
			MatchError(ContainSubstring("context must be before timing"))),
		Entry(
			"after timing",
			&cli.App{Action: cli.Assert(cli.AfterTiming, nil)},
			MatchError(ContainSubstring("context must be after timing"))),
		Entry(
			"has value",
			&cli.App{Action: cli.Assert(cli.HasValue, nil)},
			MatchError(ContainSubstring("context must be target with value"))),
	)
})

var _ = Context("universe filters", func() {

	var (
		falseFilter cli.ContextFilterFunc = func(c *cli.Context) bool {
			return false
		}

		trueFilter cli.ContextFilterFunc = func(c *cli.Context) bool {
			return true
		}
	)

	var _ = Describe("Any", func() {

		It("is Anything when empty", func() {
			Expect(cli.Any()).To(BeIdenticalTo(cli.Anything))
		})

		It("doesn't match if all don't", func() {
			Expect(cli.Any(falseFilter, falseFilter).Matches(nil)).To(BeFalse())
		})

		It("matches if any does", func() {
			Expect(cli.Any(trueFilter, falseFilter).Matches(nil)).To(BeTrue())
		})

	})

	var _ = Describe("All", func() {

		It("is Anything when empty", func() {
			Expect(cli.All()).To(BeIdenticalTo(cli.Anything))
		})

		It("doesn't match if any doesn't", func() {
			Expect(cli.All(trueFilter, falseFilter).Matches(nil)).To(BeFalse())
		})

		It("matches if all do", func() {
			Expect(cli.All(trueFilter, trueFilter).Matches(nil)).To(BeTrue())
		})

	})

})

var _ = Describe("IfMatch", func() {

	var (
		timingStrings = map[cli.Timing]string{
			cli.InitialTiming: "i",
			cli.BeforeTiming:  "b",
			cli.ActionTiming:  "c",
			cli.AfterTiming:   "a",
		}

		res []string

		appendName cli.ActionFunc = func(c *cli.Context) error {
			res = append(res, c.Name())
			return nil
		}

		appendTiming cli.ActionFunc = func(c *cli.Context) error {
			res = append(res, timingStrings[c.Timing()])
			return nil
		}

		targetApp = func(mode cli.ContextFilter) (string, *cli.App) {
			return "p c -f a", &cli.App{
				Name: "p",
				Commands: []*cli.Command{
					{
						Name:   "c",
						Before: cli.IfMatch(mode, appendName),
						Flags: []*cli.Flag{
							{
								Name:   "f",
								Value:  new(bool),
								Before: cli.IfMatch(mode, appendName),
								Data: map[string]any{
									"tag": "t",
								},
							},
						},
						Args: []*cli.Arg{
							{
								Name:   "a",
								Before: cli.IfMatch(mode, appendName),
								Data: map[string]any{
									"tag": "x",
								},
							},
						},
					},
				},
				Uses: cli.IfMatch(mode, appendName),
			}
		}

		timingApp = func(mode cli.ContextFilter) (string, *cli.App) {
			return "p", &cli.App{
				Uses:   cli.IfMatch(mode, appendTiming),
				Before: cli.IfMatch(mode, appendTiming),
				After:  cli.IfMatch(mode, appendTiming),
				Action: cli.IfMatch(mode, appendTiming),
			}
		}
	)

	JustBeforeEach(func() {
		res = nil
	})

	DescribeTable("examples", func(createApp func(cli.ContextFilter) (string, *cli.App), m cli.ContextFilter, expected types.GomegaMatcher) {
		arguments, app := createApp(m)

		args, _ := cli.Split(arguments)
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(expected)
	},
		Entry("AnyFlag", targetApp, cli.AnyFlag, Equal([]string{"-f"})),
		Entry("AnyArg", targetApp, cli.AnyArg, Equal([]string{"<a>"})),
		Entry("Anything", targetApp, cli.Anything, ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("HasValue", targetApp, cli.HasValue, Equal([]string{"-f", "<a>"})),
		Entry("RootCommand", targetApp, cli.RootCommand, Equal([]string{"p"})),
		Entry("Seen", targetApp, cli.Seen, ConsistOf([]string{"-f", "<a>"})),
		Entry("HasSeen", targetApp, cli.HasSeen("f"), ConsistOf([]string{"c", "-f", "<a>"})),
		Entry("Initial", timingApp, cli.InitialTiming, Equal([]string{"i"})),
		Entry("Before", timingApp, cli.BeforeTiming, Equal([]string{"b"})),
		Entry("After", timingApp, cli.AfterTiming, Equal([]string{"a"})),
		Entry("Action", timingApp, cli.ActionTiming, Equal([]string{"c"})),
		Entry("combination", targetApp, cli.AnyFlag|cli.Seen, Equal([]string{"-f"})),
		Entry("nil matches everything", targetApp, nil, ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("thunk", targetApp, cli.ContextFilterFunc(func(c *cli.Context) bool { return false }), BeEmpty()),
		Entry("nil thunk matches everything", targetApp, cli.ContextFilterFunc(nil), ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("pattern", targetApp, cli.PatternFilter("c -f"), Equal([]string{"-f"})),
		Entry("empty matches everything", targetApp, cli.PatternFilter(""), Equal([]string{"p", "c", "-f", "<a>"})),
		Entry("pattern multi", targetApp, cli.PatternFilter("c -f, c, <a>"), ConsistOf([]string{"-f", "c", "<a>"})),
		Entry("pattern tag", targetApp, cli.PatternFilter("{tag:t}"), ConsistOf([]string{"-f"})),
		Entry("pattern tag bool", targetApp, cli.PatternFilter("{tag}"), ConsistOf([]string{"-f", "<a>"})),
	)
})

var _ = Describe("FilterModes", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(val cli.FilterModes, expected string) {
			actual, _ := json.Marshal(val)
			Expect(string(actual)).To(Equal("\"" + expected + "\""))

			var o cli.FilterModes
			_ = json.Unmarshal(actual, &o)
			Expect(o).To(Equal(val))
			Expect(o.String()).To(Equal(expected))
		},
			Entry("AnyFlag", cli.AnyFlag, "ANY_FLAG"),
			Entry("AnyArg", cli.AnyArg, "ANY_ARG"),
			Entry("Anything", cli.Anything, "ANYTHING"),
			Entry("HasValue", cli.HasValue, "HAS_VALUE"),
			Entry("RootCommand", cli.RootCommand, "ROOT_COMMAND"),
			Entry("Seen", cli.Seen, "SEEN"),
		)
	})

	Describe("Describe", func() {

		DescribeTable("examples", func(val cli.FilterModes, expected string) {
			actual := val.Describe()
			Expect(actual).To(Equal(expected))
		},
			Entry("AnyFlag", cli.AnyFlag, "any flag"),
			Entry("AnyArg", cli.AnyArg, "any arg"),
			Entry("Anything", cli.Anything, "anything"),
			Entry("HasValue", cli.HasValue, "target with value"),
			Entry("RootCommand", cli.RootCommand, "root command"),
			Entry("Seen", cli.Seen, "option that has been seen"),
		)
	})
})

var _ = Describe("HasData", func() {

	It("matches context by key", func() {
		fake := new(joeclifakes.FakeAction)
		app := &cli.App{
			Uses: cli.Pipeline(
				cli.Data("key", "value"),
				cli.IfMatch(cli.HasData("key"), fake),
			),
		}

		_, _ = app.Initialize(context.Background())
		Expect(fake.ExecuteCallCount()).To(Equal(1))
	})

	It("matches context by inherited key", func() {
		fake := new(joeclifakes.FakeAction)
		app := &cli.App{
			Commands: []*cli.Command{
				{
					Name: "sub",
					Uses: cli.IfMatch(cli.HasData("key"), fake),
				},
			},
			Uses: cli.Data("key", "value"),
		}
		_ = app.RunContext(context.Background(), []string{"app", "sub"})
		Expect(fake.ExecuteCallCount()).To(Equal(1))
	})

	It("matches context by key and value", func() {
		fake := new(joeclifakes.FakeAction)
		app := &cli.App{
			Uses: cli.Pipeline(
				cli.Data("key", "value"),
				cli.IfMatch(cli.HasData("key", "value"), fake),
			),
		}
		_, _ = app.Initialize(context.Background())
		Expect(fake.ExecuteCallCount()).To(Equal(1))
	})

	It("does not match context with different value", func() {
		fake := new(joeclifakes.FakeAction)
		app := &cli.App{
			Uses: cli.Pipeline(
				cli.Data("key", "value"),
				cli.IfMatch(cli.HasData("key", "nonmatchingvalue"), fake),
			),
		}
		_, _ = app.Initialize(context.Background())
		Expect(fake.ExecuteCallCount()).To(Equal(0))
	})

	Context("string representation", func() {
		DescribeTable("examples", func(subj cli.ContextFilter, expected string) {
			Expect(fmt.Sprint(subj)).To(Equal(expected))
		},
			Entry("nominal", cli.HasData("tag", "t"), "{tag:t}"),
			Entry("key only", cli.HasData("tag"), "{tag}"),
			Entry("non-string", cli.HasData("t", 2), "{t 2}"),
		)

	})
})
