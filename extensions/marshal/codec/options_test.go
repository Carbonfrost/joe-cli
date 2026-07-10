// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Options", func() {

	Describe("List", func() {

		It("is empty when nothing is set", func() {
			Expect(codec.Options{}.List()).To(BeEmpty())
		})

		It("omits SetIndent when the indent size is zero", func() {
			opts := codec.Options{IndentStyle: codec.IndentTab}.List()
			Expect(opts).To(BeEmpty())
		})

		DescribeTable("SetIndent from indent size and style",
			func(o codec.Options, expected string) {
				c, err := codec.WithOptions(codec.NewJSONCodec(), o.List()...)
				Expect(err).NotTo(HaveOccurred())

				out, err := codec.Codec{Interface: c}.Marshal(map[string]any{"a": 1})
				Expect(err).NotTo(HaveOccurred())
				Expect(string(out)).To(Equal(expected))
			},
			Entry(
				"two spaces",
				codec.Options{IndentSize: 2, IndentStyle: codec.IndentSpace},
				"{\n  \"a\": 1\n}\n",
			),
			Entry(
				"default style is space",
				codec.Options{IndentSize: 2},
				"{\n  \"a\": 1\n}\n",
			),
			Entry(
				"one tab",
				codec.Options{IndentSize: 1, IndentStyle: codec.IndentTab},
				"{\n\t\"a\": 1\n}\n",
			),
		)
	})
})
