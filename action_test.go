package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("middleware", func() {

	Describe("before", func() {
		var (
			captured  *cli.Context
			before    cli.Action
			flags     []*cli.Flag
			commands  []*cli.Command
			arguments []string
		)

		JustBeforeEach(func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name:     "app",
				Before:   before,
				Action:   act,
				Flags:    flags,
				Commands: commands,
			}
			err := app.RunContext(context.TODO(), arguments)
			Expect(err).NotTo(HaveOccurred())
			if act.ExecuteCallCount() > 0 {
				captured = act.ExecuteArgsForCall(0)
			}
		})

		Context("ContextValue", func() {
			type privateKey string

			BeforeEach(func() {
				arguments = []string{"app"}
				before = cli.ContextValue(privateKey("mykey"), "context value")
			})

			It("ContextValue can set and retrieve context value", func() {
				Expect(captured.Context.Value(privateKey("mykey"))).To(BeIdenticalTo("context value"))
			})

			It("ContextValue can set and retrieve context value via Value", func() {
				Expect(captured.Value(privateKey("mykey"))).To(BeIdenticalTo("context value"))
			})

			Context("when defined on a command", func() {

				var (
					beforeFlag, afterFlag, flagAct *joeclifakes.FakeAction
				)

				BeforeEach(func() {
					beforeFlag = new(joeclifakes.FakeAction)
					afterFlag = new(joeclifakes.FakeAction)
					flagAct = new(joeclifakes.FakeAction)
					arguments = []string{"app", "sub", "--flag=0"}
					commands = []*cli.Command{
						{
							Name: "sub",
							Uses: cli.ContextValue(privateKey("command"), "context value"),
							Flags: []*cli.Flag{
								{
									Name:   "flag",
									Before: beforeFlag,
									Action: flagAct,
									After:  afterFlag,
								},
							},
						},
					}
				})

				It("makes value available to flag action", func() {
					captured := beforeFlag.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to flag action", func() {
					captured := afterFlag.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to flag action", func() {
					captured := flagAct.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

			})
		})

		Context("SetValue", func() {
			BeforeEach(func() {
				arguments = []string{"app"}
				flags = []*cli.Flag{
					{
						Name:   "int",
						Value:  cli.Int(),
						Before: cli.SetValue("420"),
					},
				}
			})

			It("can set and retrieve value", func() {
				Expect(captured.Value("int")).To(Equal(420))
			})
		})

		Describe("No", func() {
			var (
				flagAct *joeclifakes.FakeAction
			)

			BeforeEach(func() {
				initial := true
				flagAct = new(joeclifakes.FakeAction)
				flags = []*cli.Flag{
					{
						Name:    "flag",
						Aliases: []string{"f"},
						Options: cli.No,
						Value:   &initial,
						Action:  flagAct,
					},
				}
				arguments = []string{"app", "--no-flag"}
			})

			It("sets negative value", func() {
				Expect(captured.Value("flag")).To(BeFalse())
			})

			It("creates secondary flag", func() {
				s, _ := captured.LookupFlag("no-flag")
				Expect(s.Name).To(Equal("no-flag"))
			})

			It("sets custom synopsis on original flag", func() {
				s, _ := captured.LookupFlag("flag")
				Expect(s.Synopsis()).To(Equal("-f, --[no-]flag"))
			})

			Context("when invoking mirror flag", func() {
				BeforeEach(func() {
					arguments = []string{"app", "--no-flag"}
				})

				It("invokes action", func() {
					Expect(flagAct.ExecuteCallCount()).To(Equal(1))
				})

				It("action has expected value", func() {
					context := flagAct.ExecuteArgsForCall(0)
					Expect(context.Value("")).To(BeFalse())
				})

				It("action has expected name", func() {
					context := flagAct.ExecuteArgsForCall(0)
					Expect(context.Name()).To(Equal("flag"))
				})

			})

			Context("when invoking flag", func() {
				BeforeEach(func() {
					arguments = []string{"app", "--flag"}
				})

				It("invokes action", func() {
					Expect(flagAct.ExecuteCallCount()).To(Equal(1))
				})

				It("has expected value", func() {
					context := flagAct.ExecuteArgsForCall(0)
					Expect(context.Value("")).To(BeTrue())
				})

			})
		})

	})

	Describe("action", func() {
		var (
			flags     []*cli.Flag
			arguments string
		)

		JustBeforeEach(func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name:   "app",
				Action: act,
				Flags:  flags,
			}
			args, _ := cli.Split(arguments)
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("WorkingDirectory", func() {
			var original string

			AfterEach(func() {
				os.Chdir(original)
			})

			BeforeEach(func() {
				original, _ = os.Getwd()
				arguments = "app --dir=/usr"
			})

			Context("string flag", func() {
				BeforeEach(func() {
					flags = []*cli.Flag{
						{
							Name:    "dir",
							Value:   cli.String(),
							Options: cli.WorkingDirectory,
						},
					}
				})

				It("WorkingDirectory sets the working directory", func() {
					Expect(os.Getwd()).To(Equal("/usr"))
				})
			})

			Context("when unset", func() {
				BeforeEach(func() {
					arguments = "app"
					flags = []*cli.Flag{
						{
							Name:    "dir",
							Value:   cli.String(),
							Options: cli.WorkingDirectory,
						},
					}
				})

				It("WorkingDirectory does nothing", func() {
					Expect(os.Getwd()).To(Equal(original))
				})

				// It also generates no error (this is checked in JustBeforeEach for the context)
			})

			Context("File flag", func() {
				BeforeEach(func() {
					flags = []*cli.Flag{
						{
							Name:    "dir",
							Value:   &cli.File{},
							Options: cli.WorkingDirectory,
						},
					}
				})

				It("WorkingDirectory sets the working directory", func() {
					Expect(os.Getwd()).To(Equal("/usr"))
				})
			})
		})
	})

	Describe("initialization", func() {
		var (
			captured    *cli.Context
			initializer cli.Action
		)
		JustBeforeEach(func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Uses:   initializer,
						Action: act,
					},
				},
			}
			app.RunContext(context.TODO(), []string{"app", "sub"})
			Expect(act.ExecuteCallCount()).To(Equal(1))
			captured = act.ExecuteArgsForCall(0)
		})

		Context("Data", func() {
			BeforeEach(func() {
				initializer = cli.Data("ok", "money")
			})

			It("can set data", func() {
				Expect(captured.Command().Data).To(HaveKeyWithValue("ok", "money"))
			})
		})

		Context("Category", func() {
			BeforeEach(func() {
				initializer = cli.Category("bags")
			})

			It("can set category", func() {
				Expect(captured.Command().Category).To(Equal("bags"))
			})
		})
	})

})

