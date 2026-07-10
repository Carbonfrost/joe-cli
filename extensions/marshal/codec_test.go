// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal_test

import (
	"bytes"
	"context"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	_ "github.com/Carbonfrost/joe-cli/extensions/marshal/codec/toml"

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
					"TOML",
					marshal.TOML,
					map[string]any{"parent": map[string]any{"child": 1}},
					"[parent]\n  child = 1\n",
				),
			)
		})

	})

})

var _ = Describe("CodecRegistry", func() {

	Describe("ProviderNames", func() {
		It("lists the registered codecs", func() {
			// The toml codec is registered via the blank import above; yaml has
			// no implementation and must not appear.
			Expect(marshal.CodecRegistry.ProviderNames()).To(ConsistOf("json", "toml"))
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

		It("returns an error for an unregistered codec", func() {
			_, err := marshal.CodecRegistry.New("yaml", nil)
			Expect(err).To(HaveOccurred())
		})
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
		Expect(capture.String()).To(Equal(
			"json\tdisallow_unknown_fields=false, indent_size=2, indent_style=space\n" +
				"toml\tdisallow_unknown_fields=false, indent_size=2, indent_style=space\n",
		))
	})

	// TODO There is a bug with shared state in the provider context services causing this
	// to unexpectedly pass
	XIt("generates an error on no codec registry", func() {
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
