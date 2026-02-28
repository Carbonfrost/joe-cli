// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expander_test

import (
	"os"
	"time"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Interface", func() {

	os.Setenv("ENV_VAR", "an env var")

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		expander := expander.Prefix("env", expander.Env())
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("os env", "%(env.ENV_VAR)", Equal("an env var")),
		Entry("os env non-existing", "%(env.ENV_VAR__NON_EXISTENT)", Equal("<nil>")),
	)
})

var _ = Describe("Unknown", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		expander := expander.Unknown()
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("nominal", "%(var)", Equal("%!(unknown: var)")),
	)
})

var _ = Describe("Map", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		expander := expander.Map{"a": "b"}
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("map", "%(a)", Equal("b")),
		Entry("map non-existing", "%(unknown)", Equal("<nil>")),
	)
})

var _ = Describe("Time", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		expander := expander.Prefix("time",
			expander.Time(time.Date(2026, 2, 1, 20, 30, 33, 300, time.UTC)),
		)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("Day", "%(time.day)", Equal("1")),
		Entry("Hour12", "%(time.hour12)", Equal("8")),
		Entry("Hour", "%(time.hour)", Equal("20")),
		Entry("Minute", "%(time.minute)", Equal("30")),
		Entry("Month", "%(time.month)", Equal("February")),
		Entry("Nanosecond", "%(time.nanosecond)", Equal("300")),
		Entry("Second", "%(time.second)", Equal("33")),
		Entry("Unix", "%(time.unix)", Equal("1769977833")),
		Entry("Timestamp", "%(time.timestamp)", Equal("1769977833")),
		Entry("UnixNano", "%(time.unixNano)", Equal("1769977833000000300")),
		Entry("TimestampNano", "%(time.timestampNano)", Equal("1769977833000000300")),
		Entry("Weekday", "%(time.weekday)", Equal("Sunday")),
		Entry("Year", "%(time.year)", Equal("2026")),
		Entry("YearDay", "%(time.yearDay)", Equal("32")),
		Entry("Zone", "%(time.zone)", Equal("UTC")),
		Entry("ZoneOffset", "%(time.zoneOffset)", Equal("0")),
	)
})
