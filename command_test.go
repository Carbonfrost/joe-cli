package cli_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {

	It("sub-command invocation should not split arguments", func() {
		var t, u string
		app := &cli.App{
			Commands: []*cli.Command{
				{
					Name: "sub",
					Args: cli.Args("t", &t, "u", &u),
				},
			},
		}
		args, _ := cli.Split("app sub t,a,b u")
		_ = app.RunContext(context.TODO(), args)
		Expect(t).To(Equal("t,a,b"))
		Expect(u).To(Equal("u"))
	})

	It("allow arguments and sub-commands", func() {
		// to support pastiche, reversing previous behavior to require only sub-commands
		var t string
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "scope",
					NArg: cli.OptionalArg(regexp.MustCompile("^https?:").MatchString),
				},
			},
			Commands: []*cli.Command{
				{
					Name:   "sub",
					Args:   cli.Args("t", &t),
					Action: act,
				},
			},
		}
		args, _ := cli.Split("app https://example.com sub t")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(t).To(Equal("t"))
		Expect(act.ExecuteCallCount()).To(Equal(1))
	})

	Describe("actions", func() {

		var (
			act       *joeclifakes.FakeAction
			beforeAct *joeclifakes.FakeAction

			app *cli.App
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			beforeAct = new(joeclifakes.FakeAction)

			app = &cli.App{
				Commands: []*cli.Command{
					{
						Name:   "c",
						Action: act,
						Before: beforeAct,
						Args: []*cli.Arg{
							{
								Name: "a",
								NArg: -1,
							},
						},
					},
				},
			}

			args, _ := cli.Split("app c args args")
			app.RunContext(context.TODO(), args)
		})

		It("executes action on executing sub-command", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("contains args in captured context", func() {
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Args()).To(Equal([]string{"c", "args", "args"}))
		})

		It("executes before action on executing sub-command", func() {
			Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
		})
	})

	Describe("SkipFlagParsing", func() {

		It("disables parsing of flags", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name:    "app",
				Options: cli.SkipFlagParsing,
				Action:  act,
				Args: []*cli.Arg{
					{
						Name: "args",
						NArg: -1,
					},
				},
			}

			err := app.RunContext(context.TODO(), []string{"app", "-a", "-b"})
			Expect(err).NotTo(HaveOccurred())
			captured := act.ExecuteArgsForCall(0)

			Expect(captured.List("args")).To(Equal([]string{"-a", "-b"}))
		})

	})

	Describe("DisallowFlagsAfterArgs", func() {
		It("causes flags after args error", func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "whitespace",
					},
				},
				Args: []*cli.Arg{
					{
						Name: "items",
						NArg: -2,
					},
				},
				Options: cli.DisallowFlagsAfterArgs,
			}
			arguments := "app arg --whitespace"
			args, _ := cli.Split(arguments)
			err := app.RunContext(context.TODO(), args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("can't use --whitespace after arguments"))
		})
	})

	Describe("RightToLeft", func() {

		It("propagates errors on implicitly skipped arguments", func() {
			// If the arg counter actually enforces an error (on the Done() call),
			// then this error should be available
			counter := new(joeclifakes.FakeArgCounter)
			counter.DoneReturnsOnCall(0, fmt.Errorf("done error"))
			app := &cli.App{
				Name:    "app",
				Options: cli.RightToLeft,
				Args: []*cli.Arg{
					{
						Name: "a", NArg: counter, Value: cli.List(),
					},
					{
						Name: "r", NArg: 1, Value: cli.List(),
					},
				},
			}

			args, _ := cli.Split("app one")
			err := app.RunContext(context.TODO(), args)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("done error"))
			Expect(counter.DoneCallCount()).To(Equal(1))
		})

		DescribeTable("examples", func(options []*cli.Arg, arguments string, expected []string) {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name:    "app",
				Options: cli.RightToLeft,
				Action:  act,
				Args:    options,
			}

			args, _ := cli.Split("app " + arguments)
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())
			captured := act.ExecuteArgsForCall(0)

			actual := func() []string {
				res := []string{}
				for _, v := range captured.Values() {
					res = append(res, strings.Trim(fmt.Sprint(v), "[]"))
				}
				return res
			}()

			Expect(actual).To(Equal(expected))
		},
			Entry("fill remaining first", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: 1, Value: cli.List(),
				},
			}, "1", []string{"", "1"}),

			Entry("fill remaining first 2", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "b", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: 1, Value: cli.List(),
				},
			}, "1", []string{"", "", "1"}),

			Entry("fill remaining first 3", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "b", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: 1, Value: cli.List(),
				},
			}, "1 2", []string{"", "1", "2"}),

			Entry("fill remaining first 3", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "b", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: 1, Value: cli.List(),
				},
			}, "1 2 3", []string{"1", "2", "3"}),

			Entry("fill list remaining first", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: -1, Value: cli.List(),
				},
			}, "1", []string{"", "1"}),

			Entry("fill list remaining first 2", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "b", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: -1, Value: cli.List(),
				},
			}, "1", []string{"", "", "1"}),

			Entry("fill list remaining first 3", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "b", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: -1, Value: cli.List(),
				},
			}, "1 2", []string{"", "1", "2"}),

			Entry("minimum requirement met", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: -1, Value: cli.List(),
				},
			}, "1 2", []string{"1", "2"}),

			Entry("minimum requirement met excess", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "r", NArg: -1, Value: cli.List(),
				},
			}, "1 2 3", []string{"1", "2 3"}),

			Entry("discrete counts", []*cli.Arg{
				{
					Name: "a", NArg: 0, Value: cli.List(),
				},
				{
					Name: "b", NArg: 2, Value: cli.List(),
				},
				{
					Name: "r", NArg: -1, Value: cli.List(),
				},
			}, "1 2 3", []string{"", "1 2", "3"}),
		)
	})

	Describe("Synopsis", func() {
		DescribeTable("examples",
			func(cmd *cli.Command, expected string) {
				cli.InitializeCommand(cmd)
				Expect(cmd.Synopsis()).To(Equal(expected))
			},
			Entry(
				"combine and sort boolean short flags",
				&cli.Command{
					Flags: []*cli.Flag{
						{Name: "t", Value: cli.Bool()},
						{Name: "s", Value: cli.Bool()},
						{Name: "g", Value: cli.Bool()},
						{Name: "h", Value: cli.Bool()},
						{Name: "o", Value: cli.Bool()},
					},
					Name: "cmd",
				},
				"cmd [-ghost]",
			),
			Entry(
				"use long name with value",
				&cli.Command{
					Flags: []*cli.Flag{
						{Name: "tan", Aliases: []string{"a"}, Value: cli.String()},
						{Name: "h", Aliases: []string{"cos"}, Value: cli.String()},
					},
					Name: "cmd",
				},
				"cmd [--tan=STRING] [--cos=STRING]",
			),
			Entry(
				"flags and args",
				&cli.Command{
					Flags: []*cli.Flag{
						{Name: "t", Value: cli.Bool()},
					},
					Args: []*cli.Arg{
						{Name: "arg", NArg: 1},
					},
					Name: "cmd",
				},
				"cmd [-t] <arg>",
			),
			Entry(
				"optional arguments",
				&cli.Command{
					Args: []*cli.Arg{
						{Name: "arg"},
					},
					Name: "cmd",
				},
				"cmd [<arg>]",
			),
			Entry(
				"right-to-left arguments",
				&cli.Command{
					Args: []*cli.Arg{
						{Name: "a"},
						{Name: "b"},
					},
					Options: cli.RightToLeft,
					Name:    "cmd",
				},
				"cmd [[<a>] <b>]",
			),
			Entry(
				"right-to-left arguments non-optional",
				&cli.Command{
					Args: []*cli.Arg{
						{Name: "a", NArg: 1},
						{Name: "b"},
						{Name: "c"},
					},
					Options: cli.RightToLeft,
					Name:    "cmd",
				},
				"cmd <a> [[<b>] <c>]",
			),
		)

	})
})
