package cli_test

import (
	"net"
	"net/url"
	"regexp"

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
				act := new(joeclifakes.FakeAction)
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
			Entry(
				"URL",
				&cli.Flag{Name: "o", Value: cli.URL()},
				"app -o https://localhost.example:1619",
				Equal(unwrap(url.Parse("https://localhost.example:1619"))),
			),
			Entry(
				"Regexp",
				&cli.Flag{Name: "o", Value: cli.Regexp()},
				"app -o [CGAT]{512}",
				Equal(regexp.MustCompile("[CGAT]{512}")),
			),
			Entry(
				"IP",
				&cli.Flag{Name: "o", Value: cli.IP()},
				"app -o 127.0.0.1",
				Equal(net.ParseIP("127.0.0.1")),
			),
		)

		DescribeTable("List flag examples", func(arguments string, expected types.GomegaMatcher) {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:  "s",
						Value: cli.List(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split(arguments)
			app.RunContext(nil, args)
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.List("s")).To(expected)
		},
			Entry("escaped comma", "app -s 'A\\,B,C'", ContainElements("A,B", "C")),
		)

		DescribeTable("Map flag examples", func(arguments string, expected types.GomegaMatcher) {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name:  "s",
						Value: cli.Map(),
					},
				},
				Action: act,
			}

			args, _ := cli.Split(arguments)
			app.RunContext(nil, args)
			captured := act.ExecuteArgsForCall(0)
			Expect(captured.Map("s")).To(expected)
		},
			Entry("escaped comma", "app -s 'L=A\\,B'", HaveKeyWithValue("L", "A,B")),
			Entry("escaped comma multiple", "app -s 'L=A\\,B,M=Y\\,Z,N=W\\,X'",
				And(
					HaveKeyWithValue("L", "A,B"),
					HaveKeyWithValue("M", "Y,Z"),
					HaveKeyWithValue("N", "W,X"),
				)),
			Entry("escaped comma trailing", "app -s 'L=A\\,'", HaveKeyWithValue("L", "A,")),
			// Comma is implied as escaped because there is no other KVP after it
			Entry("implied escaped comma", "app -s 'L=A,B,C'", HaveKeyWithValue("L", "A,B,C")),
			Entry("escaped equal", "app -s 'L\\=A=B'", HaveKeyWithValue("L=A", "B")),
			Entry("escaped equal trailing", "app -s 'L\\==B'", HaveKeyWithValue("L=", "B")),
		)

	})
})

func unwrap(v, _ interface{}) interface{} {
	return v
}
