package cli_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

type mappedValues struct {
	Flag1 bool
	Flag2 bool
	Flag3 string
	Arg   string
}

type commanderValues struct {
	Global  bool
	Command string
}

var _ = Describe("RunContext", func() {
	DescribeTable("bind subcommand",
		func(arguments string, expectedGlobal types.GomegaMatcher, expectedSub types.GomegaMatcher) {
			var (
				global commanderValues
				sub    mappedValues
			)
			var app = &cli.App{
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "global",
						Destination: &global.Global,
					},
				},
				Commands: []*cli.Command{
					&cli.Command{
						Name: "sub",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:        "flag1",
								Destination: &sub.Flag1,
							},
						},
						Action: cli.ActionFunc(func(*cli.Context) error {
							global.Command = "sub"
							return nil
						}),
					},
				},
			}
			args, _ := cli.Split("app " + arguments)
			err := app.RunContext(context.TODO(), args)

			Expect(err).NotTo(HaveOccurred())
			Expect(global).To(expectedGlobal)
			Expect(sub).To(expectedSub)
		},
		Entry(
			"global flag only",
			"--global",
			Equal(commanderValues{
				Command: "",
				Global:  true,
			}),
			Equal(mappedValues{}),
		),
		Entry(
			"name sub-command",
			"sub",
			Equal(commanderValues{
				Command: "sub",
			}),
			Equal(mappedValues{}),
		),
		Entry(
			"simple sub-command flag use",
			"sub --flag1",
			Equal(commanderValues{
				Command: "sub",
			}),
			Equal(mappedValues{
				Flag1: true,
			}),
		),
		XEntry( // TODO interspersed global flags
			"intersperse global flags",
			"sub --flag1 --global",
			Equal(commanderValues{
				Command: "sub",
				Global:  true,
			}),
			Equal(mappedValues{
				Flag1: true,
			}),
		),
	)

	DescribeTable("bind args and flags",
		func(arguments string, expected types.GomegaMatcher) {
			var result mappedValues
			var app = &cli.App{
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "flag1",
						Destination: &result.Flag1,
					},
					&cli.BoolFlag{
						Name:        "flag2",
						Destination: &result.Flag2,
					},
					&cli.StringFlag{
						Name:        "flag3",
						Destination: &result.Flag3,
					},
				},
				Args: []cli.Arg{
					&cli.StringArg{
						Name:        "arg",
						Destination: &result.Arg,
					},
				},
			}
			args, _ := cli.Split("app " + arguments)
			err := app.RunContext(context.TODO(), args)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(expected)
		},
		Entry(
			"simple flag use",
			"--flag1",
			Equal(mappedValues{
				Flag1: true,
			}),
		),
		Entry(
			"two flags",
			"--flag1 --flag2",
			Equal(mappedValues{
				Flag1: true,
				Flag2: true,
			}),
		),
		Entry(
			"string flag",
			"--flag3=inline",
			Equal(mappedValues{
				Flag3: "inline",
			}),
		),
		Entry(
			"string flag separated by space",
			"--flag3 space",
			Equal(mappedValues{
				Flag3: "space",
			}),
		),
		Entry(
			"simple positional argument",
			"argument",
			Equal(mappedValues{
				Arg: "argument",
			}),
		),
		Entry(
			"allow options after arguments",
			"--flag1 argument --flag2",
			Equal(mappedValues{
				Flag1: true,
				Flag2: true,
				Arg:   "argument",
			}),
		),
	)

	DescribeTable("bind args and flags errors",
		func(app *cli.App, arguments string, expected types.GomegaMatcher) {
			args, _ := cli.Split("app " + arguments)
			err := app.RunContext(context.TODO(), args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(expected)
		},
		Entry(
			"too many arguments",
			app("<arg>"),
			"a b c",
			Equal("too many arguments: \"b\""),
		),
		XEntry( // TODO Required arguments
			"required argument",
			app("<FILE>"),
			"",
			Equal("argument FILE required"),
		),
	)
})
