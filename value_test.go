package cli_test

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"time"

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
				app.RunContext(context.TODO(), args)
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
				"list disable splitting",
				&cli.Flag{
					Name:    "o",
					Value:   cli.List(),
					Options: cli.DisableSplitting,
				},
				"app -o a,b,c -o d",
				Equal([]string{"a,b,c", "d"}),
			),
			Entry(
				"list resets values",
				&cli.Flag{
					Name:  "o",
					Value: &([]string{"this value is lost"}),
				},
				"app -o a",
				Equal([]string{"a"}),
			),
			Entry(
				"list merge",
				&cli.Flag{
					Name:    "o",
					Value:   &([]string{"default"}),
					Options: cli.Merge,
				},
				"app -o a",
				Equal([]string{"default", "a"}),
			),
			Entry(
				"map",
				&cli.Flag{
					Name:  "o",
					Value: cli.Map(),
				},
				"app -o hello=world -o goodbye=earth",
				Equal(map[string]string{
					"hello":   "world",
					"goodbye": "earth",
				}),
			),
			Entry(
				"map resets values",
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
				"map merge",
				&cli.Flag{
					Name:    "o",
					Value:   &(map[string]string{"default": "set"}),
					Options: cli.Merge,
				},
				"app  -o aloha=mars",
				Equal(map[string]string{
					"default": "set",
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
			app.RunContext(context.TODO(), args)
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
			app.RunContext(context.TODO(), args)
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

	Describe("DisableSplitting convention", func() {
		cv := new(customValue)
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:    "d",
					Options: cli.DisableSplitting,
					Value:   cv,
				},
			},
		}

		args, _ := cli.Split("app -d a")
		app.RunContext(context.TODO(), args)

		Expect(cv.calledDisableSplitting).To(BeTrue())
	})
})

