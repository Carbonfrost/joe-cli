package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/spf13/afero"
)

var _ = Describe("timings", func() {

	Describe("before", func() {
		var (
			captured  *cli.Context
			before    cli.Action
			uses      cli.Action
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
				Uses:     uses,
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
				uses = nil
				arguments = []string{"app"}
				before = cli.ContextValue(privateKey("mykey"), "context value")
			})

			It("ContextValue can set and retrieve context value", func() {
				Expect(captured.Context.Value(privateKey("mykey"))).To(BeIdenticalTo("context value"))
			})

			It("ContextValue can set and retrieve context value via Value", func() {
				Expect(captured.Value(privateKey("mykey"))).To(BeIdenticalTo("context value"))
			})

			Context("when defined on the app", func() {

				var (
					beforeFlag, afterFlag, flagAct, commandUses *joeclifakes.FakeAction
				)

				BeforeEach(func() {
					uses = cli.ContextValue(privateKey("app"), "has value")
					beforeFlag = new(joeclifakes.FakeAction)
					afterFlag = new(joeclifakes.FakeAction)
					flagAct = new(joeclifakes.FakeAction)
					commandUses = new(joeclifakes.FakeAction)

					arguments = []string{"app", "--flag=0"}
					flags = []*cli.Flag{
						{
							Name:   "flag",
							Before: beforeFlag,
							After:  afterFlag,
							Action: flagAct,
						},
					}
					commands = []*cli.Command{
						{
							Name: "non",
							Uses: commandUses,
						},
					}
				})

				It("makes value available to before flag action", func() {
					captured := beforeFlag.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("app"))).To(BeIdenticalTo("has value"))
				})

				It("makes value available to after flag action", func() {
					captured := afterFlag.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("app"))).To(BeIdenticalTo("has value"))
				})

				It("makes value available to flag action", func() {
					captured := flagAct.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("app"))).To(BeIdenticalTo("has value"))
				})

				It("makes value available to command uses action", func() {
					captured := commandUses.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("app"))).To(BeIdenticalTo("has value"))
				})
			})

			Context("when defined on a command", func() {

				var (
					beforeFlag, afterFlag, flagAct, afterUses, beforeCommand, afterCommand, commandAct *joeclifakes.FakeAction
				)

				BeforeEach(func() {
					beforeFlag = new(joeclifakes.FakeAction)
					afterFlag = new(joeclifakes.FakeAction)
					flagAct = new(joeclifakes.FakeAction)
					afterUses = new(joeclifakes.FakeAction)
					beforeCommand = new(joeclifakes.FakeAction)
					afterCommand = new(joeclifakes.FakeAction)
					commandAct = new(joeclifakes.FakeAction)
					arguments = []string{"app", "sub", "--flag=0"}
					flags = nil
					commands = []*cli.Command{
						{
							Name: "sub",
							Uses: cli.Pipeline(
								cli.ContextValue(privateKey("command"), "context value"),
								afterUses,
							),
							Before: beforeCommand,
							After:  afterCommand,
							Action: commandAct,
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

				It("makes value available to subsequent uses", func() {
					captured := afterUses.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to before", func() {
					captured := beforeCommand.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to after", func() {
					captured := afterCommand.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to command action", func() {
					captured := commandAct.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to before flag action", func() {
					captured := beforeFlag.ExecuteArgsForCall(0)
					Expect(captured.Value(privateKey("command"))).To(BeIdenticalTo("context value"))
				})

				It("makes value available to after flag action", func() {
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
				uses = nil
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
				uses = nil
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
				SkipOnWindows()
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

				Context("when unset File", func() {
					BeforeEach(func() {
						arguments = "app"
					})

					It("WorkingDirectory does nothing", func() {
						Expect(os.Getwd()).To(Equal(original))
					})

					// It also generates no error (this is checked in JustBeforeEach for the context)
				})

				Context("when set to blank", func() {
					BeforeEach(func() {
						arguments = "app --dir="
					})

					It("WorkingDirectory does nothing", func() {
						Expect(os.Getwd()).To(Equal(original))
					})

					// It also generates no error (this is checked in JustBeforeEach for the context)
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

	It("ensures that validation runs before other Before funcs", func() {
		var (
			events []string
			stub   = func(evt string) cli.ActionFunc {
				return func(c *cli.Context) error {
					events = append(events, evt)
					return nil
				}
			}
		)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "f",
					Uses: cli.Pipeline(
						cli.Before(stub("before")),
						cli.At(cli.ValidatorTiming, stub("validator")),
						cli.At(cli.ImplicitValueTiming, stub("implicitValue")),
					),
				},
			},
		}

		args, _ := cli.Split("app -f S")
		_ = app.RunContext(context.TODO(), args)
		Expect(events).To(Equal([]string{"validator", "before", "implicitValue"}))
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
					Args: []*cli.Arg{
						{Name: "A"},
					},
					Uses:   uses(act),
					Stderr: io.Discard,
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
			Entry("add Flag at beginning", func(act cli.Action) cli.Action {
				return cli.ActionFunc(func(c *cli.Context) error {
					c.Command().Flags = append([]*cli.Flag{{Name: "f", Uses: act}}, c.Command().Flags...)
					return nil
				})
			}),
			Entry("add Arg at beginning", func(act cli.Action) cli.Action {
				return cli.ActionFunc(func(c *cli.Context) error {
					c.Command().Args = append([]*cli.Arg{{Name: "f", Uses: act}}, c.Command().Args...)
					return nil
				})
			}),
		)

		Describe("RemoveArg", func() {

			DescribeTable("examples", func(name interface{}, expected []string) {
				var actual []string
				app := &cli.App{
					Args: []*cli.Arg{
						{Name: "1"},
						{Name: "2"},
						{Name: "3"},
						{Name: "4"},
					},
					Action: func(c *cli.Context) {
						args := c.Command().Args
						actual = make([]string, len(args))
						for i, a := range args {
							actual[i] = a.Name
						}
					},
					Uses:   cli.RemoveArg(name),
					Stderr: io.Discard,
				}
				err := app.RunContext(context.TODO(), []string{"app"})
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			},
				Entry("by name", "3", []string{"1", "2", "4"}),
				Entry("by decorated name", "<3>", []string{"1", "2", "4"}),
				Entry("by index", 1, []string{"1", "3", "4"}),
				Entry("by negative index", -1, []string{"1", "2", "3"}),
				Entry("by negative index", -2, []string{"1", "2", "4"}),
				Entry("by negative index", -4, []string{"2", "3", "4"}),
			)
		})

		Describe("RemoveFlag", func() {

			DescribeTable("examples", func(name interface{}, expected []string) {
				var actual []string
				app := &cli.App{
					Flags: []*cli.Flag{
						{Name: "1"},
						{Name: "2"},
						{Name: "3"},
						{Name: "4"},
					},
					Action: func(c *cli.Context) {
						flags := c.Command().Flags
						actual = make([]string, 0, len(flags))
						for _, f := range flags {
							if f.Name == "help" || f.Name == "version" || f.Name == "zsh-completion" {
								continue
							}
							actual = append(actual, f.Name)
						}
					},
					Uses:   cli.RemoveFlag(name),
					Stderr: io.Discard,
				}
				err := app.RunContext(context.TODO(), []string{"app"})
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			},
				Entry("by name", "3", []string{"1", "2", "4"}),
				Entry("by decorated name", "-3", []string{"1", "2", "4"}),
			)
		})
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

var _ = Describe("Required", func() {

	It("returns an error if unspecified", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:    "f",
					Value:   new(string),
					Options: cli.Required,
				},
			},
			Action: func() {},
		}
		err := app.RunContext(context.TODO(), []string{"app"})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("-f is required and must be specified"))
	})
})

var _ = Describe("ValidatorFunc", func() {
	It("invokes the function", func() {
		e := fmt.Errorf("validator err")
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "r",
					Uses: cli.ValidatorFunc(func(raw []string) error { return e }),
				},
			},
		}
		args, _ := cli.Split("app 0")

		err := app.RunContext(context.TODO(), args)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("validator err"))
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
			Expect(err).To(MatchError("too late for requested action timing"))
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
		Entry("middleware: func(Action) Action", func(cli.Action) cli.Action { act(); return nil }),
		Entry("middleware: func(*cli.Context, Action) error", func(*cli.Context, cli.Action) error { act(); return nil }),
	)

	It("invokes the context action", func() {
		ctx := &cli.Context{}
		var called int
		handlers := []cli.Action{
			cli.ActionOf(func(c context.Context) {
				called++
				Expect(c).To(BeIdenticalTo(ctx))
			}), cli.ActionOf(func(c context.Context) error {
				Expect(c).To(BeIdenticalTo(ctx))
				called++
				return nil
			}),
		}
		for _, handler := range handlers {
			handler.Execute(ctx)
		}

		Expect(called).To(Equal(2))
	})

	DescribeTable("errors",
		func(thunk interface{}) {
			called = false

			Expect(func() { cli.ActionOf(thunk) }).To(Panic())
			Expect(called).To(BeFalse())
		},
		Entry("unknown type", func(int) error { act(); return nil }),
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
			"persistent flags always run before sub-command flags",
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

	It("delegates to middleware when it exists", func() {
		act1 := new(joeclifakes.FakeMiddleware)
		act2 := new(joeclifakes.FakeAction)

		pipe := cli.Pipeline().Append(act1).Append(act2)
		pipe.Execute(&cli.Context{})

		Expect(act1.ExecuteWithNextCallCount()).To(Equal(1))
	})

	It("invoking middleware calls next", func() {
		act1 := new(joeclifakes.FakeMiddleware)
		act2 := new(joeclifakes.FakeAction)

		act1.ExecuteWithNextStub = func(_ *cli.Context, a cli.Action) error {
			return a.Execute(nil)
		}

		pipe := cli.Pipeline().Append(act1).Append(act2)
		pipe.Execute(&cli.Context{})

		Expect(act2.ExecuteCallCount()).To(Equal(1))
	})

	It("invoking middleware calls next two", func() {
		act1 := new(joeclifakes.FakeMiddleware)
		act2 := new(joeclifakes.FakeAction)
		act3 := new(joeclifakes.FakeAction)

		act1.ExecuteWithNextStub = func(_ *cli.Context, a cli.Action) error {
			return a.Execute(nil)
		}
		act2.ExecuteStub = func(_ *cli.Context) error {
			return nil
		}

		pipe := cli.Pipeline().Append(act1).Append(act2).Append(act3)
		pipe.Execute(&cli.Context{})

		Expect(act3.ExecuteCallCount()).To(Equal(1))
	})

	It("flattens nested pipelines and invokes in order", func() {
		var called []string
		var makeAction = func(name string) cli.ActionFunc {
			return func(*cli.Context) error {
				called = append(called, name)
				return nil
			}
		}

		var act1 cli.ActionPipeline
		act2 := makeAction("2")
		act1a := makeAction("1a")
		act1b := makeAction("1b")

		act1 = cli.ActionPipeline([]cli.Action{act1a, act1b})

		pipe := cli.Pipeline(cli.ActionPipeline([]cli.Action{act1, act2}))
		pipe.Execute(nil)

		Expect(called).To(Equal([]string{"1a", "1b", "2"}))
		Expect(pipe).To(HaveLen(3))
	})
})

var _ = Describe("HandleSignal", Ordered, func() {

	BeforeAll(func() {
		SkipOnWindows()
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

	It("can be used as middleware", func() {
		app := &cli.App{
			Name:   "any",
			Stderr: io.Discard,
			Action: cli.Pipeline(
				cli.Recover,
				func() {
					// this intervening action is fine
				},
				func() {
					panic("panic in action")
				},
			),
		}

		err := app.RunContext(context.Background(), []string{"app"})

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("panic in action"))
	})
})

var _ = Describe("SuppressError", func() {

	It("will print out the debug stack", func() {
		var capture bytes.Buffer
		var called bool
		app := &cli.App{
			Name:   "any",
			Stderr: &capture,
			Action: cli.SuppressError(cli.ActionFunc(func(c *cli.Context) error {
				called = true
				return fmt.Errorf("not me")
			})),
		}

		err := app.RunContext(context.Background(), []string{"app"})

		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
	})

	It("can be used as middleware", func() {
		var called bool
		app := &cli.App{
			Name:   "any",
			Stderr: io.Discard,
			Action: cli.Pipeline(
				cli.SuppressError,
				func() {
					// this intervening action is fine
				},
				func() error {
					called = true
					return fmt.Errorf("not me")
				},
			),
		}

		err := app.RunContext(context.Background(), []string{"app"})

		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
	})
})

var _ = Describe("Prototype", func() {

	DescribeTable("examples", func(proto cli.Prototype, expected Fields) {
		app := &cli.App{
			Name: "any",
			Flags: []*cli.Flag{
				{
					Uses:    proto,
					Options: cli.Required,
					EnvVars: []string{"V"},
					Data:    map[string]interface{}{"A": 1},
				},
			},
			Args: []*cli.Arg{
				{
					Uses:    proto,
					Options: cli.Required,
					EnvVars: []string{"V"},
					Data:    map[string]interface{}{"A": 1},
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(app.Flags[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
		Expect(app.Args[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry("DefaultText", cli.Prototype{DefaultText: "e"}, Fields{"DefaultText": Equal("e")}),
		Entry("Description", cli.Prototype{Description: "d"}, Fields{"Description": Equal("d")}),
		Entry("FilePath", cli.Prototype{FilePath: "f"}, Fields{"FilePath": Equal("f")}),
		Entry("Category", cli.Prototype{Category: "f"}, Fields{"Category": Equal("f")}),
		Entry("HelpText", cli.Prototype{HelpText: "new help text"}, Fields{"HelpText": Equal("new help text")}),
		Entry("ManualText", cli.Prototype{ManualText: "explain"}, Fields{"ManualText": Equal("explain")}),
		Entry("Name", cli.Prototype{Name: "nom"}, Fields{"Name": Equal("nom")}),
		Entry("UsageText", cli.Prototype{UsageText: "nom"}, Fields{"UsageText": Equal("nom")}),
		Entry("Options", cli.Prototype{Options: cli.Hidden}, Fields{"Options": Equal(cli.Hidden | cli.Required)}),
		Entry("EnvVars", cli.Prototype{EnvVars: []string{"A"}}, Fields{"EnvVars": Equal([]string{"V", "A"})}),
		Entry("Data", cli.Prototype{Data: map[string]interface{}{"B": 3}}, Fields{"Data": Equal(map[string]interface{}{"A": 1, "B": 3})}),
		Entry("Value", cli.Prototype{Value: new(time.Duration)}, Fields{"Value": Equal(new(time.Duration))}),
		Entry("Completion", cli.Prototype{
			Completion: cli.CompletionFunc(func(*cli.CompletionContext) []cli.CompletionItem {
				return nil
			}),
		}, Fields{"Completion": Not(BeNil())}),
	)

	DescribeTable("Command examples", func(proto cli.Prototype, expected Fields) {
		app := &cli.App{
			Name:   "any",
			Stderr: io.Discard,
			Commands: []*cli.Command{
				{
					Uses:    proto,
					Data:    map[string]interface{}{"A": 1},
					Options: cli.RightToLeft,
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(app.Commands[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry("Description", cli.Prototype{Description: "d"}, Fields{"Description": Equal("d")}),
		Entry("Category", cli.Prototype{Category: "f"}, Fields{"Category": Equal("f")}),
		Entry("HelpText", cli.Prototype{HelpText: "new help text"}, Fields{"HelpText": Equal("new help text")}),
		Entry("ManualText", cli.Prototype{ManualText: "explain"}, Fields{"ManualText": Equal("explain")}),
		Entry("Name", cli.Prototype{Name: "nom"}, Fields{"Name": Equal("nom")}),
		Entry("UsageText", cli.Prototype{UsageText: "nom"}, Fields{"UsageText": Equal("nom")}),
		Entry("Options", cli.Prototype{Options: cli.Hidden}, Fields{"Options": Equal(cli.Hidden | cli.RightToLeft)}),
		Entry("Data", cli.Prototype{Data: map[string]interface{}{"B": 3}}, Fields{"Data": Equal(map[string]interface{}{"A": 1, "B": 3})}),
		Entry("Completion", cli.Prototype{
			Completion: cli.CompletionFunc(func(*cli.CompletionContext) []cli.CompletionItem {
				return nil
			}),
		}, Fields{"Completion": Not(BeNil())}),
	)

	DescribeTable("preserve existing values", func(proto cli.Prototype, expected Fields) {
		app := &cli.App{
			Name: "any",
			Flags: []*cli.Flag{
				{
					Uses:        proto,
					Value:       new(int),
					DefaultText: "existing DefaultText",
					HelpText:    "existing HelpText",
					ManualText:  "existing ManualText",
					UsageText:   "existing UsageText",
					Description: "existing Description",
					Category:    "existing Category",
				},
			},
			Args: []*cli.Arg{
				{
					Uses:        proto,
					Value:       new(int),
					DefaultText: "existing DefaultText",
					HelpText:    "existing HelpText",
					ManualText:  "existing ManualText",
					UsageText:   "existing UsageText",
					Description: "existing Description",
					Category:    "existing Category",
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(app.Flags[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
		Expect(app.Args[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry("DefaultText", cli.Prototype{DefaultText: "e"}, Fields{"DefaultText": Equal("existing DefaultText")}),
		Entry("Description", cli.Prototype{Description: "d"}, Fields{"Description": Equal("existing Description")}),
		Entry("Category", cli.Prototype{Category: "d"}, Fields{"Category": Equal("existing Category")}),
		Entry("HelpText", cli.Prototype{HelpText: "e"}, Fields{"HelpText": Equal("existing HelpText")}),
		Entry("ManualText", cli.Prototype{ManualText: "d"}, Fields{"ManualText": Equal("existing ManualText")}),
		Entry("UsageText", cli.Prototype{UsageText: "d"}, Fields{"UsageText": Equal("existing UsageText")}),
		Entry("FilePath", cli.Prototype{FilePath: "f"}, Fields{"FilePath": Equal("f")}),
		Entry("Name", cli.Prototype{Name: "nom"}, Fields{"Name": Equal("nom")}),
		Entry("Value", cli.Prototype{Value: new(time.Duration)}, Fields{"Value": Equal(new(int))}),
	)

	DescribeTable("flag-only examples", func(proto cli.Prototype, expected Fields) {
		app := &cli.App{
			Name: "any",
			Flags: []*cli.Flag{
				{
					Uses:    proto,
					Aliases: []string{"r"},
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(app.Flags[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry("Aliases", cli.Prototype{Aliases: []string{"age"}}, Fields{"Aliases": Equal([]string{"r", "age"})}),
	)

	DescribeTable("arg-only examples", func(proto cli.Prototype, expected Fields) {
		app := &cli.App{
			Name: "any",
			Args: []*cli.Arg{
				{
					Uses: proto,
				},
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(app.Args[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry("NArg", cli.Prototype{NArg: -2}, Fields{"NArg": Equal(-2)}),
	)

	DescribeTable("command-only examples", func(proto cli.Prototype, expected Fields) {
		app := &cli.App{
			Name: "any",
			Commands: []*cli.Command{
				{
					Uses:    proto,
					Aliases: []string{"r"},
				},
			},
			Stderr: io.Discard,
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(app.Commands[0]).To(PointTo(MatchFields(IgnoreExtras, expected)))
	},
		Entry("Aliases", cli.Prototype{Aliases: []string{"age"}}, Fields{"Aliases": Equal([]string{"r", "age"})}),
	)

	It("ensures value inside nested prototype (addresses bug)", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					// The inner prototype value should not apply because
					// the outer should have cleared the owner bit
					Name: "b",
					Uses: cli.Prototype{
						Value: new(bool),
						Setup: cli.Setup{
							Uses: cli.Prototype{
								Value: new(string),
							},
						},
					},
					Action: act,
				},
			},
		}
		args, _ := cli.Split("app -b")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		value := act.ExecuteArgsForCall(0).Value("")
		Expect(value).To(BeTrue())
	})

	Describe("Use", func() {
		It("appends to the pipeline", func() {
			step1 := new(joeclifakes.FakeAction)
			step2 := new(joeclifakes.FakeAction)
			pro := &cli.Prototype{
				Setup: cli.Setup{
					Optional: true,
					Uses:     step1,
				},
			}
			res := pro.Use(step2)
			Expect(res.Setup.Optional).To(BeTrue())
			Expect(res.Setup.Uses).To(Equal(cli.ActionPipeline([]cli.Action{
				step1,
				step2,
			})))
		})
	})
})

var _ = Describe("Setup", func() {
	var (
		setup cli.Setup
		err   error
	)

	JustBeforeEach(func() {
		app := &cli.App{
			Name:   "app",
			Action: setup,
		}

		err = app.RunContext(context.Background(), []string{"app"})
	})

	Context("when Optional is true", func() {
		BeforeEach(func() {
			setup = cli.Setup{
				Optional: true,
				Uses:     func() {},
			}
		})

		It("does not return timing mismatch error ", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when Optional is false", func() {
		BeforeEach(func() {
			setup = cli.Setup{
				Optional: false,
				Uses:     func() {},
			}
		})

		It("returns timing mismatch error ", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(cli.ErrTimingTooLate))
		})
	})

	Describe("Use", func() {
		It("appends to the pipeline", func() {
			before := new(joeclifakes.FakeAction)
			step1 := new(joeclifakes.FakeAction)
			step2 := new(joeclifakes.FakeAction)
			setup = cli.Setup{
				Optional: true,
				Uses:     step1,
				Before:   before,
			}
			res := setup.Use(step2)
			Expect(res.Optional).To(BeTrue())
			Expect(res.Uses).To(Equal(cli.ActionPipeline([]cli.Action{
				step1,
				step2,
			})))
			Expect(res.Before).To(Equal(before))
		})
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
			Stderr: io.Discard,
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
				Stderr: io.Discard,
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
				Stderr:  io.Discard,
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

var _ = Describe("HookBefore", func() {

	DescribeTable("errors",
		func(a *cli.App) {
			err := a.RunContext(context.Background(), []string{"app", "-f", "_"})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(cli.ErrTimingTooLate))
		},
		Entry(
			"HookBefore in action",
			&cli.App{
				Action: cli.HookBefore("*", nil),
				Flags: []*cli.Flag{
					{
						Name: "f",
					},
				},
			}),
		Entry(
			"HookBefore delegated to parent in action",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name:   "f",
						Action: cli.Implies("other", ""), // works by using a hook on parent command
					},
				},
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

		Entry("customizes a command",
			cli.Customize("sub", cli.Hidden),
			func(app *cli.App) {
				root, _ := app.Command("")
				Expect(root.VisibleSubcommands()).To(HaveLen(1 + 2)) // Includes "help" and "version"
			}),
	)

	It("flag can customize itself", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: cli.Customize("", cli.FlagSetup(func(f *cli.Flag) {
						f.SetData("ok", "2")
					})),
				},
			},
		}
		_, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Flags[0].Data).To(HaveKeyWithValue("ok", "2"))
	})

	It("inner customization win over outer", func() {
		innerCustomization := cli.Customize("--flag", cli.Data("scope", "inner"))
		outerCustomization := cli.Customize("--flag", cli.Data("scope", "outer"))

		app := &cli.App{
			Commands: []*cli.Command{
				{
					Name: "sub",
					Flags: []*cli.Flag{
						{
							Name: "flag",
						},
					},
					Uses: innerCustomization,
				},
			},
			Uses: outerCustomization,
		}
		app.Initialize(context.Background())
		Expect(app.Commands[0].Flags[0].Data).To(
			HaveKeyWithValue("scope", "inner"),
		)
	})

})

var _ = Describe("Accessory", func() {

	It("creates the flag", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:     "files",
					Value:    new(cli.FileSet),
					Category: "same category",
					Uses:     cli.Accessory("custom", (*cli.FileSet).RecursiveFlag),
				},
			},
			Action: act,
		}
		_ = app.RunContext(context.TODO(), []string{"app"})
		flags := act.ExecuteArgsForCall(0).Command().Flags
		flag := flags[len(flags)-1]

		Expect(flag.Name).To(Equal("custom"))
		Expect(flag.Value).To(Equal(new(bool)))
		Expect(flag.Category).To(Equal("same category"))
	})

	It("creates the flag with user specified name", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: new(cli.FileSet),
					Uses:  cli.Accessory("", (*cli.FileSet).RecursiveFlag),
				},
			},
			Action: act,
		}

		_ = app.RunContext(context.TODO(), []string{"app"})
		flags := act.ExecuteArgsForCall(0).Command().Flags
		flag := flags[len(flags)-1]
		Expect(flag.Name).To(Equal("recursive"))
	})

	It("creates the flag with implied name", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: new(cli.FileSet),
					Uses:  cli.Accessory("-", (*cli.FileSet).RecursiveFlag),
				},
			},
			Action: act,
		}

		_ = app.RunContext(context.TODO(), []string{"app"})
		flags := act.ExecuteArgsForCall(0).Command().Flags
		flag := flags[len(flags)-1]
		Expect(flag.Name).To(Equal("files-recursive"))
	})

	It("runs additional actions on the generated flag", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: new(cli.FileSet),
					Uses:  cli.Accessory("n", (*cli.FileSet).RecursiveFlag, cli.Description("my custom description")),
				},
			},
			Action: act,
		}

		_ = app.RunContext(context.TODO(), []string{"app"})
		flags := act.ExecuteArgsForCall(0).Command().Flags
		flag := flags[len(flags)-1]
		Expect(flag.Description).To(Equal("my custom description"))
	})
})

var _ = Describe("Bind", func() {

	It("invokes bind func with value from flag", func() {
		var value uint64
		binder := func(r uint64) error {
			value = r
			return nil
		}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:   "memory",
					Value:  new(uint64),
					Action: cli.Bind(binder),
				},
			},
		}
		args, _ := cli.Split("app --memory 33")
		_ = app.RunContext(context.TODO(), args)
		Expect(value).To(Equal(uint64(33)))
	})

	It("invokes bind func with static value", func() {
		var value uint64
		binder := func(r uint64) error {
			value = r
			return nil
		}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "max-memory",
					Uses: cli.Bind(binder, 1024),
				},
			},
		}
		args, _ := cli.Split("app --max-memory")
		_ = app.RunContext(context.TODO(), args)
		Expect(value).To(Equal(uint64(1024)))
		Expect(app.Flags[0].Value).To(PointTo(BeTrue()))
	})

	DescribeTable("generics",
		func(uses cli.Action, expected interface{}) {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "f",
						Uses: uses,
					},
				},
				Action: act,
			}
			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(act.ExecuteArgsForCall(0).Value("f")).To(BeAssignableToTypeOf(expected))
		},

		Entry("bool",
			cli.Bind(func(_ bool) error { return nil }),
			false),
		Entry("File",
			cli.Bind(func(_ *cli.File) error { return nil }),
			new(cli.File)),
		Entry("FileSet",
			cli.Bind(func(_ *cli.FileSet) error { return nil }),
			new(cli.FileSet)),
		Entry("Regexp",
			cli.Bind(func(_ *regexp.Regexp) error { return nil }),
			new(regexp.Regexp)),
		Entry("NameValue",
			cli.Bind(func(_ *cli.NameValue) error { return nil }),
			new(cli.NameValue)),
		Entry("List",
			cli.Bind(func(_ []string) error { return nil }),
			[]string{}),
		Entry("Map",
			cli.Bind(func(_ map[string]string) error { return nil }),
			map[string]string{}),
	)
})

var _ = Describe("BindIndirect", func() {

	It("copies the implied value of the function", func() {
		fs := &cli.FileSet{Recursive: true}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "no-recursive",
					Uses: cli.BindIndirect("files", (*cli.FileSet).SetRecursive, false),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		app.Initialize(context.Background())
		Expect(app.Flags[0].Value).To(Equal(new(bool)))
	})

	It("invokes bind func with static value", func() {
		fs := &cli.FileSet{Recursive: true}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "no-recursive",
					Value: new(bool),
					Uses:  cli.BindIndirect("files", (*cli.FileSet).SetRecursive, false),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app --no-recursive .")
		_ = app.RunContext(context.TODO(), args)
		Expect(fs.Recursive).To(BeFalse())
	})

	It("invokes bind func with corresponding value", func() {
		fs := new(cli.FileSet)
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:   "recursive",
					Value:  new(bool),
					Action: act,
					Uses:   cli.BindIndirect("files", (*cli.FileSet).SetRecursive),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app --recursive .")
		_ = app.RunContext(context.TODO(), args)
		Expect(act.ExecuteCallCount()).To(Equal(1), "action should still be called")
		Expect(fs.Recursive).To(BeTrue())
	})
})

var _ = Describe("EachOccurrence", func() {

	It("provides access to Raw and RawOccurrence", func() {
		act := new(joeclifakes.FakeAction)
		raw := [][]string{}
		rawOccurrences := [][]string{}
		act.ExecuteCalls(func(c *cli.Context) error {
			raw = append(raw, c.Raw(""))
			rawOccurrences = append(rawOccurrences, c.RawOccurrences(""))
			return nil
		})

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:    "f",
					Value:   new(string),
					Action:  act,
					Options: cli.EachOccurrence,
				},
			},
		}
		args, _ := cli.Split("app -f h -f i")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(rawOccurrences).To(Equal([][]string{{"h"}, {"i"}}))
		Expect(raw).To(Equal([][]string{{"-f", "h"}, {"-f", "i"}}))
	})

	It("works with Bind in Uses pipeline", func() {
		// Tests for a bug:  pipeline additions provided by Bind
		// where not wrapped in EachOccurrence
		fs := new(cli.FileSet)
		var values []uint64
		binder := func(r uint64) error {
			values = append(values, r)
			return nil
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:    "f",
					Options: cli.EachOccurrence,
					Uses:    cli.Bind(binder),
				},
				{
					Name:  "vars",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app -f 1019 -f 1044")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(values).To(Equal([]uint64{1019, 1044}))
	})

	It("stops after first error", func() {
		var values []uint64
		binder := func(r uint64) error {
			values = append(values, r)
			if r == 1 {
				return fmt.Errorf("error: 1")
			}
			return nil
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:    "f",
					Options: cli.EachOccurrence,
					Uses:    cli.Bind(binder),
				},
			},
		}
		args, _ := cli.Split("app -f 0 -f 1 -f 1044")
		err := app.RunContext(context.TODO(), args)
		Expect(err).To(MatchError("error: 1"))
		Expect(values).To(Equal([]uint64{0, 1}))
	})

	DescribeTable("examples", func(flag *cli.Flag, arguments string, expected []interface{}) {
		act := new(joeclifakes.FakeAction)
		var callIndex int // keep track of which index is called
		act.ExecuteCalls(func(c *cli.Context) error {
			actual := c.Value("")
			Expect(actual).To(Equal(expected[callIndex]))
			callIndex++
			return nil
		})
		flag.Action = act
		flag.Options |= cli.EachOccurrence

		app := &cli.App{
			Flags: []*cli.Flag{
				flag,
			},
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(callIndex).To(Equal(len(expected)))
	},
		Entry("string",
			&cli.Flag{
				Name:  "f",
				Value: new(string),
			},
			"app -f h -f u -f g",
			[]interface{}{"h", "u", "g"},
		),
		Entry("string merged",
			&cli.Flag{
				Name:    "f",
				Value:   new(string),
				Options: cli.Merge,
			},
			"app -f h -f u -f g",
			[]interface{}{"h", "h u", "h u g"},
		),
		Entry("string initial value",
			&cli.Flag{
				Name: "f",
				Value: func() *string {
					s := "hello"
					return &s
				}(),
			},
			"app -f world -f earth",
			[]interface{}{"world", "earth"},
		),
		Entry("int",
			&cli.Flag{
				Name:  "f",
				Value: new(int),
			},
			"app -f 1 -f 2",
			[]interface{}{1, 2},
		),
		Entry("bool",
			&cli.Flag{
				Name:  "f",
				Value: new(bool),
			},
			"app -f -f -f",
			[]interface{}{true, true, true},
		),
		Entry("NameValue",
			&cli.Flag{
				Name:  "f",
				Value: new(cli.NameValue),
			},
			"app -f a=b -f d=e -f j=k",
			[]interface{}{&cli.NameValue{Name: "a", Value: "b"}, &cli.NameValue{Name: "d", Value: "e"}, &cli.NameValue{Name: "j", Value: "k"}},
		),
	)
})

