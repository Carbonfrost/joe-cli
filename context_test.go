package cli_test

import (
	"context"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context", func() {

	Describe("Value", func() {
		It("contains flag value at the app level", func() {
			act := new(joeclifakes.FakeActionHandler)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.Bool(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app -f")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal(true))
		})

		It("contains flag value from inherited context", func() {
			act := new(joeclifakes.FakeActionHandler)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: cli.String(),
					},
				},
				Commands: []*cli.Command{
					{
						Name:   "sub",
						Action: act,
					},
				},
			}

			args, _ := cli.Split("app -f dom sub")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal("dom"))
		})

		It("contains flag value set using one of its aliases", func() {
			act := new(joeclifakes.FakeActionHandler)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "f",
						Aliases: []string{"alias"},
						Value:   cli.Bool(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app --alias")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal(true))
		})

		It("contains arg value", func() {
			act := new(joeclifakes.FakeActionHandler)
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name:  "f",
						Value: cli.List(),
						NArg:  -1,
					},
				},
				Action: act,
			}

			args, _ := cli.Split("app s r o")
			_ = app.RunContext(context.TODO(), args)

			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.Value("f")).To(Equal([]string{"s", "r", "o"}))
		})
	})

	Describe("Before", func() {

		It("defers when set from initializer", func() {
			act := new(joeclifakes.FakeActionHandler)
			act.ExecuteCalls(func(c *cli.Context) error {
				Expect(c.IsBefore()).To(BeTrue())
				return nil
			})
			app := &cli.App{
				Uses: func(c *cli.Context) {
					c.Before(act)

					Expect(act.ExecuteCallCount()).To(Equal(0))
				},
			}

			_ = app.RunContext(context.TODO(), []string{"app"})
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("invokes immediately in the before context", func() {
			ctx := &cli.Context{}
			act := new(joeclifakes.FakeActionHandler)
			cli.SetBeforeTiming(ctx)

			_ = ctx.Before(act)
			Expect(act.ExecuteCallCount()).To(Equal(1))
			capturedContext := act.ExecuteArgsForCall(0)
			Expect(capturedContext.IsBefore()).To(BeTrue())
		})

		DescribeTable("error when timing after",
			func(timing func(*cli.Context)) {
				act := new(joeclifakes.FakeActionHandler)
				ctx := &cli.Context{}
				timing(ctx)

				err := ctx.Before(act)
				Expect(err).To(HaveOccurred())
			},
			Entry("action timing", cli.SetActionTiming),
			Entry("after timing", cli.SetAfterTiming),
		)
	})

})

var _ = Describe("ContextPath", func() {

	DescribeTable("Match",
		func(pattern string, path string) {
			p := cli.ContextPath(strings.Fields(path))
			Expect(p.Match(pattern)).To(BeTrue())
		},
		Entry("simple", "app", "app"),
		Entry("simple command", "sub", "app sub"),
		Entry("nested command", "sub", "app app sub"),
		Entry("simple flag", "--flag", "app --flag"),
		Entry("nested flag", "--flag", "app app sub --flag"),
		Entry("anything", "*", "app --flag"),
		Entry("any flag", "-", "app --flag"),
		Entry("any arg", "<>", "app <arg>"),
		Entry("sub path", "sub cmd", "app sub cmd"),
	)
})
