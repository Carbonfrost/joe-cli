package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {

	Describe("actions", func() {

		var (
			act       *joeclifakes.FakeActionHandler
			beforeAct *joeclifakes.FakeActionHandler

			app *cli.App
		)

		BeforeEach(func() {
			act = new(joeclifakes.FakeActionHandler)
			beforeAct = new(joeclifakes.FakeActionHandler)

			app = &cli.App{
				Commands: []*cli.Command{
					{
						Name:   "c",
						Action: act,
						Before: beforeAct,
					},
				},
			}

			args, _ := cli.Split("app c")
			app.RunContext(nil, args)
		})

		It("executes action on executing sub-command", func() {
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("executes before action on executing sub-command", func() {
			Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
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
