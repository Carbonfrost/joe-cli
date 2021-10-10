package cli_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {

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

	Describe("Synopsis", func() {
		DescribeTable("examples",
			func(cmd *cli.Command, expected string) {
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
						{Name: "arg"},
					},
					Name: "cmd",
				},
				"cmd [-t] <arg>",
			))

	})
})