var _ = Describe("Uses", func() {

	DescribeTable("timing examples",
		func(arguments string, actualApp func(cli.Action) *cli.App) {
			var actual = new(struct {
				IsInitializing bool
				CallCount      int
			})
			handler := cli.ActionFunc(func(c *cli.Context) error {
				actual.IsInitializing = c.IsInitializing()
				actual.CallCount += 1
				return nil
			})
			app := actualApp(handler)
			args, _ := cli.Split(arguments)
			_ = app.RunContext(context.TODO(), args)
			Expect(actual.CallCount).To(Equal(1))
			Expect(actual.IsInitializing).To(BeTrue())
		},
		Entry("app", "app", func(h cli.Action) *cli.App {
			return &cli.App{
				Uses: h,
			}
		}),
		Entry("command", "app r", func(h cli.Action) *cli.App {
			return &cli.App{
				Commands: []*cli.Command{
					{
						Name: "r",
						Uses: h,
					},
				},
			}
		}),
		Entry("arg", "app a", func(h cli.Action) *cli.App {
			return &cli.App{
				Args: []*cli.Arg{
					{
						Name: "r",
						Uses: h,
					},
				},
			}
		}),
		Entry("flag", "app -f", func(h cli.Action) *cli.App {
			return &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
						Uses:  h,
					},
				},
			}
		}),
	)

	Context("concurrent modifications", func() {

		// When a flag, etc. is added during our initialization pass, we also
		// process its initialization

		DescribeTable("new values are initialized",
			func(uses func(cli.Action) cli.Action) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Flags: []*cli.Flag{
						{Name: "1"},
						{Name: "2"},
					},
					Uses:   uses(act),
					Stderr: ioutil.Discard,
				}
				_ = app.RunContext(context.TODO(), []string{"app"})
				Expect(act.ExecuteCallCount()).To(Equal(1))
			},
			Entry("add Flag to end", func(act cli.Action) cli.Action {
				return cli.AddFlag(&cli.Flag{Name: "f", Uses: act})
			}),
			Entry("add Arg to end", func(act cli.Action) cli.Action {
				return cli.AddArg(&cli.Arg{Name: "f", Uses: act})
			}),
			Entry("add Command to end", func(act cli.Action) cli.Action {
				return cli.AddCommand(&cli.Command{Name: "f", Uses: act})
			}),
		)
	})
})

