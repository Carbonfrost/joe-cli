// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IndentStyle", func() {

	Describe("UnmarshalText", func() {

		DescribeTable("examples",
			func(text string, expected codec.IndentStyle) {
				var s codec.IndentStyle
				Expect(s.UnmarshalText([]byte(text))).NotTo(HaveOccurred())
				Expect(s).To(Equal(expected))
			},
			Entry("space", "space", codec.IndentSpace),
			Entry("tab", "tab", codec.IndentTab),
			Entry("spaces misspelling", "spaces", codec.IndentSpace),
			Entry("tabs misspelling", "tabs", codec.IndentTab),
			Entry("surrounding whitespace", "  tab  ", codec.IndentTab),
		)

		It("errors on an unknown value", func() {
			var s codec.IndentStyle
			Expect(s.UnmarshalText([]byte("nope"))).To(HaveOccurred())
		})
	})

	DescribeTable("String and MarshalText",
		func(s codec.IndentStyle, expected string) {
			Expect(s.String()).To(Equal(expected))

			text, err := s.MarshalText()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(text)).To(Equal(expected))
		},
		Entry("space (default zero value)", codec.IndentSpace, "space"),
		Entry("tab", codec.IndentTab, "tab"),
	)
})
