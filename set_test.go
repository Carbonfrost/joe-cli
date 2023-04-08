package cli_test

import (
	"errors"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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

		Expect(err).To(PointTo(MatchFields(IgnoreExtras, Fields{
			"Code":      Equal(cli.ExpectedArgument),
			"Name":      Equal("<arg>"),
			"Err":       MatchError("expected 2 arguments"),
			"Value":     Equal(""),
			"Remaining": BeNil(),
		})))
		Expect(err).To(MatchError("expected 2 arguments for <arg>"))

		Expect(actualFlat).To(Equal(map[string][]string{
			"arg": {"<arg> first"},
		}))
	})

	DescribeTable("examples", func(arguments string, mode int, expected, expectedErr types.GomegaMatcher) {
		args, _ := cli.Split(arguments)
		set := &testFlagSet{
			args: []string{"arg"},
			counters: map[string]cli.ArgCounter{
				"arg":   cli.ArgCount(mode),
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
		Entry("nominal args bind", "app a b c", cli.TakeUntilNextFlag,
			Equal(map[string][]string{
				"arg": {"<arg> a", "<arg> b", "<arg> c"},
			}),
			Not(HaveOccurred())),

		Entry("nominal long flag", "app --long space", cli.TakeUntilNextFlag,
			HaveKeyWithValue("long", []string{"--long space"}),
			Not(HaveOccurred())),

		Entry("nominal short flag", "app -s a", cli.TakeUntilNextFlag,
			HaveKeyWithValue("short", []string{"-s a"}),
			Not(HaveOccurred())),

		Entry("short flag run-in", "app -space", cli.TakeUntilNextFlag,
			HaveKeyWithValue("short", []string{"-s pace"}),
			Not(HaveOccurred())),

		// This allows args and flags to be interspersed:
		// git add set.go -u set_test.go  -- is a valid invocation
		Entry("mix flags and args", "app a --boolt b", cli.TakeExceptForFlags,
			Equal(map[string][]string{
				"arg":   {"<arg> a", "<arg> b"},
				"boolt": {"--boolt "},
			}),
			Not(HaveOccurred())),

		// If an equal sign is present in the short flag syntax, it
		// is always interpreted as setting the value (including the
		// leading =)
		Entry("equal in short flag is its value", "app -s=pace", cli.TakeUntilNextFlag,
			HaveKeyWithValue("short", []string{"-s =pace"}),
			Not(HaveOccurred()),
		),

		// Equal sign leads to an error because value is unexpected
		Entry("equal in short flag causes error", "app -vt=always", cli.TakeUntilNextFlag,
			Equal(map[string][]string{
				"boolv": {"-v "}, // { "-v", ""}
			}),
			Equal(&cli.ParseError{
				Code:      cli.InvalidArgument,
				Name:      "-t",
				Err:       errors.New("option -t does not take a value"),
				Value:     "=always",
				Remaining: []string{"-t=always"},
			}),
		),

		Entry("stop on unknown long flag", "app --unknown rest of args", cli.TakeUntilNextFlag,
			BeEmpty(),
			Equal(&cli.ParseError{
				Code:      cli.UnknownOption,
				Name:      "--unknown",
				Err:       errors.New("unknown option: --unknown"),
				Value:     "",
				Remaining: []string{"--unknown", "rest", "of", "args"},
			}),
		),

		Entry("stop on unknown short flag", "app -u rest of args", cli.TakeUntilNextFlag,
			BeEmpty(),
			Equal(&cli.ParseError{
				Code:      cli.UnknownOption,
				Name:      "-u",
				Err:       errors.New("unknown option: -u"),
				Value:     "",
				Remaining: []string{"-u", "rest", "of", "args"},
			}),
		),

		Entry("stop on unknown short flag run-in", "app -tuvwx another", cli.TakeUntilNextFlag,
			Equal(map[string][]string{
				"boolt": {"-t "}, // {"-t", ""}
			}),
			Equal(&cli.ParseError{
				Code:      cli.UnknownOption,
				Name:      "-u",
				Err:       errors.New("unknown option: -u"),
				Value:     "",
				Remaining: []string{"-uvwx", "another"},
			}),
		),

		Entry("long flag missing required arg", "app --short", cli.TakeUntilNextFlag,
			BeEmpty(),
			And(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Code":      Equal(cli.ExpectedArgument),
					"Name":      Equal("--short"),
					"Err":       MatchError("expected argument"),
					"Value":     Equal(""),
					"Remaining": Equal([]string{"--short"}),
				})),
				MatchError("expected argument for --short"),
			),
		),

		Entry("long flag missing arg by default", "app --long", cli.TakeUntilNextFlag,
			BeEmpty(),
			And(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Code":      Equal(cli.ExpectedArgument),
					"Name":      Equal("--long"),
					"Err":       MatchError("expected argument"),
					"Value":     Equal(""),
					"Remaining": Equal([]string{"--long"}),
				})),
				MatchError("expected argument for --long"),
			),
		),

		Entry("short flag missing required arg", "app -s", cli.TakeUntilNextFlag,
			BeEmpty(),
			And(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Code":      Equal(cli.ExpectedArgument),
					"Name":      Equal("--short"),
					"Err":       MatchError("expected argument"),
					"Value":     Equal(""),
					"Remaining": Equal([]string{"-s"}),
				})),
				MatchError("expected argument for --short"),
			),
		),

		Entry("unexpected argument", "app arg -s a other args", cli.TakeUntilNextFlag,
			Equal(map[string][]string{
				"arg":   {"<arg> arg"},
				"short": {"-s a"},
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

func (t *testFlagSet) PositionalArgNames() []string {
	return t.args
}

func (t *testFlagSet) BehaviorFlags(name string) (optional bool) {
	return false
}

func (t *testFlagSet) LookupOption(name string) (cli.TransformFunc, cli.ArgCounter, cli.BindingState, bool) {
	c, ok := t.counters[name]
	return nil, c, nil, ok
}

func (t *testFlagSet) ResolveAlias(name string) (string, bool) {
	c, ok := t.aliases[name]
	if ok {
		return c, true
	}
	_, ok = t.counters[name]
	return name, ok
}

func (*testFlagSet) SetOccurrenceData(name string, v any) error {
	return nil
}

func (*testFlagSet) SetOccurrence(name string, values ...string) error {
	return nil
}

// Flatten actual one layer to simplify comparisons
func flattenParseResults(actual map[string][][]string) map[string][]string {
	actualFlat := map[string][]string{}
	for n, v := range actual {
		// skip over "", which represents the inputs
		if n == "" {
			continue
		}
		flat := make([]string, len(v))
		for i, occur := range v {
			flat[i] = strings.Join(occur, " ")
		}
		actualFlat[n] = flat
	}
	return actualFlat
}
