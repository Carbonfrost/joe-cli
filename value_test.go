package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Value", func() {

	Describe("Set", func() {

		DescribeTable("generic values",
			func(f *cli.Flag, arguments string, expected types.GomegaMatcher) {
				act := new(joeclifakes.FakeActionHandler)
				app := &cli.App{
					Name: "app",
					Flags: []*cli.Flag{
						f,
					},
					Action: act,
				}

				args, _ := cli.Split(arguments)
				app.RunContext(nil, args)
				captured := act.ExecuteArgsForCall(0)
				Expect(captured.Value("o")).To(expected)
			},
			Entry(
				"list",
				&cli.Flag{
					Name:  "o",
					Value: cli.List(),
				},
				"app -o a -o b",
				Equal([]string{"a", "b"}),
			),
			Entry(
				"list run-in",
				&cli.Flag{
					Name:  "o",
					Value: cli.List(),
				},
				"app -o a,b,c -o d",
				Equal([]string{"a", "b", "c", "d"}),
			),
			Entry(
				"map",
				&cli.Flag{
					Name:  "o",
					Value: &map[string]string{"existing": "values"}, // Existing values are overwritten
				},
				"app -o hello=world -o goodbye=earth",
				Equal(map[string]string{
					"hello":   "world",
					"goodbye": "earth",
				}),
			),
			Entry(
				"map run-in",
				&cli.Flag{
					Name:  "o",
					Value: cli.Map(),
				},
				"app -o hello=world,goodbye=earth -o aloha=mars",
				Equal(map[string]string{
					"hello":   "world",
					"goodbye": "earth",
					"aloha":   "mars",
				}),
			),
		)
	})
})
