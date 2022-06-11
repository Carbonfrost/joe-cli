package cli_test

import (
	"errors"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("RawParse", func() {

	It("expected argument", func() {
		args, _ := cli.Split("app first")
		set := &testFlagSet{
			args: []string{"arg"},
			counters: map[string]cli.ArgCounter{
				"arg": cli.ArgCount(2),
			},
			aliases: map[string]string{},
		}

		actual, err := cli.RawParse(args, set, cli.RawSkipProgramName)
		actualFlat := flattenParseResults(actual)

		Expect(err).To(Equal(&cli.ParseError{
			Code:      cli.ExpectedArgument,
			Name:      "<arg>",
			Err:       errors.New("expected 2 arguments for <arg>"),
			Value:     "",
			Remaining: nil,
		}))
		Expect(actualFlat).To(Equal(map[string][]string{
			"arg": []string{"<arg> first"},
		}))
	})

	DescribeTable("examples", func(arguments string, expected, expectedErr types.GomegaMatcher) {
		args, _ := cli.Split(arguments)
		set := &testFlagSet{
			args: []string{"arg"},
			counters: map[string]cli.ArgCounter{
				"arg":   cli.ArgCount(cli.TakeUntilNextFlag),
				"long":  cli.DefaultFlagCounter(),
				"short": cli.ArgCount(1),
				"boolt": cli.NoArgs(),
				"boolv": cli.NoArgs(),
			},
			aliases: map[string]string{
				"s": "short",
				"t": "boolt",
				"v": "boolv",
			},
		}

		actual, err := cli.RawParse(args, set, cli.RawSkipProgramName)

		// Flatten actual one layer to simplify comparisons
		actualFlat := flattenParseResults(actual)

		Expect(err).To(expectedErr)
		Expect(actualFlat).To(expected)
	},
		Entry("nominal args bind", "app a b c",
			Equal(map[string][]string{
				"arg": []string{"<arg> a", "<arg> b", "<arg> c"},
			}),
			Not(HaveOccurred())),

		Entry("nominal long flag", "app --long space",
			HaveKeyWithValue("long", []string{"--long space"}),
			Not(HaveOccurred())),

		Entry("nominal short flag", "app -s a",
			HaveKeyWithValue("short", []string{"-s a"}),
			Not(HaveOccurred())),

		Entry("short flag run-in", "app -space",
			HaveKeyWithValue("short", []string{"-s pace"}),
			Not(HaveOccurred())),

		// If an equal sign is present in the short flag syntax, it
		// is always interpreted as setting the value (including the
		// leading =)
		Entry("equal in short flag is its value", "app -s=pace",
			HaveKeyWithValue("short", []string{"-s =pace"}),
			Not(HaveOccurred()),
		),

		// Equal sign leads to an error because value is unexpected
		Entry("equal in short flag causes error", "app -vt=always",
			Equal(map[string][]string{
				"boolv": []string{"-v "}, // { "-v", ""}
			}),
			Equal(&cli.ParseError{
				Code:      cli.InvalidArgument,
				Name:      "-t",
				Err:       errors.New("option -t does not take a value"),
				Value:     "=always",
				Remaining: []string{"-t=always"},
			}),
		),

		Entry("stop on unknown long flag", "app --unknown rest of args",
			BeEmpty(),
			Equal(&cli.ParseError{
				Code:      cli.UnknownOption,
				Name:      "--unknown",
				Err:       errors.New("unknown option: --unknown"),
				Value:     "",
				Remaining: []string{"--unknown", "rest", "of", "args"},
			}),
		),

		Entry("stop on unknown short flag", "app -u rest of args",
			BeEmpty(),
			Equal(&cli.ParseError{
				Code:      cli.UnknownOption,
				Name:      "-u",
				Err:       errors.New("unknown option: -u"),
				Value:     "",
				Remaining: []string{"-u", "rest", "of", "args"},
			}),
		),

		Entry("stop on unknown short flag run-in", "app -tuvwx another",
			Equal(map[string][]string{
				"boolt": []string{"-t "}, // {"-t", ""}
			}),
			Equal(&cli.ParseError{
				Code:      cli.UnknownOption,
				Name:      "-u",
				Err:       errors.New("unknown option: -u"),
				Value:     "",
				Remaining: []string{"-uvwx", "another"},
			}),
		),

		Entry("long flag missing required arg", "app --short",
			BeEmpty(),
			Equal(&cli.ParseError{
				Code:      cli.ExpectedArgument,
				Name:      "--short",
				Err:       errors.New("expected argument for --short"),
				Value:     "",
				Remaining: []string{"--short"},
			}),
		),

		Entry("long flag missing arg by default", "app --long",
			BeEmpty(),
			Equal(&cli.ParseError{
				Code:      cli.ExpectedArgument,
				Name:      "--long",
				Err:       errors.New("expected argument for --long"),
				Value:     "",
				Remaining: []string{"--long"},
			}),
		),

		Entry("unexpected argument", "app arg -s a other args",
			Equal(map[string][]string{
				"arg":   []string{"<arg> arg"},
				"short": []string{"-s a"},
			}),
			Equal(&cli.ParseError{
				Code:  cli.UnexpectedArgument,
				Name:  "",
				Err:   errors.New(`unexpected argument "other"`),
				Value: "other",
				// TODO This should also include args []string{"other", "args"}
				Remaining: []string{"other"},
			}),
		),
	)
})

type testFlagSet struct {
	args     []string
	counters map[string]cli.ArgCounter
	aliases  map[string]string
}

func (t *testFlagSet) Args() []string {
	return t.args
}

func (t *testFlagSet) IsOptionalValue(name string) bool {
	return false
}

func (t *testFlagSet) Lookup(name string) (cli.ArgCounter, bool) {
	c, ok := t.counters[name]
	return c, ok
}

func (t *testFlagSet) FlagName(name string) (string, bool) {
	c, ok := t.aliases[name]
	if ok {
		return c, true
	}
	_, ok = t.counters[name]
	return name, ok
}

// Flatten actual one layer to simplify comparisons
func flattenParseResults(actual map[string][][]string) map[string][]string {
	actualFlat := map[string][]string{}
	for n, v := range actual {
		flat := make([]string, len(v))
		for i, occur := range v {
			flat[i] = strings.Join(occur, " ")
		}
		actualFlat[n] = flat
	}
	return actualFlat
}
