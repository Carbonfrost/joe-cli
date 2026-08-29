// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
		falseFilter contextFilterFunc = func(context.Context) bool {
			return false
		}

		trueFilter contextFilterFunc = func(context.Context) bool {
			return true
		}
	)

	var _ = Describe("Any", func() {

		It("is Anything when empty", func() {
			Expect(cli.Any()).To(BeIdenticalTo(cli.Anything))
		})

		It("doesn't match if all don't", func() {
			Expect(cli.Any(falseFilter, falseFilter).Matches(context.Background())).To(BeFalse())
		})

		It("matches if any does", func() {
			Expect(cli.Any(trueFilter, falseFilter).Matches(context.Background())).To(BeTrue())
		})

	})

	var _ = Describe("All", func() {

		It("is Anything when empty", func() {
			Expect(cli.All()).To(BeIdenticalTo(cli.Anything))
		})

		It("doesn't match if any doesn't", func() {
			Expect(cli.All(trueFilter, falseFilter).Matches(context.Background())).To(BeFalse())
		})

		It("matches if all do", func() {
			Expect(cli.All(trueFilter, trueFilter).Matches(context.Background())).To(BeTrue())
		})

	})

})

var _ = Describe("IfMatch", func() {

	It("invokes the then action", func() {
		action1 := new(joeclifakes.FakeAction)
		action2 := new(joeclifakes.FakeAction)
		app := &cli.App{
			Uses: cli.IfMatch(cli.Anything, action1, action2),
		}
		_, _ = app.Initialize(context.Background())
		Expect(action1.ExecuteCallCount()).To(Equal(1))
		Expect(action2.ExecuteCallCount()).To(Equal(0))
	})

	It("invokes the else action", func() {
		action1 := new(joeclifakes.FakeAction)
		action2 := new(joeclifakes.FakeAction)
		app := &cli.App{
			Uses: cli.IfMatch(cli.Completing, action1, action2),
		}
		_, _ = app.Initialize(context.Background())
		Expect(action1.ExecuteCallCount()).To(Equal(0))
		Expect(action2.ExecuteCallCount()).To(Equal(1))
	})

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

		interactiveApp = func(mode cli.ContextFilter) (string, *cli.App) {
			SkipOnWindows()

			// Need to actually re-open the device because Ginkgo or other module
			// may change os.Stdin to prevent the test suite itself from expecting input
			tty, _ := os.OpenFile("/dev/tty", os.O_RDWR, 0)

			return "a", &cli.App{
				Name:   "q",
				Stdin:  tty,
				Action: cli.IfMatch(mode, appendName),
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
		Entry("HasFlag", targetApp, cli.HasFlag("f"), Equal([]string{"c"})),
		Entry("HasArg", targetApp, cli.HasArg("a"), Equal([]string{"c"})),
		Entry("HasCommand", targetApp, cli.HasCommand("c"), Equal([]string{"p"})),
		Entry("Initial", timingApp, cli.InitialTiming, Equal([]string{"i"})),
		Entry("Before", timingApp, cli.BeforeTiming, Equal([]string{"b"})),
		Entry("After", timingApp, cli.AfterTiming, Equal([]string{"a"})),
		Entry("Action", timingApp, cli.ActionTiming, Equal([]string{"c"})),
		// Note that Defines it covered below
		Entry("combination", targetApp, cli.AnyFlag|cli.Seen, Equal([]string{"-f"})),
		Entry("nil matches everything", targetApp, nil, ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("thunk", targetApp, cli.ContextFilterFunc(func(c *cli.Context) bool { return false }), BeEmpty()),
		Entry("nil thunk matches everything", targetApp, cli.ContextFilterFunc(nil), ConsistOf([]string{"-f", "<a>", "c", "p"})),
		Entry("pattern", targetApp, cli.PatternFilter("c -f"), Equal([]string{"-f"})),
		Entry("empty matches everything", targetApp, cli.PatternFilter(""), Equal([]string{"p", "c", "-f", "<a>"})),
		Entry("pattern multi", targetApp, cli.PatternFilter("c -f, c, <a>"), ConsistOf([]string{"-f", "c", "<a>"})),
		Entry("pattern tag", targetApp, cli.PatternFilter("{tag:t}"), ConsistOf([]string{"-f"})),
		Entry("pattern tag bool", targetApp, cli.PatternFilter("{tag}"), ConsistOf([]string{"-f", "<a>"})),
		Entry("interactive", interactiveApp, cli.Interactive, ConsistOf([]string{"q"})),
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
			Entry("Completing", cli.Completing, "COMPLETING"),
			Entry("Defines", cli.Defines, "DEFINES"),
			Entry("Interactive", cli.Interactive, "INTERACTIVE"),
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
			Entry("Completing", cli.Completing, "in shell completion"),
			Entry("Defines", cli.Defines, "defined in joe-cli pkg"),
			Entry("Interactive", cli.Interactive, "interactive"),
		)
	})

	Describe("Defines", func() {
		It("defines on all", func() {
			actual := map[string]bool{}
			app := &cli.App{
				Name: "app",
				Action: func(c *cli.Context) {
					for _, flag := range c.Flags() {
						actual[flag.Name] = c.ContextOf(flag).Matches(cli.Defines)
					}
				},
			}
			_ = app.RunContext(context.Background(), nil)

			Expect(actual).To(Equal(map[string]bool{
				"zsh-completion": true,
				"help":           true,
				"version":        true,
			}))
		})

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

var _ = Describe("PersistentIn", func() {

	var (
		globalAction *joeclifakes.FakeAction

		nothing cli.ActionFunc = func(*cli.Context) error { return nil }

		// app defines --global at the root, but it only applies to sub and to the
		// nested command sub dom
		newApp = func() *cli.App {
			globalAction = new(joeclifakes.FakeAction)
			return &cli.App{
				Name:   "app",
				Action: nothing,
				Flags: []*cli.Flag{
					{
						Name:   "global",
						Value:  cli.Bool(),
						Uses:   cli.PersistentIn(cli.PatternFilter("app sub, app sub dom")),
						Action: globalAction,
					},
				},
				Commands: []*cli.Command{
					{
						Name:        "sub",
						Action:      nothing,
						Subcommands: []*cli.Command{{Name: "dom", Action: nothing}},
					},
					{
						Name:   "other",
						Action: nothing,
					},
				},
			}
		}

		run = func(arguments string) error {
			args, _ := cli.Split(arguments)
			return newApp().RunContext(context.Background(), args)
		}
	)

	DescribeTable("allowed examples", func(arguments string) {
		Expect(run(arguments)).NotTo(HaveOccurred())
		Expect(globalAction.ExecuteCallCount()).To(BeNumerically(">", 0))
	},
		Entry("used within the command", "app sub --global"),
		Entry("used ahead of the command", "app --global sub"),
		Entry("used within a nested command", "app sub dom --global"),
		Entry("used ahead of a nested command", "app --global sub dom"),
	)

	DescribeTable("usage error examples", func(arguments string, expected string) {
		err := run(arguments)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(expected))
		Expect(err).To(BeAssignableToTypeOf(&cli.ParseError{}))
		Expect(err.(*cli.ParseError).Code).To(Equal(cli.UnavailablePersistentOption))
	},
		Entry("used within a command that doesn't match",
			"app other --global", `--global cannot be used with "other"`),
		Entry("used ahead of a command that doesn't match",
			"app --global other", `--global cannot be used with "other"`),
		Entry("used with the command that defines it",
			"app --global", `--global cannot be used with "app"`),
	)

	It("does not run the action within a command that doesn't match", func() {
		_ = run("app other --global")
		Expect(globalAction.ExecuteCallCount()).To(Equal(0))
	})

	It("does not run the action while passing through a command that doesn't match", func() {
		// sub matches, but the root command it was written on does not, so the
		// action is only invoked once
		Expect(run("app --global sub")).NotTo(HaveOccurred())
		Expect(globalAction.ExecuteCallCount()).To(Equal(1))
	})

	DescribeTable("no error when the flag isn't used", func(arguments string) {
		Expect(run(arguments)).NotTo(HaveOccurred())
		Expect(globalAction.ExecuteCallCount()).To(Equal(0))
	},
		Entry("command that matches", "app sub"),
		Entry("command that doesn't match", "app other"),
	)

	It("is an internal error to use on an arg", func() {
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Name: "a",
					Uses: cli.PersistentIn(cli.Anything),
				},
			},
		}
		_, err := app.Initialize(context.Background())
		Expect(err).To(MatchError(ContainSubstring("action can only be used with a flag")))
	})

	It("is too late to use outside of initialization", func() {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:   "global",
					Value:  cli.Bool(),
					Before: cli.PersistentIn(cli.Anything),
				},
			},
		}
		args, _ := cli.Split("app")
		Expect(app.RunContext(context.Background(), args)).To(
			MatchError(ContainSubstring("too late for requested action timing")))
	})
})

type contextFilterFunc func(context.Context) bool

func (f contextFilterFunc) Matches(c context.Context) bool {
	return f(c)
}