var _ = Describe("Implies", func() {

	DescribeTable("examples", func(arguments string, expected map[string]string) {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "encryption-key",
					Uses: cli.Implies("mode", "encrypt"),
				},
				{
					Name: "mode",
				},
			},
			Action: act,
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		c := act.ExecuteArgsForCall(0)
		actual := map[string]string{
			"encryption-key": c.String("encryption-key"),
			"mode":           c.String("mode"),
		}

		Expect(actual).To(Equal(expected))
	},
		Entry("implicit value", "app --encryption-key=AAA", map[string]string{
			"mode":           "encrypt",
			"encryption-key": "AAA",
		}),
		Entry("explicit value wins no matter order", "app --mode=decrypt --encryption-key=AAA", map[string]string{
			"mode":           "decrypt",
			"encryption-key": "AAA",
		}),
		Entry("don't invoke when not set", "app", map[string]string{
			"mode":           "",
			"encryption-key": "",
		}),
	)

	It("invokes action when ImpliedAction is set", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "encryption-key",
					Uses: cli.Implies("mode", "encrypt"),
				},
				{
					Name:    "mode",
					Options: cli.ImpliedAction,
					Action:  act,
				},
			},
		}
		args, _ := cli.Split("app --encryption-key=AAA")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))
	})
})