var _ = Describe("Action", func() {
	DescribeTable("timing examples",
		func(arguments string, actualApp func(cli.Action) *cli.App) {
			var actual = new(struct {
				IsInitializing bool
				IsAction       bool
				CallCount      int
			})
			handler := cli.ActionFunc(func(c *cli.Context) error {
				actual.IsInitializing = c.IsInitializing()
				actual.IsAction = c.Timing() == cli.ActionTiming
				actual.CallCount += 1
				return nil
			})
			app := actualApp(handler)
			args, _ := cli.Split(arguments)
			_ = app.RunContext(context.TODO(), args)
			Expect(actual.CallCount).To(Equal(1))
			Expect(actual.IsInitializing).To(BeFalse())
			Expect(actual.IsAction).To(BeTrue())
		},
		Entry("app", "app", func(h cli.Action) *cli.App {
			return &cli.App{
				Uses: func(c *cli.Context) { c.Action(h) },
			}
		}),
		Entry("command", "app r", func(h cli.Action) *cli.App {
			return &cli.App{
				Commands: []*cli.Command{
					{
						Name: "r",
						Uses: func(c *cli.Context) { c.Action(h) },
					},
				},
			}
		}),
		Entry("arg", "app a", func(h cli.Action) *cli.App {
			return &cli.App{
				Args: []*cli.Arg{
					{
						Name: "r",
						Uses: func(c *cli.Context) { c.Action(h) },
					},
				},
			}
		}),
		Entry("flag", "app -f", func(h cli.Action) *cli.App {
			return &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
						Uses:  func(c *cli.Context) { c.Action(h) },
					},
				},
			}
		}),
	)
})

var _ = Describe("ProvideValueInitializer", func() {
	It("invokes a context scope on the value", func() {
		setup := cli.Setup{
			Uses:   new(joeclifakes.FakeAction),
			Before: new(joeclifakes.FakeAction),
			Action: new(joeclifakes.FakeAction),
			After:  new(joeclifakes.FakeAction),
		}
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "r",
					Uses: cli.ProvideValueInitializer(nil, "myname", setup),
				},
			},
		}
		args, _ := cli.Split("app 0")
		_ = app.RunContext(context.TODO(), args)

		Expect(setup.Uses.(*joeclifakes.FakeAction).ExecuteCallCount()).To(Equal(1))
		Expect(setup.Before.(*joeclifakes.FakeAction).ExecuteCallCount()).To(Equal(1))
		Expect(setup.After.(*joeclifakes.FakeAction).ExecuteCallCount()).To(Equal(1))
		Expect(setup.Action.(*joeclifakes.FakeAction).ExecuteCallCount()).To(Equal(1))

		Expect(setup.Uses.(*joeclifakes.FakeAction).ExecuteArgsForCall(0).Name()).To(Equal("<-myname>"))
	})
})

