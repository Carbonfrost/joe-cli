// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package color_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/color"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mode", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(arg string, expected color.Mode) {
				actual := new(color.Mode)
				err := actual.Set(arg)

				Expect(err).NotTo(HaveOccurred())
				Expect(*actual).To(Equal(expected))
			},
			Entry("nominal", "auto", color.Auto),
			Entry("bool true", "true", color.Always),
			Entry("bool false", "false", color.Never),
			Entry("bool on", "on", color.Always),
			Entry("always", "always", color.Always),
			Entry("never", "never", color.Never),
		)

		DescribeTable("errors",
			func(arg string, expected string) {
				actual := new(color.Mode)
				err := actual.Set(arg)

				Expect(err).To(MatchError(expected))
			},
			Entry("error", "unknown", "invalid value: \"unknown\""),
		)
	})

	Describe("String", func() {
		DescribeTable("examples",
			func(arg color.Mode, expected string) {
				Expect(arg.String()).To(Equal(expected))
			},
			Entry("nominal", color.Auto, "auto"),
			Entry("bool true", color.Never, "never"),
			Entry("always", color.Always, "always"),
		)
	})

})
