package cli_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
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
				Name: "a",
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
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Args()).To(Equal([]string{"c", "args", "args"}))
		})

		It("executes before action on executing sub-command", func() {
			Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
		})

		It("obtains context path", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Path().IsCommand()).To(BeTrue())
			Expect(captured.Path().Last()).To(Equal("c"))
			Expect(captured.Path().String()).To(Equal("a c"))
		})
	})

	DescribeTable("initializers", func(act cli.Action, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "a",
			Commands: []*cli.Command{
				{
					Name: "c",
					Uses: act,
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
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		cmd, ok := app.Command("c")
		Expect(ok).To(BeTrue())
		Expect(cmd).To(PointTo(expected))
	},
		Entry(
			"Category",
			cli.Category("abc"),
			MatchFields(IgnoreExtras, Fields{"Category": Equal("abc")}),
		),
		Entry(
			"Alias",
			cli.Alias("abc"),
			MatchFields(IgnoreExtras, Fields{"Aliases": Equal([]string{"abc"})}),
		),
		Entry(
			"ManualText",
			cli.ManualText("abc"),
			MatchFields(IgnoreExtras, Fields{"ManualText": Equal("abc")}),
		),
		Entry(
			"HelpText",
			cli.HelpText("abc"),
			MatchFields(IgnoreExtras, Fields{"HelpText": Equal("abc")}),
		),
		Entry(
			"UsageText",
			cli.UsageText("abc"),
			MatchFields(IgnoreExtras, Fields{"UsageText": Equal("abc")}),
		),
		Entry(
			"Description",
			cli.Description("abc"),
			MatchFields(IgnoreExtras, Fields{"Description": Equal("abc")}),
		),
	)

	Describe("visibility", func() {
		DescribeTable("examples", func(cmd *cli.Command, finder string, visibleExpected bool) {
			var found any
			app := &cli.App{
				Commands: []*cli.Command{cmd},
				Action: func(c *cli.Context) {
					parent, _ := c.FindTarget(strings.Fields("app " + finder))
					found = parent.Target()
				},
				Name: "app",
			}
			args, _ := cli.Split("app")
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(cli.IsVisible(found)).To(Equal(visibleExpected))
		},
			Entry("nominal", &cli.Command{Name: "visible"}, "visible", true),
			Entry("visible", &cli.Command{Name: "visible", Options: cli.Visible}, "visible", true),
			Entry("implicitly hidden by name", &cli.Command{Name: "_hidden"}, "_hidden", false),
			Entry("disable implicitly hidden behavior (self)", &cli.Command{Name: "_hidden", Options: cli.DisableAutoVisibility}, "_hidden", true),
			Entry("disable implicitly hidden behavior of a command (parent)", &cli.Command{
				Name:    "parent",
				Options: cli.DisableAutoVisibility,
				Subcommands: []*cli.Command{
					{Name: "_hidden"},
				},
			}, "parent _hidden", true),
			Entry("disable implicitly hidden behavior of a flag (parent)", &cli.Command{
				Name:    "parent",
				Options: cli.DisableAutoVisibility,
				Flags: []*cli.Flag{
					{Name: "_hidden"},
				},
			}, "parent -_hidden", true),
			Entry("explicitly made visible implicitly hidden behavior", &cli.Command{Name: "_hidden", Options: cli.Visible}, "_hidden", true),
			Entry("hidden wins over visible", &cli.Command{Name: "hidden", Options: cli.Hidden | cli.Visible}, "hidden", false),
		)
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
			captured := cli.FromContext(act.ExecuteArgsForCall(0))

			Expect(captured.List("args")).To(Equal([]string{"-a", "-b"}))
		})

	})

	Describe("DisallowFlagsAfterArgs", func() {
		DescribeTable("causes flags after args error", func(arguments string) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "whitespace",
						Aliases: []string{"w"},
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
			args, _ := cli.Split(arguments)
			err := app.RunContext(context.TODO(), args)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("can't use -.+ after arguments")))
		},
			Entry("short", "app arg -w"),
			Entry("long", "app arg --whitespace"),
		)
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
			Expect(err).To(MatchError("done error"))
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
			captured := cli.FromContext(act.ExecuteArgsForCall(0))

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
				cli.Initialized(cmd)
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

	Describe("naming", func() {

		It("implicitly names args", func() {
			app := &cli.App{
				Args: []*cli.Arg{
					{},
				},
			}

			_, _ = app.Initialize(context.TODO())
			Expect(app.Args[0].Name).To(Equal("_1"))
		})

		var addFlagOrArg = func(option any) cli.Action {
			if flag, ok := option.(*cli.Flag); ok {
				return cli.AddFlag(flag)
			}
			return cli.AddArg(option.(*cli.Arg))
		}

		var (
			beInvalid = MatchError(ContainSubstring("not a valid name"))
		)

		DescribeTable("undefined behavior", func(option any) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "inuse", Aliases: []string{"alsoinuse"}},
				},
				Uses: addFlagOrArg(option),
			}

			_, err := app.Initialize(context.TODO())
			Expect(err).NotTo(HaveOccurred())
		},
			Entry(
				"arg duplicates flag alias", &cli.Arg{Name: "alsoinuse"}),
			Entry(
				"flag duplicates flag alias", &cli.Flag{Name: "alsoinuse"}),
		)

		DescribeTable("errors", func(option any, expected types.GomegaMatcher) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "inuse", Aliases: []string{"alsoinuse"}},
				},
				Args: []*cli.Arg{
					{Name: "arg"},
				},
				Uses: addFlagOrArg(option),
			}

			_, err := app.Initialize(context.TODO())
			Expect(err).To(expected)
		},
			Entry(
				"no name",
				&cli.Flag{},
				MatchError(ContainSubstring("flag at index #1 must have a name"))),
			Entry(
				"duplicate flag name",
				&cli.Flag{Name: "inuse"},
				MatchError(ContainSubstring(`duplicate name used: "inuse"`))),
			Entry(
				"duplicate arg name",
				&cli.Arg{Name: "arg"},
				MatchError(ContainSubstring(`duplicate name used: "arg"`))),
			Entry(
				"arg duplicates flag name",
				&cli.Arg{Name: "inuse"},
				MatchError(ContainSubstring(`duplicate name used: "inuse"`))),

			Entry("flag with dashes", &cli.Flag{Name: "flag-dash"}, Succeed()),
			Entry("flag with underscores", &cli.Flag{Name: "flag_under"}, Succeed()),
			Entry("flag with numeric", &cli.Flag{Name: "123"}, Succeed()),
			Entry("flag with special char @", &cli.Flag{Name: "@"}, Succeed()),
			Entry("flag with special char #", &cli.Flag{Name: "#"}, Succeed()),
			Entry("flag with special char *", &cli.Flag{Name: "*"}, Succeed()),
			Entry("flag with special char +", &cli.Flag{Name: "+"}, Succeed()),
			Entry("flag with special char :", &cli.Flag{Name: ":"}, Succeed()),
			Entry("flag with spaces", &cli.Flag{Name: "flag name with spaces"}, beInvalid),
			Entry("flag with invalid cahr", &cli.Flag{Name: "flag&flag"}, beInvalid),
		)
	})

})