var _ = Describe("Before", func() {

	DescribeTable("timing examples",
		func(arguments string, actualApp func(cli.Action) *cli.App) {
			var actual = new(struct {
				IsInitializing bool
				IsBefore       bool
				CallCount      int
			})
			handler := cli.ActionFunc(func(c *cli.Context) error {
				actual.IsInitializing = c.IsInitializing()
				actual.IsBefore = c.IsBefore()
				actual.CallCount += 1
				return nil
			})
			app := actualApp(handler)
			args, _ := cli.Split(arguments)
			_ = app.RunContext(context.TODO(), args)
			Expect(actual.CallCount).To(Equal(1))
			Expect(actual.IsInitializing).To(BeFalse())
			Expect(actual.IsBefore).To(BeTrue())
		},
		Entry("app", "app", func(h cli.Action) *cli.App {
			return &cli.App{
				Uses: cli.Before(h),
			}
		}),
		Entry("command", "app r", func(h cli.Action) *cli.App {
			return &cli.App{
				Commands: []*cli.Command{
					{
						Name: "r",
						Uses: cli.Before(h),
					},
				},
			}
		}),
		Entry("arg", "app a", func(h cli.Action) *cli.App {
			return &cli.App{
				Args: []*cli.Arg{
					{
						Name: "r",
						Uses: cli.Before(h),
					},
				},
			}
		}),
		Entry("flag", "app -f", func(h cli.Action) *cli.App {
			return &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
						Uses:  cli.Before(h),
					},
				},
			}
		}),
	)
})

var _ = Describe("Do", func() {

	var (
		handler cli.Action
		err     error
	)

	JustBeforeEach(func() {
		app := &cli.App{
			Commands: []*cli.Command{
				{
					Name: "r",
					Action: func(c *cli.Context) error {
						return c.Do(handler)
					},
				},
			},
		}

		args, _ := cli.Split("app r")
		err = app.RunContext(context.TODO(), args)
	})

	// When Do is called on an action that has timing specified, it should
	// be run later
	Context("when After timing", func() {
		var actual = new(struct {
			IsAfter   bool
			CallCount int
		})

		BeforeEach(func() {
			handler = cli.After(cli.ActionFunc(func(c *cli.Context) error {
				actual.IsAfter = c.IsAfter()
				actual.CallCount += 1
				return nil
			}))
		})

		It("applies After timing if specified", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(actual.CallCount).To(Equal(1))
			Expect(actual.IsAfter).To(BeTrue())
		})
	})

	Context("when Before timing", func() {
		BeforeEach(func() {
			handler = cli.Before(new(joeclifakes.FakeAction))
		})

		It("applies Before timing if specified", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("too late to exec action"))
		})
	})
})

var _ = Describe("After", func() {
	DescribeTable("timing examples",
		func(arguments string, actualApp func(cli.Action) *cli.App) {
			var actual = new(struct {
				IsInitializing bool
				IsAfter        bool
				CallCount      int
			})
			handler := cli.ActionFunc(func(c *cli.Context) error {
				actual.IsInitializing = c.IsInitializing()
				actual.IsAfter = c.IsAfter()
				actual.CallCount += 1
				return nil
			})
			app := actualApp(handler)
			args, _ := cli.Split(arguments)
			_ = app.RunContext(context.TODO(), args)
			Expect(actual.CallCount).To(Equal(1))
			Expect(actual.IsInitializing).To(BeFalse())
			Expect(actual.IsAfter).To(BeTrue())
		},
		Entry("app", "app", func(h cli.Action) *cli.App {
			return &cli.App{
				Uses: cli.After(h),
			}
		}),
		Entry("command", "app r", func(h cli.Action) *cli.App {
			return &cli.App{
				Commands: []*cli.Command{
					{
						Name: "r",
						Uses: cli.After(h),
					},
				},
			}
		}),
		Entry("arg", "app a", func(h cli.Action) *cli.App {
			return &cli.App{
				Args: []*cli.Arg{
					{
						Name: "r",
						Uses: cli.After(h),
					},
				},
			}
		}),
		Entry("flag", "app -f", func(h cli.Action) *cli.App {
			return &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
						Uses:  cli.After(h),
					},
				},
			}
		}),
	)
})

