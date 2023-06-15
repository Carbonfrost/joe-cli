package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
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
	DescribeTable("examples", func(in any, out string) {
		actual := cli.Quote(in)
		Expect(actual).To(Equal(out))
	},
		Entry("string", "text", `text`),
		Entry("escape string", "$'b", `'$'"'"'b'`),
		Entry("empty string", "", `''`),

		Entry("nil", nil, ""),
		Entry("bool", true, "true"),
		Entry("float32", float32(2.2), "2.2"),
		Entry("float64", float64(2.2), "2.2"),
		Entry("int", int(16), "16"),
		Entry("int16", int16(16), "16"),
		Entry("int32", int32(16), "16"),
		Entry("int64", int64(16), "16"),
		Entry("int8", int8(16), "16"),
		Entry("list", []string{"text,plus"}, "text,plus"),
		Entry("map", map[string]string{"key": "value"}, "key=value"),
		Entry("uint", uint(19), "19"),
		Entry("uint16", uint16(19), "19"),
		Entry("uint32", uint32(19), "19"),
		Entry("uint64", uint64(19), "19"),
		Entry("uint8", uint8(19), "19"),
		Entry("bytes", []byte{0xCE, 0xC3}, "cec3"),
		Entry("Value", cli.Octal(0o20), "0o20"),
		Entry("NameValues", cli.NameValues("key", "on"), "key=on"),
		Entry("Duration", 250*time.Second, "4m10s"),
		Entry("URL", unwrap(url.Parse("https://localhost")), "https://localhost"),
		Entry("URL ptr", addr(unwrap(url.Parse("https://localhost"))), "https://localhost"),
		Entry("Regexp", regexp.MustCompile("blc"), "blc"),
		Entry("Regexp ptr", addr(regexp.MustCompile("blc")), "blc"),
		Entry("IP", net.ParseIP("127.0.0.1"), "127.0.0.1"),
		Entry("IP ptr", addr(net.ParseIP("127.0.0.1")), "127.0.0.1"),
		Entry("big.Float", parseBigFloat("201.12"), "201.12"),
		Entry("big.Int", parseBigInt("200"), "200"),
		Entry("text marshal", new(fakeMarshal), "fake"),
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

var _ = Describe("SplitMap", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		Expect(cli.SplitMap(text)).To(expected)
	},
		Entry("escaped comma", "L=A\\,B", HaveKeyWithValue("L", "A,B")),
		Entry("escaped comma multiple", "L=A\\,B,M=Y\\,Z,N=W\\,X",
			And(
				HaveKeyWithValue("L", "A,B"),
				HaveKeyWithValue("M", "Y,Z"),
				HaveKeyWithValue("N", "W,X"),
			)),
		Entry("escaped comma trailing", "L=A\\,", HaveKeyWithValue("L", "A,")),
		// Comma is implied as escaped because there is no other KVP after it
		Entry("implied escaped comma", "L=A,B,C", HaveKeyWithValue("L", "A,B,C")),
		Entry("escaped equal", "L\\=A=B", HaveKeyWithValue("L=A", "B")),
		Entry("escaped equal trailing", "L\\==B", HaveKeyWithValue("L=", "B")),
		Entry("empty", "", BeEmpty()),
	)
})

var _ = Describe("RunContext", func() {
	DescribeTable("bind sub-command",
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
				Stderr: io.Discard,
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

var _ = Describe("ReadPasswordString", func() {

	// Note: use of term.ReadPassword limits ability to test

	It("generates the expected prompt", func() {
		var buf bytes.Buffer
		app := &cli.App{
			Name:   "any",
			Stderr: &buf,
			Stdin:  &fakeFD{strings.NewReader("my pass\n")},
			Action: func(c *cli.Context) {
				c.ReadPasswordString("Enter password: ")
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(buf.String()).To(Equal("Enter password: "))
	})

	It("returns error on non-file descriptor", func() {
		app := &cli.App{
			Name:   "any",
			Stdout: io.Discard,
			Stdin:  strings.NewReader("pass"),
			Action: func(c *cli.Context) error {
				_, err := c.ReadPasswordString("")
				return err
			},
		}

		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("stdin not tty"))
	})
})

var _ = Describe("ReadString", func() {

	It("reads from file descriptor", func() {
		var pass string
		app := &cli.App{
			Name:   "any",
			Stderr: io.Discard,
			Stdin:  &fakeFD{strings.NewReader("my pass\n")},
			Action: func(c *cli.Context) {
				pass, _ = c.ReadString("the prompt")
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(pass).To(Equal("my pass"))
	})

	It("generates the expected prompt", func() {
		var buf bytes.Buffer
		app := &cli.App{
			Name:   "any",
			Stderr: &buf,
			Stdin:  &fakeFD{strings.NewReader("my pass\n")},
			Action: func(c *cli.Context) {
				c.ReadString("Some prompt")
			},
		}

		_ = app.RunContext(context.Background(), []string{"app"})
		Expect(buf.String()).To(Equal("Some prompt"))
	})

	It("returns error on non-file descriptor", func() {
		app := &cli.App{
			Name:   "any",
			Stdout: io.Discard,
			Stdin:  strings.NewReader("my pass\n"),
			Action: func(c *cli.Context) error {
				_, err := c.ReadString("")
				return err
			},
		}

		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("stdin not tty"))
	})

	It("propogates inner reader error", func() {
		app := &cli.App{
			Name:   "any",
			Stdout: io.Discard,
			Stdin:  new(fakeFD),
			Action: func(c *cli.Context) error {
				_, err := c.ReadString("")
				return err
			},
		}

		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("inner reader error"))
	})
})

type fakeMarshal struct{}

func (*fakeMarshal) MarshalText() ([]byte, error) {
	return []byte("fake"), nil
}

type fakeFD struct {
	io.Reader
}

func (*fakeFD) Fd() uintptr {
	return 0
}

func (f *fakeFD) Read(p []byte) (n int, err error) {
	if f.Reader == nil {
		return 0, fmt.Errorf("inner reader error")
	}
	return f.Reader.Read(p)
}