var _ = Describe("Enum", func() {

	It("set expected values in prototype", func() {
		flag := &cli.Flag{
			Name:    "test",
			Aliases: []string{"t"},
			Uses:    cli.Enum("case", "suite"),
		}
		_ = cli.InitializeFlag(flag)

		Expect(flag.UsageText).To(Equal("(case|suite)"))
	})

	DescribeTable("validation", func(arguments string, uses cli.Action, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Name: "long", Uses: uses},
			},
			Action: func() {},
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)
		Expect(err).To(HaveOccurred())
		Expect(err).To(expected)
	},
		Entry("two options",
			"app --long yes",
			cli.Enum("ok", "no"),
			MatchError("unrecognized value \"yes\" for --long, expected `ok' or `no'"),
		),
		Entry("three options",
			"app --long oui",
			cli.Enum("ok", "no", "yes"),
			MatchError("unrecognized value \"oui\" for --long, expected `ok', `no', or `yes'"),
		),
	)

	DescribeTable("completion", func(arguments string, incomplete string, uses cli.Action, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Name: "long", Aliases: []string{"s"}, Uses: uses},
			},
			Action: func() {},
		}
		args, _ := cli.Split(arguments)
		ctx, err := app.Initialize(context.TODO())
		Expect(err).NotTo(HaveOccurred())
		Expect(ctx.Complete(args, incomplete)).To(expected)
	},
		Entry("long enum", "app --long=", "--long=", cli.Enum("ok", "no"), ConsistOf([]cli.CompletionItem{
			{Value: "--long=ok"},
			{Value: "--long=no"},
		})),
		Entry("short enum", "app -s", "-s", cli.Enum("ok", "no"), ConsistOf([]cli.CompletionItem{
			{Value: "-sok"},
			{Value: "-sno"},
		})),
	)

	DescribeTable("synopsis", func(uses cli.Action, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{Name: "s", Uses: uses},
			},
			Action: func() {},
		}
		_, err := app.Initialize(context.TODO())
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Flags[0].Synopsis()).To(expected)
	},
		Entry("2 items", cli.Enum("ok", "no"), Equal("-s (ok|no)")),
		Entry("3 items", cli.Enum("ok", "maybe", "no"), Equal("-s (ok|maybe|no)")),
		Entry("overflow", cli.Enum("ok", "maybe", "no", "duh"), Equal("-s (ok|maybe|no|...)")),
	)
})