var _ = Describe("ActionOf", func() {

	var called bool
	act := func() { called = true }

	DescribeTable("examples",
		func(thunk interface{}) {
			var handler cli.Action
			Expect(func() {
				handler = cli.ActionOf(thunk)
			}).NotTo(Panic())

			called = false
			handler.Execute(&cli.Context{})
			Expect(called).To(BeTrue())
		},
		Entry("func(*cli.Context) error", func(*cli.Context) error { act(); return nil }),
		Entry("func(*cli.Context)", func(*cli.Context) { act() }),
		Entry("func(context.Context) error", func(context.Context) error { act(); return nil }),
		Entry("func(context.Context)", func(context.Context) { act() }),
		Entry("func() error", func() error { act(); return nil }),
	)
})

var _ = Describe("events", func() {
	DescribeTable("execution order of events",
		func(arguments string, expected types.GomegaMatcher) {
			result := make([]string, 0)
			event := func(name string) cli.Action {
				return cli.ActionOf(func() {
					result = append(result, name)
				})
			}
			app := &cli.App{
				Before: event("before app"),
				Action: event("app"),
				After:  event("after app"),
				Uses: cli.Pipeline(
					cli.HookBefore("*", event("app hook before")),
					cli.HookAfter("*", event("app hook after")),
				),
				Flags: []*cli.Flag{
					{
						Name:   "global",
						Value:  cli.Bool(),
						Before: event("before --global"),
						Action: event("--global"),
						After:  event("after --global"),
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Before: event("before sub"),
						Action: event("sub"),
						After:  event("after sub"),
						Flags: []*cli.Flag{
							{
								Name:   "local",
								Value:  cli.Bool(),
								Before: event("before --local"),
								Action: event("--local"),
								After:  event("after --local"),
							},
						},
						Subcommands: []*cli.Command{
							{
								Name:   "dom",
								Before: event("before dom"),
								After:  event("after dom"),
								Action: event("dom"),
								Flags: []*cli.Flag{
									{
										Name:   "nest",
										Value:  cli.Bool(),
										Before: event("before --nest"),
										Action: event("--nest"),
									},
								},
								Args: []*cli.Arg{
									{
										Name:   "a",
										Before: event("before a"),
										Action: event("a"),
									},
								},
							},
						},
					},
				},
			}
			args, _ := cli.Split(arguments)
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(expected)
		},
		Entry(
			"persistent flags always run before subcommand flags",
			"app sub --local --global", // despite being used after, before --global is run first
			ContainElements("before --global", "before --local"),
		),
		Entry(
			"sub-command call",
			"app sub",
			And(
				ContainElements("before app", "before sub"),
				ContainElements("after sub", "after app"),
			),
		),
		Entry(
			"nested command persistent flag is called",
			"app sub --global ",
			ContainElements("before --global", "--global"),
		),
		Entry(
			"doubly nested command before hooks",
			"app sub dom",
			ContainElements("before app", "before sub", "before dom", "app hook before", "dom"),
		),
		Entry(
			"doubly nested command after hooks",
			"app sub dom",
			ContainElements("dom", "after dom", "after sub", "app hook after", "after app"),
		),
	)
})

var _ = Describe("Pipeline", func() {
	It("invokes all actions in pipeline", func() {
		act1 := new(joeclifakes.FakeAction)
		act2 := new(joeclifakes.FakeAction)

		pipe := cli.Pipeline().Append(act1).Append(act2)
		pipe.Execute(&cli.Context{})

		Expect(act1.ExecuteCallCount()).To(Equal(1))
		Expect(act2.ExecuteCallCount()).To(Equal(1))
	})
})