var _ = Describe("HandleCommandNotFound", func() {

	Context("when a default handler is specified", func() {

		var (
			fn        func(*cli.Context, error) (*cli.Command, error)
			err       error
			arguments string = "app unknown --flag --option 3"

			existsAct *joeclifakes.FakeAction
		)

		JustBeforeEach(func() {
			existsAct = new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Uses: cli.HandleCommandNotFound(fn),
				Commands: []*cli.Command{
					{
						Name:   "exists",
						Action: existsAct,
						Flags: []*cli.Flag{
							{
								Name:  "flag",
								Value: new(bool),
							},
							{
								Name: "option",
							},
						},
					},
				},
				Stderr: io.Discard,
			}

			args, _ := cli.Split(arguments)
			err = app.RunContext(context.TODO(), args)
		})

		Context("when func specifies an existing command", func() {

			BeforeEach(func() {
				fn = func(c *cli.Context, err error) (*cli.Command, error) {
					cmd, _ := c.Command().Command("exists")
					return cmd, nil
				}
			})

			It("invokes selected command", func() {
				Expect(existsAct.ExecuteCallCount()).To(Equal(1))

				captured := cli.FromContext(existsAct.ExecuteArgsForCall(0))
				Expect(captured.Args()).To(Equal([]string{"unknown", "--flag", "--option", "3"}))
			})

			It("uses the default command handler to locate other commands", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	It("composes function call", func() {
		var fn1Called, fn2Called bool
		fn1 := func(*cli.Context, error) (*cli.Command, error) {
			fn1Called = true
			return nil, nil
		}
		fn2 := func(*cli.Context, error) (*cli.Command, error) {
			fn2Called = true
			return nil, nil
		}

		app := cli.App{
			Commands: []*cli.Command{
				{Name: "exists"},
			},
			Uses: cli.Pipeline(
				cli.HandleCommandNotFound(fn1),
				cli.HandleCommandNotFound(fn2),
			),
			Stderr: io.Discard,
		}

		args, _ := cli.Split("app unknown")
		_ = app.RunContext(context.TODO(), args)
		Expect(fn1Called).To(BeTrue())
		Expect(fn2Called).To(BeTrue())
	})

	It("inherits behavior from containing sub-command", func() {
		var inheritedCalled bool
		inheritedFn := func(*cli.Context, error) (*cli.Command, error) {
			inheritedCalled = true
			return nil, nil
		}

		app := cli.App{
			Commands: []*cli.Command{
				{
					Name: "sub",
					Subcommands: []*cli.Command{
						{Name: "dom"},
					},
				},
			},
			Uses: cli.Pipeline(
				cli.HandleCommandNotFound(inheritedFn),
			),
			Stderr: io.Discard,
		}

		args, _ := cli.Split("app sub unknown")
		err := app.RunContext(context.TODO(), args)
		Expect(err).To(HaveOccurred())
		Expect(inheritedCalled).To(BeTrue())
	})

	It("resets behavior when set to nil", func() {
		var fnCalled bool
		fn := func(*cli.Context, error) (*cli.Command, error) {
			fnCalled = true
			return nil, nil
		}

		app := cli.App{
			Commands: []*cli.Command{
				{
					Name: "sub",
				},
			},
			Uses: cli.Pipeline(
				cli.HandleCommandNotFound(fn),
				cli.HandleCommandNotFound(nil), // Should reset handling to default
			),
			Stderr: io.Discard,
		}

		args, _ := cli.Split("app rex")
		err := app.RunContext(context.TODO(), args)
		Expect(err).To(HaveOccurred())
		Expect(fnCalled).To(BeFalse())
	})

	It("resets inherited behavior when set to nil", func() {
		var inheritedCalled bool
		inheritedFn := func(*cli.Context, error) (*cli.Command, error) {
			inheritedCalled = true
			return nil, nil
		}

		app := cli.App{
			Commands: []*cli.Command{
				{
					Name: "sub",
					Subcommands: []*cli.Command{
						{Name: "dom"},
					},
					Uses: cli.Pipeline(
						cli.HandleCommandNotFound(nil),
					),
				},
			},
			Uses: cli.Pipeline(
				cli.HandleCommandNotFound(inheritedFn),
			),
			Stderr: io.Discard,
		}

		args, _ := cli.Split("app sub unknown")
		err := app.RunContext(context.TODO(), args)
		Expect(err).To(HaveOccurred())
		Expect(inheritedCalled).To(BeFalse())
		Expect(err).To(MatchError(`"unknown" is not a command`))
	})

})

var _ = Describe("ImplicitCommand", func() {

	It("invokes with the correct arguments", func() {
		act := new(joeclifakes.FakeAction)
		app := cli.App{
			Commands: []*cli.Command{
				{
					Name: "exec",
					Args: []*cli.Arg{
						{
							Name:  "cmd",
							NArg:  1,
							Value: new(string),
						},
						{
							Name:  "args",
							NArg:  cli.TakeUntilNextFlag,
							Value: new([]string),
						},
					},
					Flags: []*cli.Flag{
						{
							Name:  "f",
							Value: new(bool),
						},
					},
					Action: act,
				},
			},
			Uses: cli.ImplicitCommand("exec"),
		}

		args, _ := cli.Split("app tail /var/output/logs -f")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))

		captured := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(captured.Args()).To(Equal([]string{"exec", "tail", "/var/output/logs", "-f"}))
		Expect(app.Commands[0].Args[0].Value).To(PointTo(Equal("tail")))
		Expect(app.Commands[0].Args[1].Value).To(PointTo(Equal([]string{"/var/output/logs"})))
		Expect(app.Commands[0].Flags[0].Value).To(PointTo(BeTrue()))
	})
})
