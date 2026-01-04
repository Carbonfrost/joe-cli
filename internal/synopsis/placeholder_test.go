// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package synopsis_test

import (
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("usage", func() {

	Describe("parse", func() {

		DescribeTable("extract placeholders",
			func(text string, expected []string) {
				Expect(synopsis.ParseUsage(text).Placeholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", []string{}),
			Entry("placeholder", "{PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("placeholder used twice", "{PLACEHOLDER} {PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("2 placeholders", "{A} {B}", []string{"A", "B"}),
			Entry("2 placeholders with indexes", "{1:A} {0:B}", []string{"B", "A"}),
		)

		DescribeTable("without placeholders text",
			func(text string, expected string) {
				Expect(synopsis.ParseUsage(text).WithoutPlaceholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", "Literal text"),
			Entry("placeholder", "Load configuration from {FILE}s", "Load configuration from FILEs"),
		)
	})
})