var _ = Describe("HandleSignal", Ordered, func() {

	BeforeAll(func() {
		signal.Reset(os.Interrupt) // remove ginkgo signal handling
	})

	It("can use context done", func() {
		proc, err := os.FindProcess(os.Getpid())
		if err != nil {
			Fail(err.Error())
		}

		// Simulate ^C being pressed
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt)

		go func() {
			<-sigc
			signal.Stop(sigc)
		}()

		app := &cli.App{
			Name: "any",
			Uses: cli.HandleSignal(os.Interrupt),
			Before: func() {
				// Simulate user key press when app is ready
				proc.Signal(os.Interrupt)
			},
			Action: func(c context.Context) error {
				select {
				case <-time.After(2 * time.Second):
					return fmt.Errorf("expected signal to be handled within action before timeout")
				case <-c.Done():
					return fmt.Errorf("expected output error")
				}
				return nil
			},
		}

		err = app.RunContext(context.Background(), []string{"app"})

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("expected output error"))
	})
})

var _ = Describe("Timeout", func() {

	It("can use context done", func() {
		app := &cli.App{
			Name: "any",
			Uses: cli.Timeout(200 * time.Millisecond),
			Action: func(c context.Context) error {
				select {
				case <-time.After(1 * time.Second):
					return fmt.Errorf("expected proper timeout to be handled within action")
				case <-c.Done():
					return fmt.Errorf("expected output error")
				}
				return nil
			},
		}

		err := app.RunContext(context.Background(), []string{"app"})

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("expected output error"))
	})
})

var _ = Describe("Recover", func() {

	It("will print out the debug stack", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "any",
			Stderr: &capture,
			Action: cli.Recover(cli.ActionFunc(func(c *cli.Context) error {
				panic("panic in action")
			})),
		}

		err := app.RunContext(context.Background(), []string{"app"})

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("panic in action"))
		Expect(capture.String()).To(ContainSubstring("runtime/debug.Stack()"))
	})
})