var _ = Describe("Mutex", func() {
	DescribeTable("examples", func(arguments string, expected types.GomegaMatcher) {
		app := cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "a",
					Uses:  cli.Mutex("b", "c", "d"),
					Value: cli.Bool(),
				},
				{Name: "b", Value: cli.Bool()},
				{Name: "c", Value: cli.Bool()},
				{Name: "d", Value: cli.Bool()},
			},
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.Background(), args)
		Expect(err).To(expected)
	},
		Entry("one other", "app -ab", MatchError("either -a or -b can be used, but not both")),
		Entry("two others", "app -abc", MatchError("can't use -a together with -b or -c")),
		Entry("three others", "app -abcd", MatchError("can't use -a together with -b, -c, or -d")),
	)

})

var _ = Describe("ValueTransform", func() {

	var testFileSystem = func() fs.FS {
		appFS := afero.NewMemMapFs()

		afero.WriteFile(appFS, "world", []byte("earth"), 0644)
		afero.WriteFile(appFS, "plan", []byte("b"), 0644)
		return afero.NewIOFS(appFS)
	}()

	DescribeTable("examples", func(arguments string, value interface{}, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:  "long",
					Uses:  cli.ValueTransform(cli.TransformFileReference(testFileSystem, false)),
					Value: value,
				},
			},
			Action: func() {},
		}
		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(expected)
	},
		Entry("NameValues",
			"app --long file=world --long file2=plan",
			cli.NameValues(),
			Equal(&[]*cli.NameValue{
				{
					Name:  "file",
					Value: "earth",
				},
				{
					Name:  "file2",
					Value: "b",
				},
			}),
		),
		Entry("Map",
			"app --long exec=plan",
			cli.Map(),
			Equal(&map[string]string{
				"exec": "b",
			}),
		),
	)
})

