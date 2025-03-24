package cli_test

import (
	"context"
	"encoding"
	"flag"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"testing/fstest"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	joeclifakes "github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
				err := app.RunContext(context.Background(), args)
				Expect(err).NotTo(HaveOccurred())
				captured := cli.FromContext(cli.FromContext(act.ExecuteArgsForCall(0)))
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
				"name-value",
				&cli.Flag{
					Name:  "o",
					Value: &cli.NameValue{},
				},
				"app -o hello=world",
				Equal(&cli.NameValue{
					Name:  "hello",
					Value: "world",
				}),
			),
			Entry(
				"name-value arg counter semantics",
				&cli.Flag{
					Name:  "o",
					Value: &cli.NameValue{},
				},
				"app -o hello world",
				Equal(&cli.NameValue{
					Name:  "hello",
					Value: "world",
				}),
			),
			Entry(
				"name-value last occurrence wins",
				&cli.Flag{
					Name:  "o",
					Value: &cli.NameValue{},
				},
				"app -o hello=world -o goodbye=earth",
				Equal(&cli.NameValue{
					Name:  "goodbye",
					Value: "earth",
				}),
			),
			Entry(
				"name-value only name sets true",
				&cli.Flag{
					Name:  "o",
					Value: &cli.NameValue{},
				},
				"app -o hello",
				Equal(&cli.NameValue{
					Name:  "hello",
					Value: "true",
				}),
			),

			Entry(
				"name-values",
				&cli.Flag{
					Name:  "o",
					Value: cli.NameValues(),
				},
				"app -o hello=world -o goodbye=earth",
				Equal([]*cli.NameValue{
					{Name: "hello", Value: "world"},
					{Name: "goodbye", Value: "earth"},
				}),
			),
			Entry(
				"name-values resets values",
				&cli.Flag{
					Name:  "o",
					Value: cli.NameValues("existing", "values"), // Existing values are overwritten
				},
				"app -o hello=world -o goodbye=earth",
				Equal([]*cli.NameValue{
					{Name: "hello", Value: "world"},
					{Name: "goodbye", Value: "earth"},
				}),
			),
			Entry(
				"name-values run-in",
				&cli.Flag{
					Name:  "o",
					Value: cli.NameValues(),
				},
				"app -o hello=world,goodbye=earth -o aloha=mars",
				Equal([]*cli.NameValue{
					{Name: "hello", Value: "world"},
					{Name: "goodbye", Value: "earth"},
					{Name: "aloha", Value: "mars"},
				}),
			),
			Entry(
				"name-values merge",
				&cli.Flag{
					Name:    "o",
					Value:   cli.NameValues("default", "set"),
					Options: cli.Merge,
				},
				"app -o aloha=mars",
				Equal([]*cli.NameValue{
					{Name: "default", Value: "set"},
					{Name: "aloha", Value: "mars"},
				}),
			),
			Entry(
				"string appends given Merge",
				&cli.Flag{
					Name:    "o",
					Value:   new(string),
					Options: cli.Merge,
				},
				"app -o abc -o 123",
				Equal("abc 123"),
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
			Entry(
				"Duration",
				&cli.Flag{Name: "o", Value: cli.Duration()},
				"app -o 55ms",
				Equal(time.Millisecond*55),
			),
			Entry(
				"file set resets values",
				&cli.Flag{
					Name:  "o",
					Value: &cli.FileSet{Files: []string{"default"}},
				},
				"app -o a",
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Files": Equal([]string{"a"}),
				})),
			),
			Entry(
				"file set merge",
				&cli.Flag{
					Name:    "o",
					Value:   &cli.FileSet{Files: []string{"default"}},
					Options: cli.Merge,
				},
				"app -o a",
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Files": Equal([]string{"default", "a"}),
				})),
			),
			Entry(
				"BigInt",
				&cli.Flag{Name: "o", Value: cli.BigInt()},
				"app -o 15000",
				Equal(big.NewInt(15000)),
			),
			Entry(
				"BigFloat",
				&cli.Flag{Name: "o", Value: cli.BigFloat()},
				"app -o 150.2",
				WithTransform(func(v any) any {
					f, _ := v.(*big.Float).Float64()
					return f
				}, Equal(float64(150.2))),
			),
			Entry(
				"bytes",
				&cli.Flag{Name: "o", Value: cli.Bytes()},
				"app -o beadedfacade",
				Equal([]byte{0xbe, 0xad, 0xed, 0xfa, 0xca, 0xde}),
			),
			Entry(
				"bytes from AllowFileReference",
				&cli.Flag{
					Name: "o",
					// Note that using AllowFileReference causes the value to be
					// a literal file (and not hex bytes)
					Options: cli.AllowFileReference,
					Value:   cli.Bytes(),
				},
				"app -o literal",
				Equal([]byte("literal")),
			),
			Entry(
				"text unmarshaler",
				&cli.Flag{
					Name:  "o",
					Value: new(textMarshaler),
				},
				"app -o v",
				Equal(textMarshaler("v")),
			),
			Entry(
				"Time (via text unmarshaler)",
				&cli.Flag{
					Name:  "o",
					Value: new(time.Time),
				},
				"app -o 2021-11-02T00:00:00Z",
				Equal(time.Date(2021, 11, 02, 0, 0, 0, 0, time.UTC)),
			),
			Entry(
				"IP (via text unmarshaler)",
				&cli.Flag{Name: "o", Value: new(net.IP)},
				"app -o 127.0.0.1",
				Equal(net.ParseIP("127.0.0.1")),
			),
		)

		It("panics for invalid flag types", func() {
			Expect(func() {
				cli.Set(&struct{}{}, "OK")
			}).To(PanicWith("unsupported flag type: *struct {}"))
		})

		DescribeTable("errors",
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
				err := app.RunContext(context.Background(), args)
				Expect(err).To(HaveOccurred())
				Expect(err).To(expected)
			},
			Entry(
				"bytes non-hex",
				&cli.Flag{
					Name:  "o",
					Value: cli.Bytes(),
				},
				"app -o itsbad",
				MatchError("invalid bytes: encoding/hex: invalid byte: U+0069 'i'"),
			),
			Entry(
				"can't parse int",
				&cli.Flag{
					Name:  "o",
					Value: cli.Int(),
				},
				"app -o itsbad",
				MatchError("not a valid number: itsbad"),
			),
			Entry(
				"too big int",
				&cli.Flag{
					Name:  "o",
					Value: cli.Int8(),
				},
				"app -o 512",
				MatchError("value out of range: 512"),
			),
			Entry(
				"bad IP",
				&cli.Flag{
					Name:  "o",
					Value: cli.IP(),
				},
				"app -o 512.123.123.122",
				MatchError("not a valid IP address"),
			),
			Entry(
				"bad URL",
				&cli.Flag{
					Name:  "o",
					Value: cli.URL(),
				},
				"app -o ://missingscheme",
				MatchError(ContainSubstring("missing protocol scheme")),
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
			app.RunContext(context.Background(), args)
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
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
			app.RunContext(context.Background(), args)
			captured := cli.FromContext(act.ExecuteArgsForCall(0))
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

		It("is called when DisableSplitting is set", func() {
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
			app.RunContext(context.Background(), args)

			Expect(cv.calledDisableSplitting).To(BeTrue())
		})
	})

	Describe("Initializer convention", func() {

		It("is called and invoked", func() {
			act := new(joeclifakes.FakeAction)
			app := &cli.App{
				Name: "app",
				Flags: []*cli.Flag{
					{
						Name: "d",
						Value: &customValue{
							init: act,
						},
					},
				},
			}

			args, _ := cli.Split("app -d a")
			app.RunContext(context.Background(), args)
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})
	})
})