var _ = Describe("FlagSetup", func() {
	It("is called to apply to the flag", func() {
		var called bool
		app := &cli.App{
			Name: "any",
			Flags: []*cli.Flag{
				{
					Name: "ok",
					Uses: cli.FlagSetup(func(f *cli.Flag) {
						Expect(f.Name).To(Equal("ok"))
						called = true
					}),
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeTrue())
	})

	It("is ignored on non-Flag", func() {
		var called bool
		app := &cli.App{
			Name: "any",
			Args: []*cli.Arg{
				{
					Name: "ok",
					Uses: cli.FlagSetup(func(f *cli.Flag) {
						called = true
					}),
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeFalse())
	})
})

var _ = Describe("ArgSetup", func() {
	It("is called to apply to the Arg", func() {
		var called bool
		app := &cli.App{
			Name: "any",
			Args: []*cli.Arg{
				{
					Name: "ok",
					Uses: cli.ArgSetup(func(a *cli.Arg) {
						Expect(a.Name).To(Equal("ok"))
						called = true
					}),
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeTrue())
	})

	It("is ignored on non-Arg", func() {
		var called bool
		app := &cli.App{
			Name: "any",
			Flags: []*cli.Flag{
				{
					Name: "ok",
					Uses: cli.ArgSetup(func(f *cli.Arg) {
						called = true
					}),
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeFalse())
	})
})

var _ = Describe("CommandSetup", func() {
	It("is called to apply to the Command", func() {
		var called bool
		app := &cli.App{
			Name:   "any",
			Stderr: ioutil.Discard,
			Commands: []*cli.Command{
				{
					Name: "ok",
					Uses: cli.CommandSetup(func(c *cli.Command) {
						Expect(c.Name).To(Equal("ok"))
						called = true
					}),
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeTrue())
	})

	It("can apply to inherited command scope", func() {
		var called bool
		var which string
		app := &cli.App{
			Name: "root",
			Flags: []*cli.Flag{
				{
					Name: "ok",
					Uses: cli.CommandSetup(func(c *cli.Command) {
						called = true
						which = c.Name
					}),
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(called).To(BeTrue())
		Expect(which).To(Equal("root"))
	})
})

var _ = Describe("PreventSetup", func() {
	var (
		flagSetup = func(thunk func()) cli.Action {
			return cli.FlagSetup(func(*cli.Flag) {
				thunk()
			})
		}
		argSetup = func(thunk func()) cli.Action {
			return cli.ArgSetup(func(*cli.Arg) {
				thunk()
			})
		}
		commandSetup = func(thunk func()) cli.Action {
			return cli.CommandSetup(func(*cli.Command) {
				thunk()
			})
		}
	)

	DescribeTable("entry", func(create func(func()) *cli.App) {
		thunk := func() {
			Fail("should not call setup method")
		}
		app := create(thunk)

		args, _ := cli.Split("app")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
	},
		Entry("flag", func(t func()) *cli.App {
			return &cli.App{
				Flags: []*cli.Flag{
					{
						Options: cli.PreventSetup,
						Uses:    flagSetup(t),
					},
				},
			}
		}),
		Entry("arg", func(t func()) *cli.App {
			return &cli.App{
				Args: []*cli.Arg{
					{
						Options: cli.PreventSetup,
						Uses:    argSetup(t),
					},
				},
			}
		}),
		Entry("command", func(t func()) *cli.App {
			return &cli.App{
				Stderr: ioutil.Discard,
				Commands: []*cli.Command{
					{
						Options: cli.PreventSetup,
						Name:    "sub",
						Uses:    commandSetup(t),
					},
				},
			}
		}),
		Entry("flag recursive", func(t func()) *cli.App {
			return &cli.App{
				Options: cli.PreventSetup,
				Flags: []*cli.Flag{
					{
						Uses: flagSetup(t),
					},
				},
			}
		}),
		Entry("flag via command recursive", func(t func()) *cli.App {
			return &cli.App{
				Options: cli.PreventSetup,
				Stderr:  ioutil.Discard,
				Commands: []*cli.Command{
					{
						Flags: []*cli.Flag{
							{
								Uses: flagSetup(t),
							},
						},
					},
				},
			}
		}),
	)
})

var _ = Describe("Customize", func() {
	DescribeTable("examples",
		func(uses cli.Action, expected func(app *cli.App)) {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "flag",
					},
				},
				Args: []*cli.Arg{
					{
						Name:  "arg",
						Value: &cli.Expression{},
						Uses:  cli.Data("arg", 1),
					},
				},
				Commands: []*cli.Command{
					{
						Name: "sub",
						Flags: []*cli.Flag{
							{
								Name: "flag",
							},
						},
					},
					{
						Name:  "dom",
						Flags: []*cli.Flag{},
					},
				},
				Uses: uses,
			}
			app.Initialize(context.Background())
			expected(app)
		},

		Entry("customizes a flag",
			cli.Customize("--flag", cli.Data("match", "flag")),
			func(app *cli.App) {
				Expect(app.Flags[0].Data).To(HaveKeyWithValue("match", "flag"))
			}),

		Entry("customizes flags recursively within commands",
			cli.Customize("--flag", cli.Data("match", "flag")),
			func(app *cli.App) {
				Expect(app.Commands[0].Flags[0].Data).To(
					HaveKeyWithValue("match", "flag"),
				)
			}),
	)

	It("flag can customize itself", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: cli.Customize("", cli.FlagSetup(func(f *cli.Flag) {
						f.Data["ok"] = "2"
					})),
				},
			},
		}
		app.Initialize(context.Background())
		Expect(app.Flags[0].Data).To(HaveKeyWithValue("ok", "2"))
	})
})