var _ = Describe("FromEnv", Ordered, func() {
	UseEnvVars(map[string]string{
		"NOMINAL":           "nominal_value",
		"FLAG_NAME":         "flag_name",
		"APP_FLAG_NAME":     "app_flag_name",
		"APP__FLAG_NAME":    "app_dbl_flag_name",
		"FLAG_NAME__APP":    "flag_name_dbl_app",
		"APP_FLAG_NAME_APP": "app_flag_name_app",
		"EMPTY_VAR":         "",
	})

	DescribeTable("examples", func(pattern string, expected string) {
		var value string
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:    "f",
					Uses:    cli.FromEnv(pattern),
					Value:   &value,
					Aliases: []string{"flag-name"},
				},
			},
			Action: func() {},
		}
		args, _ := cli.Split("app")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(expected))
	},
		Entry("nominal", "NOMINAL", "nominal_value"),
		Entry("flag name template", "{}", "flag_name"),
		Entry("flag name template end", "APP{}", "app_flag_name"),
		Entry("add underscore", "APP_{}", "app_dbl_flag_name"),
		Entry("flag name template start", "{}_APP", "flag_name_dbl_app"),
		Entry("flag name template middle", "APP{}APP", "app_flag_name_app"),
	)

	It("empty env var is not treated as true (addresses a bug)", func() {
		var value bool
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:  "f",
					Uses:  cli.FromEnv("EMPTY_VAR"),
					Value: &value,
				},
			},
			Action: func() {},
		}
		args, _ := cli.Split("app")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(BeFalse())
	})
})

var _ = Describe("FromFilePath", func() {

	It("sets up value from option", func() {
		act := new(joeclifakes.FakeAction)
		var testFileSystem = func() fs.FS {
			appFS := afero.NewMemMapFs()

			appFS.MkdirAll("src/a", 0755)
			afero.WriteFile(appFS, "src/a/b.txt", []byte("b contents"), 0644)
			return afero.NewIOFS(appFS)
		}()
		var actual string

		app := &cli.App{
			FS: testFileSystem,
			Args: []*cli.Arg{
				{
					Name:  "f",
					Uses:  cli.FromFilePath(nil, "src/a/b.txt"),
					Value: &actual,
				},
			},
			Action: act,
		}

		args, _ := cli.Split("app")
		app.RunContext(context.TODO(), args)

		Expect(actual).To(Equal("b contents"))
		Expect(act.ExecuteCallCount()).To(Equal(1))
	})
})

func SkipOnWindows() {
	if runtime.GOOS == "windows" {
		Skip("not tested on Windows")
	}
}

func UseEnvVars(env map[string]string) {
	BeforeAll(func() {
		for k, v := range env {
			os.Setenv(k, v)
		}
	})
	AfterAll(func() {
		for k := range env {
			os.Unsetenv(k)
		}
	})
}
