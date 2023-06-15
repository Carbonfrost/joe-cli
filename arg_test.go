package cli_test

import (
	"context"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/spf13/afero"
)

var _ = Describe("Arg", func() {

	It("sets default name by index", func() {
		var (
			called         bool
			nameInPipeline string
		)
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					NArg: 1,
					Uses: func(c *cli.Context) {
						called = true
						nameInPipeline = c.Arg().Name
					},
				},
			},
		}
		app.RunContext(context.TODO(), []string{"app", "a"})

		Expect(app.Args[0].Name).To(Equal("_1"))
		Expect(called).To(BeTrue())
		Expect(nameInPipeline).To(
			Equal(""), "the name should be set to a generated name after all initializers have run",
		)
	})

	Describe("Action", func() {
		var (
			act       *joeclifakes.FakeAction
			app       *cli.App
			arguments = "app f"
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			app = &cli.App{
				Name: "app",
				Args: []*cli.Arg{
					{
						Name:   "f",
						Action: act,
					},
				},
			}
		})

		JustBeforeEach(func() {
			args, _ := cli.Split(arguments)
			app.RunContext(context.TODO(), args)
		})

		It("executes action on setting Arg", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("contains args in captured context", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Args()).To(Equal([]string{"f"}))
		})

		It("provides properly initialized context", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Name()).To(Equal("<f>"))
			Expect(captured.Path().String()).To(Equal("app <f>"))
		})

		It("contains the value in the context", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Value("")).To(Equal("f"))
		})

		It("contains the correct Occurrences count", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Occurrences("")).To(Equal(1))
		})

		It("obtains context path", func() {
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
			Expect(captured.Path().IsArg()).To(BeTrue())
			Expect(captured.Path().Last()).To(Equal("<f>"))
			Expect(captured.Path().String()).To(Equal("app <f>"))
		})
	})

	Describe("Value", func() {

		DescribeTable(
			"inferred from NArg",
			func(count int, expected interface{}) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: count,
						},
					},
					Action: act,
				}
				arguments := "f"
				if count > 0 {
					arguments = strings.Repeat(" g", count)
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(context.TODO(), args)
				Expect(err).NotTo(HaveOccurred())

				captured := cli.FromContext(cli.FromContext(act.ExecuteArgsForCall(0)))
				arg, _ := captured.LookupArg("f")
				Expect(arg.Value).To(BeAssignableToTypeOf(expected))
			},
			Entry("list when 0", 0, cli.String()),
			Entry("string when 1", 1, cli.String()),
			Entry("list when 2", 2, cli.List()),
			Entry("list when -2", -2, cli.List()),
		)

	})

	Describe("NArg", func() {

		DescribeTable(
			"inferred from Value",
			func(value interface{}, validArgs string, expected types.GomegaMatcher) {
				act := new(joeclifakes.FakeAction)
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name:  "f",
							Value: value,
						},
					},
					Action: act,
				}
				args, _ := cli.Split("app " + validArgs)
				err := app.RunContext(context.TODO(), args)
				Expect(err).NotTo(HaveOccurred())

				captured := cli.FromContext(act.ExecuteArgsForCall(0))
				arg, _ := captured.LookupArg("f")
				Expect(arg.ActualArgCounter()).To(expected)
			},
			Entry("string", new(string), "_", Equal(cli.ArgCount(0))),
			Entry("[]string", new([]string), "_", Equal(cli.ArgCount(-2))),
			Entry("*File", new(cli.File), "_", Equal(cli.ArgCount(0))),
			Entry("*FileSet", new(cli.FileSet), "_", Equal(cli.ArgCount(-2))),
			Entry("value counter convention", &valueHasCounter{}, "_", BeAssignableToTypeOf(argCounterImpl{})),
		)

		DescribeTable(
			"arg parsing",
			func(count int, arguments string, match types.GomegaMatcher) {
				items := []string{}
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name:  "a",
							NArg:  count,
							Value: &items,
						},
					},
					Flags: []*cli.Flag{
						{
							Name:  "f",
							Value: cli.Bool(),
						},
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(context.TODO(), args)
				Expect(err).NotTo(HaveOccurred())
				Expect(items).To(match)
			},
			Entry("exactly 1", 1, "one", Equal([]string{"one"})),
			Entry("exactly 3", 3, "one two three", Equal([]string{"one", "two", "three"})),
			Entry("all values even flags", -1, "one -f two -f", Equal([]string{"one", "-f", "two", "-f"})),
			Entry("values stop on flags", -2, "one two -f", Equal([]string{"one", "two"})),
			Entry("optional empty", 0, "", Equal([]string{})),
		)

		DescribeTable(
			"errors",
			func(count int, arguments string, match types.GomegaMatcher) {
				app := &cli.App{
					Name: "app",
					Args: []*cli.Arg{
						{
							Name: "f",
							NArg: count,
						},
					},
				}
				args, _ := cli.Split("app " + arguments)
				err := app.RunContext(context.TODO(), args)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(match))
			},
			Entry("missing when 1", 1, "", Equal("expected argument")),
			Entry("too few by 1", 2, "a", Equal("expected 2 arguments for <f>")),
			Entry("too many by 1", 1, "a b", Equal(`unexpected argument "b"`)),
		)
	})

	Describe("Synopsis", func() {

		DescribeTable("examples",
			func(f *cli.Arg, expected string) {
				Expect(f.Synopsis()).To(Equal(expected))
			},
			Entry(
				"bool arg",
				&cli.Arg{
					Name:  "arg",
					Value: cli.Bool(),
				},
				"<arg>",
			),
			Entry(
				"repeat arg",
				&cli.Arg{
					Name:  "arg",
					NArg:  -1,
					Value: cli.Bool(),
				},
				"<arg>...",
			),
		)

	})

	Describe("EnvVars", Ordered, func() {
		UseEnvVars(map[string]string{
			"ANOTHER_ONE": "another_one",
		})

		It("uses env vars mutated by initializer", func() {
			// EnvVars could be mutated by the time the implicit value is actually
			// generated (this is defined behavior)
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name: "f",
						Uses: func(c *cli.Context) {
							c.Arg().EnvVars = append(c.Arg().EnvVars, "ANOTHER_ONE")
						},
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app")
			app.RunContext(context.TODO(), args)
			Expect(cli.FromContext(act.ExecuteArgsForCall(0)).String("f")).To(Equal("another_one"))
		})
	})

	Describe("FilePath", func() {

		It("sets up value from option", func() {
			impliedAct := new(joeclifakes.FakeAction)
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
						Name:     "f",
						FilePath: "src/a/b.txt",
						Value:    &actual,
						Options:  cli.ImpliedAction,
						Action:   impliedAct,
					},
				},
			}

			args, _ := cli.Split("app")
			app.RunContext(context.TODO(), args)

			Expect(actual).To(Equal("b contents"))
			Expect(impliedAct.ExecuteCallCount()).To(Equal(1))
		})
	})

	Context("when environment variables are set", func() {
		var (
			actual    string
			arguments string
			occurs    int
			options   cli.Option
			executed  bool
		)

		BeforeEach(func() {
			arguments = "app "
			actual = ""
			options = 0
		})

		JustBeforeEach(func() {
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name:    "f",
						EnvVars: []string{"_GOCLI_F"},
						Value:   &actual,
						Options: options,
						Action: func(c *cli.Context) {
							occurs = c.Occurrences("")
							executed = true
						},
					},
				},
			}

			os.Setenv("_GOCLI_F", "environment value")
			args, _ := cli.Split(arguments)
			app.RunContext(context.TODO(), args)
		})

		Context("when ImpliedAction is set", func() {
			BeforeEach(func() {
				options = cli.ImpliedAction
			})

			It("executes implied action", func() {
				Expect(executed).To(BeTrue())
			})
		})

		It("sets up value from environment", func() {
			Expect(actual).To(Equal("environment value"))
		})

		Context("when value also set", func() {
			BeforeEach(func() {
				arguments = "app 'option text'"
			})

			It("sets up value from option", func() {
				Expect(actual).To(Equal("option text"))
			})

			It("has 1 occurrence", func() {
				Expect(occurs).To(Equal(1))
			})
		})
	})

	It("supports sequence -- to next arg", func() {
		arg1 := cli.List()
		arg2 := cli.List()
		arg3 := cli.List()
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:  "arg1",
					Value: arg1,
				},
				{
					Name:  "arg2",
					Value: arg2,
				},
				{
					Name:  "arg3",
					Value: arg3,
				},
			},
		}
		args, _ := cli.Split("app -- arg1 -- arg2 -- arg3")
		_ = app.RunContext(context.TODO(), args)

		// These should accumulate single values rather than lists
		Expect(*arg1).To(Equal([]string{"arg1"}))
		Expect(*arg2).To(Equal([]string{"arg2"}))
		Expect(*arg3).To(Equal([]string{"arg3"}))
	})

	It("can set and define name and value by initializer", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Uses: func(c *cli.Context) {
						f := c.Arg()
						f.Name = "uses"
						f.Value = new(bool)
						f.Action = act
					},
				},
			},
		}

		err := app.RunContext(context.TODO(), []string{"app", "true"})
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))

		Expect(app.Args[0].Name).To(Equal("uses"))
		Expect(app.Args[0].Value).To(PointTo(BeTrue()))
	})

	It("can set and define name by initializer", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Value: &customValue{
						init: cli.ActionOf(func(c *cli.Context) {
							f := c.Arg()
							f.Name = "uses"
							f.Value = new(bool)
							f.Action = act
						}),
					},
				},
			},
		}

		err := app.RunContext(context.TODO(), []string{"app", "true"})
		Expect(err).NotTo(HaveOccurred())
		Expect(act.ExecuteCallCount()).To(Equal(1))

		Expect(app.Args[0].Name).To(Equal("uses"))
		Expect(app.Args[0].Value).To(PointTo(BeTrue()))
	})

	It("can set and define NArg by initializer", func() {
		app := &cli.App{
			Name: "app",
			Args: []*cli.Arg{
				{
					Name: "u", // Allocate a list because NArg is set
					Uses: cli.ArgSetup(func(a *cli.Arg) {
						a.NArg = 5
					}),
				},
				{
					Name:  "v",
					Value: new(string), // Keep this value even though NArg changes
					Uses: cli.ArgSetup(func(a *cli.Arg) {
						a.NArg = 5
					}),
				},
			},
		}

		err := app.RunContext(context.TODO(), []string{"app", "1", "2", "3", "4", "5", "1", "2", "3", "4", "5"})
		Expect(err).NotTo(HaveOccurred())

		Expect(app.Args[0].Value).To(PointTo(Equal([]string{"1", "2", "3", "4", "5"})))
		Expect(app.Args[1].Value).To(PointTo(Equal("1 2 3 4 5")))
	})

	DescribeTable("initializers", func(act cli.Action, expected types.GomegaMatcher) {
		app := &cli.App{
			Name: "a",
			Args: []*cli.Arg{
				{
					Uses: act,
					Name: "a",
				},
			},
		}

		args, _ := cli.Split("app s")
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		cmd, _ := app.Command("")
		Expect(cmd.Args[0]).To(PointTo(expected))
	},
		Entry(
			"Category",
			cli.Category("abc"),
			MatchFields(IgnoreExtras, Fields{"Category": Equal("abc")}),
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
		Entry(
			"SetCompletion",
			cli.SetCompletion(cli.CompletionValues("ok")),
			MatchFields(IgnoreExtras, Fields{"Completion": Not(BeNil())}),
		),
	)
})