var _ = Describe("NameValue", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected *cli.NameValue) {
				actual := &cli.NameValue{}
				for _, a := range args {
					err := actual.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(actual).To(Equal(expected))
			},
			Entry(
				"nominal",
				[]string{"name=value"},
				&cli.NameValue{Name: "name", Value: "value"},
			),
			Entry(
				"escaped equal sign",
				[]string{"name\\=value=value"},
				&cli.NameValue{Name: "name=value", Value: "value"},
			),
			Entry(
				"separated by spaces",
				[]string{"name", "value"},
				&cli.NameValue{Name: "name", Value: "value"},
			),
			Entry(
				"key only",
				[]string{"name="},
				&cli.NameValue{Name: "name", Value: ""},
			),
		)
	})

	It("loads from file reference", func() {

		testFileSystem := fstest.MapFS{
			"world": {Data: []byte("file contents")},
		}

		app := &cli.App{
			FS: testFileSystem,
			Flags: []*cli.Flag{
				{
					Name:  "v",
					Value: &cli.NameValue{},
					// Slightly more interesting to do this in the Uses pipeline to ensure
					// the timing of the Initializer
					Uses: func(c *cli.Context) error {
						return c.NameValue("").SetAllowFileReference(true)
					},
				},
			},
		}

		// Doing this indirectly is more interesting because it examines the timing of
		// the Initializer.
		args, _ := cli.Split("app -v hello=@world")
		app.Run(args)
		Expect(app.Flags[0].Value.(*cli.NameValue).Value).To(Equal("file contents"))
	})

	It("loads from FileReferences with EachOccurrence", func() {

		testFileSystem := fstest.MapFS{
			"world":  {Data: []byte("Earth")},
			"planet": {Data: []byte("Mars")},
		}

		var values []*cli.NameValue
		binder := func(r *cli.NameValue) error {
			// Notice that we are able to use *NameValue here without having
			// to do copying ourselves.  The value is copied because NameValue.Copy()
			// exists.
			values = append(values, r)
			return nil
		}

		app := &cli.App{
			FS: testFileSystem,
			Flags: []*cli.Flag{
				{
					Name:    "v",
					Value:   &cli.NameValue{AllowFileReference: true},
					Options: cli.EachOccurrence,
					Uses:    cli.Bind(binder),
				},
			},
		}

		args, _ := cli.Split("app -v hello=@world -v hello2=@planet -v hello3=Ceres")
		app.Run(args)

		Expect(values).To(HaveLen(3))
		Expect(values[0].Name).To(Equal("hello"))
		Expect(values[0].Value).To(Equal("Earth"))
		Expect(values[1].Name).To(Equal("hello2"))
		Expect(values[1].Value).To(Equal("Mars"))
		Expect(values[2].Name).To(Equal("hello3"))
		Expect(values[2].Value).To(Equal("Ceres"))
	})

})

