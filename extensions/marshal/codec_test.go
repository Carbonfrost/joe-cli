// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal_test

import (
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
