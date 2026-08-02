// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	_ "github.com/Carbonfrost/joe-cli/extensions/marshal/codec/toml"
	_ "github.com/Carbonfrost/joe-cli/extensions/marshal/codec/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Codec", func() {

	Describe("New", func() {

		Describe("option support", func() {

			DescribeTable("examples",
				func(co marshal.Codec, option codec.Option, expected types.GomegaMatcher) {
					_, err := co.New(option)
					Expect(err).To(expected)
				},
				Entry(
					"JSON DisallowUnknownFields",
					marshal.JSON,
					marshal.DisallowUnknownFields(),
					Not(HaveOccurred()),
				),
				Entry(
					"TOML DisallowUnknownFields",
					marshal.TOML,
					marshal.DisallowUnknownFields(),
					Not(HaveOccurred()),
				),
				Entry(
					"JSON WithIndent",
					marshal.JSON,
					marshal.WithIndent("  "),
					Not(HaveOccurred()),
				),
				Entry(
					"TOML WithIndent",
					marshal.TOML,
					marshal.WithIndent("  "),
					Not(HaveOccurred()),
				),
			)
		})

		Describe("WithIndent", func() {

			DescribeTable("indents encoded output",
				func(co marshal.Codec, value any, expected string) {
					c, err := co.New(marshal.WithIndent("  "))
					Expect(err).NotTo(HaveOccurred())

					out, err := codec.Codec{Interface: c}.Marshal(value)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(out)).To(Equal(expected))
				},
				Entry(
					"JSON",
					marshal.JSON,
					map[string]any{"a": 1},
					"{\n  \"a\": 1\n}\n",
				),
				Entry(
					"YAML",
					marshal.YAML,
					map[string]any{"parent": map[string]any{"child": 1}},
					"parent:\n  child: 1\n",
				),
				Entry(
					"TOML",
					marshal.TOML,
					map[string]any{"parent": map[string]any{"child": 1}},
					"[parent]\n  child = 1\n",
				),
			)
		})

		Describe("DisallowUnknownFields", func() {
			DescribeTable("examples", func(c marshal.Codec, in string) {
				impl, _ := c.New(marshal.DisallowUnknownFields())
				actual := struct{ K string }{}
				err := impl.UnmarshalRead(bytes.NewReader([]byte(in)), &actual)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(
					Or(
						ContainSubstring("unknown"),
						ContainSubstring("fields in the document are missing"),
					),
				))
			},
				Entry("json", marshal.JSON, `{"unknown": 2}`),
				Entry("yaml", marshal.YAML, `unknown: 2`),
				Entry("toml", marshal.TOML, "unknown = 2"),
			)

		})

	})

})

var _ = Describe("CodecRegistry", func() {

	Describe("ProviderNames", func() {
		It("lists the registered codecs", func() {
			// The toml codec is registered via the blank import above; yaml has
			// no implementation and must not appear.
			Expect(marshal.CodecRegistry.ProviderNames()).To(ConsistOf("json", "toml", "yaml"))
		})
	})

	Describe("New", func() {
		It("creates a codec from Options", func() {
			actual, err := marshal.CodecRegistry.New("json", map[string]string{
				"disallow_unknown_fields": "true",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).NotTo(BeNil())
			Expect(actual).To(BeAssignableToTypeOf(codec.NewJSONCodec()))
		})

		It("returns an error for an unknown codec", func() {
			_, err := marshal.CodecRegistry.New("xml", nil)
			Expect(err).To(HaveOccurred())
		})
	})

	It("creates a codec from Options", func() {
		co, err := marshal.CodecRegistry.New("json", codec.Options{
			IndentSize: 4,
			EscapeHTML: true,
		})
		Expect(err).NotTo(HaveOccurred())

		c := codec.Codec{co.(codec.Interface)}
		actual, _ := c.Marshal(struct {
			F string
		}{F: "<o></o>"})
		Expect(string(actual)).To(MatchJSON(`{"F": "\u003co\u003e\u003c/o\u003e"}`))
		Expect(string(actual)).To(MatchRegexp(`(?m)^    "F"`)) // indented
	})
})

var _ = Describe("ListCodecs", func() {

	It("prints the registered codecs then exits", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Flags: []*cli.Flag{
				{
					Name:  "list-codec",
					Value: new(bool),
					Uses:  marshal.ListCodecs(),
				},
			},
			Uses: marshal.CodecRegistry,
		}

		// The list-codec flag uses cli.Exits, so the app exits after printing.
		_ = app.RunContext(context.Background(), []string{"app", "--list-codec"})
		lines := slices.Collect(strings.Lines(capture.String()))
		Expect(lines).To(ConsistOf(
			"json\tdisallow_unknown_fields=false, escape_html=false, indent_size=2, indent_style=space\n",
			"toml\tdisallow_unknown_fields=false, indent_size=2, indent_style=space\n",
			"yaml\tdisallow_unknown_fields=false, indent_size=2, indent_style=space\n",
		))
	})

	It("generates an error on no codec registry", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Flags: []*cli.Flag{
				{
					Name:  "list-codec",
					Value: new(bool),
					Uses:  marshal.ListCodecs(),
				},
			},
		}

		err := app.RunContext(context.Background(), []string{"app", "--list-codec"})
		Expect(err).To(MatchError("no codecs registered"))
	})
})

