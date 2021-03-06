package cli_test

import (
	"context"
	"io/ioutil"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
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

var _ = Describe("Quote", func() {
	DescribeTable("examples", func(in, out string) {
		actual := cli.Quote(in)
		Expect(actual).To(Equal(out))
	},
		Entry("b", "$'b", `'$'"'"'b'`),
		Entry("empty", "", `''`),
	)
})

var _ = Describe("Join", func() {
	DescribeTable("examples", func(in []string, expected string) {
		actual := cli.Join(in)
		Expect(actual).To(Equal(expected))
	},
		Entry("nominal", []string{"s"}, `s`),
		Entry("empty", []string{}, ""),
		Entry("whitespace", []string{"a b", "c d"}, "'a b' 'c d'"),
	)
})

var _ = Describe("RunContext", func() {
	DescribeTable("bind subcommand",
		func(arguments string, expectedGlobal types.GomegaMatcher, expectedSub types.GomegaMatcher) {
			var (
				global commanderValues
				sub    mappedValues
			)
			var app = &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "global",
						Value: &global.Global,
					},
				},
				Commands: []*cli.Command{
					{
						Name: "sub",
						Flags: []*cli.Flag{
							{
								Name:  "flag1",
								Value: &sub.Flag1,
							},
						},
						Action: cli.ActionFunc(func(*cli.Context) error {
							global.Command = "sub"
							return nil
						}),
					},
				},
				Stderr: ioutil.Discard,
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
		Entry(
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
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:    "flag1",
						Aliases: []string{"a"},
						Value:   &result.Flag1,
					},
					{
						Name:    "flag2",
						Aliases: []string{"b"},
						Value:   &result.Flag2,
					},
					{
						Name:    "flag3",
						Aliases: []string{"c"},
						Value:   &result.Flag3,
					},
				},
				Args: []*cli.Arg{
					{
						Name:  "arg",
						Value: &result.Arg,
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
		Entry(
			"double dash",
			"-- --flag1",
			Equal(mappedValues{
				Flag1: false,
				Flag2: false,
				Arg:   "--flag1",
			}),
		),
		Entry(
			"inline booleans",
			"-ab",
			Equal(mappedValues{
				Flag1: true,
				Flag2: true,
			}),
		),
		Entry(
			"inline parameter",
			"-acHasValue",
			Equal(mappedValues{
				Flag1: true,
				Flag2: false,
				Flag3: "HasValue",
			}),
		),
		Entry(
			"erroneous use of long syntax with short",
			"--a --b --c Value",
			Equal(mappedValues{
				Flag1: true,
				Flag2: true,
				Flag3: "Value",
			}),
		),
	)

	DescribeTable("bind args and flags errors",
		func(app *cli.App, arguments string, expected types.GomegaMatcher) {
			args, _ := cli.Split("app " + arguments)
			err := app.RunContext(context.TODO(), args)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		},
		Entry(
			"too many arguments",
			&cli.App{
				Args: []*cli.Arg{
					{
						Name: "a",
					},
				},
			},
			"a b c",
			Equal("unexpected argument \"b\""),
		),
		Entry(
			"required argument",
			&cli.App{
				Args: []*cli.Arg{
					{
						Name: "FILE",
						NArg: 1,
					},
				},
			},
			"",
			Equal("expected argument"),
		),
		Entry(
			"missing command",
			&cli.App{
				Flags: []*cli.Flag{
					{
						Name: "flag1",
					},
				},
				Commands: []*cli.Command{
					{
						Name: "sub",
					},
				},
			},
			"unknown",
			Equal(`"unknown" is not a command`),
		),
	)
})