var _ = Describe("NameValues", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected []*cli.NameValue) {
				actual := cli.NameValues()
				for _, a := range args {
					err := cli.Set(actual, a)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(actual).To(Equal(&expected))
			},
			Entry(
				"nominal",
				[]string{"name=value"},
				[]*cli.NameValue{{Name: "name", Value: "value"}},
			),
			Entry(
				"two",
				[]string{"a=b", "c=d"},
				[]*cli.NameValue{{Name: "a", Value: "b"}, {Name: "c", Value: "d"}},
			),
			Entry(
				"inline escapes",
				[]string{"a=b\\,c=d"},
				[]*cli.NameValue{{Name: "a", Value: "b,c=d"}},
			),
		)
	})

})

var _ = Describe("NameValueCounter", func() {

	var (
		newCounter = func() cli.ArgCounter {
			return new(cli.NameValue).NewCounter()
		}
	)

	DescribeTable("examples",
		func(args []string) {
			actual := newCounter()
			for _, a := range args {
				err := actual.Take(a, true)
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(actual.Done()).NotTo(HaveOccurred())
		},
		Entry(
			"nominal",
			[]string{"name=value"},
		),
		Entry(
			"separated by spaces",
			[]string{"name", "value"},
		),
		Entry(
			"key only",
			[]string{"name="},
		),
	)

	DescribeTable("errors",
		func(args []string, expected string) {
			actual := newCounter()
			for _, a := range args {
				err := actual.Take(a, true)
				Expect(err).NotTo(HaveOccurred())
			}

			err := actual.Done()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		},
		Entry(
			"missing both",
			[]string{},
			"missing name and value",
		),
	)

})

func unwrap[V any](v V, _ any) V {
	return v
}

func addr[V any](v V) *V {
	return &v
}

type customValue struct {
	calledDisableSplitting bool
	init                   cli.Action
}

func (*customValue) Set(arg string) error      { return nil }
func (*customValue) String() string            { return "" }
func (c *customValue) Initializer() cli.Action { return c.init }
func (c *customValue) DisableSplitting() {
	c.calledDisableSplitting = true
}

type hasDereference struct {
	v any
}

func (*hasDereference) Set(string) error { return nil }
func (*hasDereference) String() string   { return "" }
func (d *hasDereference) Value() any {
	return d.v
}

type hasGetter struct {
	v any
}

func (*hasGetter) Set(string) error { return nil }
func (*hasGetter) String() string   { return "" }
func (d *hasGetter) Get() any {
	return d.v
}

type textMarshaler string

func (t *textMarshaler) UnmarshalText(text []byte) error {
	*t = textMarshaler(string(text))
	return nil
}

var _ encoding.TextUnmarshaler = (*textMarshaler)(nil)
var _ flag.Getter = (*hasGetter)(nil)