var _ = Describe("Dump", func() {

	It("prints the value as JSON when no context provider", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Action: marshal.Dump(
				struct {
					F string
					L string
				}{F: "O", L: "D"},
			),
			Uses: marshal.CodecRegistry,
		}

		// The list-codec flag uses cli.Exits, so the app exits after printing.
		err := app.RunContext(context.Background(), []string{"app"})
		Expect(err).NotTo((HaveOccurred()))
		Expect(capture.String()).To(MatchJSON(`{"F": "O", "L": "D"}`))
	})

	It("configures via context provider", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Action: marshal.Dump(struct {
				Name  string           `toml:"name"`
				ID    uint             `toml:"id"`
				Table []map[string]int `toml:"table"`
			}{
				Name: "J",
				ID:   234,
				Table: []map[string]int{
					{"t": 3},
				},
			}),
			Uses: cli.Pipeline(
				marshal.NewCodecProvider(),
			),
		}

		// TODO An upstream bug in providers prevents testing --output-arg indent_size=2
		args, _ := cli.Split("app --output=toml,indent_size=2")
		_ = app.RunContext(context.Background(), args)
		Expect(capture.String()).To(Equal("name = 'J'\nid = 234\n\n[[table]]\n  t = 3\n\n"))

	})

})

var _ = Describe("Dumper", func() {

	It("prints the value as JSON when no context provider", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Action: func(c *cli.Context) error {
				return marshal.Dumper{}.Evaluate(c, map[string]string{"F": "O"}, func(any) error {
					return nil
				})
			},
		}

		Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
		Expect(capture.String()).To(MatchJSON(`{"F": "O"}`))
	})

	It("prints the value using the codec configured in the context", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Uses:   marshal.NewCodecProvider(),
			Action: func(c *cli.Context) error {
				return marshal.Dumper{}.Evaluate(c, map[string]string{"name": "J"}, func(any) error {
					return nil
				})
			},
		}

		args, _ := cli.Split("app --output=toml")
		Expect(app.RunContext(context.Background(), args)).NotTo(HaveOccurred())
		Expect(capture.String()).To(Equal("name = 'J'\n\n"))
	})

	It("always yields the value", func() {
		var capture bytes.Buffer
		var yielded []any
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Action: func(c *cli.Context) error {
				return marshal.Dumper{}.Evaluate(c, "hello", func(v any) error {
					yielded = append(yielded, v)
					return nil
				})
			},
		}

		Expect(app.RunContext(context.Background(), []string{"app"})).NotTo(HaveOccurred())
		Expect(yielded).To(Equal([]any{"hello"}))
	})

	It("propagates the error from the yielder", func() {
		var capture bytes.Buffer
		expected := errors.New("stop")
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Action: func(c *cli.Context) error {
				return marshal.Dumper{}.Evaluate(c, "hello", func(any) error {
					return expected
				})
			},
		}

		Expect(app.RunContext(context.Background(), []string{"app"})).To(MatchError(expected))
	})

	It("dumps and yields each value within an expression", func() {
		var capture bytes.Buffer
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Args: []*cli.Arg{
				{
					Name: "start",
					NArg: -2,
				},
				{
					Name: "e",
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name:     "print",
								Evaluate: marshal.Dumper{},
							},
						},
					},
				},
			},
			Action: func(c *cli.Context) error {
				items := make([]any, 0)
				for _, v := range c.List("start") {
					items = append(items, v)
				}
				return expr.FromContext(c, "e").Evaluate(c, items...)
			},
		}

		args, _ := cli.Split("app a b -print")
		Expect(app.RunContext(context.Background(), args)).NotTo(HaveOccurred())
		Expect(capture.String()).To(Equal("\"a\"\n\n\"b\"\n\n"))
	})

})

var _ expr.Evaluator = marshal.Dumper{}