var _ = Describe("Args", func() {

	It("sets default name by index", func() {
		//lint:ignore SA5012 namevalue args are intentionally used incorrectly to test for the panic
		Expect(func() { cli.Args("unevent") }).To(PanicWith(Equal("unexpected number of arguments")))
	})

})

var _ = Describe("ArgCount", func() {
	DescribeTable("examples", func(value interface{}, expected types.GomegaMatcher) {
		Expect(func() {
			cli.ArgCount(value)
		}).To(expected)
	},
		Entry("ArgCounter", new(joeclifakes.FakeArgCounter), Not(Panic())),
		Entry("1", 1, Not(Panic())),
		Entry("nil", nil, Not(Panic())),
		Entry("bad", "", Panic()),
		Entry("uninitialized arg", &cli.Arg{}, Panic()),
		Entry("uninitialized flag", &cli.Flag{}, Panic()),
	)

	DescribeTable("examples", func(value interface{}, expected types.GomegaMatcher) {
		Expect(cli.ArgCount(value)).To(expected)
	},
		Entry("arg", cli.Initialized(&cli.Arg{NArg: 1}).Arg(), Equal(cli.ArgCount(1))),
		Entry("flag", cli.Initialized(&cli.Flag{}).Flag(), Equal(cli.DefaultFlagCounter())),
	)
})

var _ = Describe("OptionalArg", func() {
	DescribeTable("examples", func(arguments string, expectedA types.GomegaMatcher, expectedB types.GomegaMatcher) {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name: "a",
					NArg: cli.OptionalArg(regexp.MustCompile("hel?").MatchString),
				},
				{
					Name:  "b",
					Value: cli.List(),
				},
			},
			Action: act,
		}

		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.String("a")).To(expectedA)
		Expect(context.List("b")).To(expectedB)
	},
		Entry("no argument", "app", Equal(""), BeNil()),
		Entry("match a", "app help", Equal("help"), BeNil()),
		Entry("no match a", "app something else", Equal(""), Equal([]string{"something", "else"})),
		Entry("match a twice", "app help help else", Equal("help"), Equal([]string{"help", "else"})),
	)
})

type valueHasCounter struct{}

func (*valueHasCounter) NewCounter() cli.ArgCounter { return argCounterImpl{} }
func (*valueHasCounter) Set(arg string) error       { return nil }
func (*valueHasCounter) String() string             { return "" }

type argCounterImpl struct{}

func (argCounterImpl) Take(arg string, possibleFlag bool) error { return nil }
func (argCounterImpl) Done() error                              { return nil }