var _ = Describe("Lookup", func() {
	Describe("accessors", func() {
		DescribeTable("examples",
			func(v interface{}, lookup func(cli.Lookup) interface{}, expected types.GomegaMatcher) {
				lk := cli.LookupValues{"a": v}
				Expect(lookup(lk)).To(expected)
				Expect(lk.Value("a")).To(expected)
			},
			Entry(
				"bool",
				cli.Bool(),
				func(lk cli.Lookup) interface{} { return lk.Bool("a") },
				Equal(false),
			),

			Entry(
				"File",
				&cli.File{},
				func(lk cli.Lookup) interface{} { return lk.File("a") },
				Equal(&cli.File{}),
			),
			Entry(
				"Float32",
				cli.Float32(),
				func(lk cli.Lookup) interface{} { return lk.Float32("a") },
				Equal(float32(0)),
			),
			Entry(
				"Float64",
				cli.Float64(),
				func(lk cli.Lookup) interface{} { return lk.Float64("a") },
				Equal(float64(0)),
			),
			Entry(
				"Int",
				cli.Int(),
				func(lk cli.Lookup) interface{} { return lk.Int("a") },
				Equal(int(0)),
			),
			Entry(
				"Int16",
				cli.Int16(),
				func(lk cli.Lookup) interface{} { return lk.Int16("a") },
				Equal(int16(0)),
			),
			Entry(
				"Int32",
				cli.Int32(),
				func(lk cli.Lookup) interface{} { return lk.Int32("a") },
				Equal(int32(0)),
			),
			Entry(
				"Int64",
				cli.Int64(),
				func(lk cli.Lookup) interface{} { return lk.Int64("a") },
				Equal(int64(0)),
			),
			Entry(
				"Int8",
				cli.Int8(),
				func(lk cli.Lookup) interface{} { return lk.Int8("a") },
				Equal(int8(0)),
			),
			Entry(
				"Duration",
				cli.Duration(),
				func(lk cli.Lookup) interface{} { return lk.Duration("a") },
				Equal(time.Duration(0)),
			),
			Entry(
				"List",
				cli.List(),
				func(lk cli.Lookup) interface{} { return lk.List("a") },
				BeAssignableToTypeOf([]string{}),
			),
			Entry(
				"Map",
				cli.Map(),
				func(lk cli.Lookup) interface{} { return lk.Map("a") },
				BeAssignableToTypeOf(map[string]string{}),
			),
			Entry(
				"String",
				cli.String(),
				func(lk cli.Lookup) interface{} { return lk.String("a") },
				Equal(""),
			),
			Entry(
				"UInt",
				cli.UInt(),
				func(lk cli.Lookup) interface{} { return lk.UInt("a") },
				Equal(uint(0)),
			),
			Entry(
				"UInt16",
				cli.UInt16(),
				func(lk cli.Lookup) interface{} { return lk.UInt16("a") },
				Equal(uint16(0)),
			),
			Entry(
				"UInt32",
				cli.UInt32(),
				func(lk cli.Lookup) interface{} { return lk.UInt32("a") },
				Equal(uint32(0)),
			),
			Entry(
				"UInt64",
				cli.UInt64(),
				func(lk cli.Lookup) interface{} { return lk.UInt64("a") },
				Equal(uint64(0)),
			),
			Entry(
				"UInt8",
				cli.UInt8(),
				func(lk cli.Lookup) interface{} { return lk.UInt8("a") },
				Equal(uint8(0)),
			),

			Entry(
				"URL",
				cli.URL(),
				func(lk cli.Lookup) interface{} { return lk.URL("a") },
				BeAssignableToTypeOf(&url.URL{}),
			),
			Entry(
				"Regexp",
				cli.Regexp(),
				func(lk cli.Lookup) interface{} { return lk.Regexp("a") },
				BeAssignableToTypeOf(&regexp.Regexp{}),
			),
			Entry(
				"IP",
				cli.IP(),
				func(lk cli.Lookup) interface{} { return lk.IP("a") },
				BeAssignableToTypeOf(net.IP{}),
			),
		)
	})

	Describe("conversion", func() {
		DescribeTable("examples",
			func(v interface{}, text string, expected types.GomegaMatcher) {
				f := &cli.Flag{Value: v}
				c := cli.InitializeFlag(f)

				f.Set(text)
				Expect(c.Value("")).To(expected)
			},
			Entry(
				"bool",
				cli.Bool(),
				"true",
				Equal(true),
			),

			Entry(
				"Float32",
				cli.Float32(),
				"2.0",
				Equal(float32(2.0)),
			),
			Entry(
				"Float64",
				cli.Float64(),
				"2.0",
				Equal(float64(2.0)),
			),
			Entry(
				"Int",
				cli.Int(),
				"16",
				Equal(int(16)),
			),
			Entry(
				"Int16",
				cli.Int16(),
				"16",
				Equal(int16(16)),
			),
			Entry(
				"Int32",
				cli.Int32(),
				"16",
				Equal(int32(16)),
			),
			Entry(
				"Int64",
				cli.Int64(),
				"16",
				Equal(int64(16)),
			),
			Entry(
				"Int8",
				cli.Int8(),
				"16",
				Equal(int8(16)),
			),
			Entry(
				"List",
				cli.List(),
				"text,plus",
				Equal([]string{"text", "plus"}),
			),
			Entry(
				"Map",
				cli.Map(),
				"key=value",
				Equal(map[string]string{"key": "value"}),
			),
			Entry(
				"String",
				cli.String(),
				"text",
				Equal("text"),
			),
			Entry(
				"UInt",
				cli.UInt(),
				"19",
				Equal(uint(19)),
			),
			Entry(
				"UInt16",
				cli.UInt16(),
				"19",
				Equal(uint16(19)),
			),
			Entry(
				"UInt32",
				cli.UInt32(),
				"19",
				Equal(uint32(19)),
			),
			Entry(
				"UInt64",
				cli.UInt64(),
				"19",
				Equal(uint64(19)),
			),
			Entry(
				"UInt8",
				cli.UInt8(),
				"19",
				Equal(uint8(19)),
			),

			Entry(
				"URL",
				cli.URL(),
				"https://localhost",
				Equal(unwrap(url.Parse("https://localhost"))),
			),
			Entry(
				"Regexp",
				cli.Regexp(),
				"blc",
				Equal(regexp.MustCompile("blc")),
			),
			Entry(
				"IP",
				cli.IP(),
				"127.0.0.1",
				Equal(net.ParseIP("127.0.0.1")),
			),
		)
	})
})

func unwrap(v, _ interface{}) interface{} {
	return v
}

type customValue struct {
	calledDisableSplitting bool
}

func (*customValue) Set(arg string) error { return nil }
func (*customValue) String() string       { return "" }
func (c *customValue) DisableSplitting() {
	c.calledDisableSplitting = true
}
