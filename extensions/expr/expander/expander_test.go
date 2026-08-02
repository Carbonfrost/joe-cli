// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expander_test

import (
	"fmt"
	"os"
	"runtime"
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

var _ = Describe("Reflect", func() {

	DescribeTable("examples", func(value any, text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		expander := expander.Reflect(value)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("nominal", struct {
			S string
			T string
		}{"1", "2"}, "%(S) %(T)", Equal("1 2")),
		Entry("case insensitive", struct{ R string }{"1"}, "%(r)", Equal("1")),
		Entry("pointer", &struct{ M complex128 }{4 + 80i}, "%(M)", Equal("(4+80i)")),
		Entry("nil", nil, "%(unknown)", Equal("<nil>")),
		Entry("non-existing", struct{ A string }{"L"}, "%(unknown)", Equal("<nil>")),
		Entry("unexported are not expanded", struct{ a string }{"L"}, "%(a)", Equal("<nil>")),

		Entry("method", greeter{}, "%(greeting)", Equal("hello")),
		Entry("method, case insensitive", greeter{}, "%(Greeting)", Equal("hello")),
		Entry("method on pointer receiver", &greeter{}, "%(shout)", Equal("HELLO")),
		Entry("method on pointer receiver is not addressable",
			greeter{}, "%(shout)", Equal("<nil>")),
		Entry("method with arguments is not expanded", greeter{}, "%(repeat)", Equal("<nil>")),
		Entry("method with multiple results is not expanded", greeter{}, "%(result)", Equal("<nil>")),
		Entry("field takes precedence over method", named{Name: "field"}, "%(name)", Equal("field")),
		Entry("time", time.Date(2026, 2, 1, 20, 30, 33, 300, time.UTC), "%(year)-%(month)",
			Equal("2026-February")),
		Entry("nil pointer", (*greeter)(nil), "%(greeting)", Equal("<nil>")),
		Entry("non-struct", "text", "%(unknown)", Equal("<nil>")),
	)
})

type greeter struct{}

func (greeter) Greeting() string { return "hello" }

func (*greeter) Shout() string { return "HELLO" }

func (greeter) Repeat(n int) string { return fmt.Sprint(n) }

func (greeter) Result() (string, error) { return "", nil }

type named struct {
	Name string
}

func (named) NAME() string { return "method" }

var _ = Describe("Runtime", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		expander := expander.Prefix("go", expander.Runtime())
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("numCPU", "%(go.numCPU)", Equal(fmt.Sprintf("%d", runtime.NumCPU()))),
		Entry("os", "%(go.os)", Equal(runtime.GOOS)),
		Entry("arch", "%(go.arch)", Equal(runtime.GOARCH)),
		Entry("version", "%(go.version)", Equal(runtime.Version())),
		Entry("unknown key", "%(go.unknown)", Equal("<nil>")),
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

		Entry("ISOWeek", "%(time.isoWeek)", Equal("2026W5")),
		Entry("ISOWeek", "%(time.isoWeek.week)", Equal("5")),
		Entry("ISOWeek year", "%(time.isoWeek.year)", Equal("2026")),

		Entry("Clock", "%(time.clock)", Equal("20:30:33")),
		Entry("Clock hour", "%(time.clock.hour)", Equal("8")),
		Entry("Clock minute", "%(time.clock.minute)", Equal("30")),
		Entry("Clock second", "%(time.clock.second)", Equal("33")),

		Entry("Date", "%(time.date)", Equal("2026-02-01")),
		Entry("Date year", "%(time.date.year)", Equal("2026")),
		Entry("Date month", "%(time.date.month)", Equal("February")),
		Entry("Date day", "%(time.date.day)", Equal("1")),
	)

	DescribeTable("zone offset examples", func(text string, expected types.GomegaMatcher) {
		e := expander.Compile(text)

		where, err := time.LoadLocation("America/Los_Angeles")
		if err != nil {
			panic(err)
		}
		expander := expander.Prefix("time",
			expander.Time(time.Date(2026, 2, 1, 20, 30, 33, 300, where)),
		)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("UTC conversion", "%(time.utc)", Equal("2026-02-02T04:30:33Z")),
		Entry("UTC conversion hour", "%(time.utc.hour)", Equal("4")),

		Entry("Zone", "%(time.zone)", Equal("PST")),
		Entry("Zone name", "%(time.zone.name)", Equal("PST")),
		Entry("Zone offset", "%(time.zone.offset)", Equal("-28800")),

		Entry("Location", "%(time.location)", Equal("America/Los_Angeles")),
	)
})

var _ = Describe("ExpandSlice", func() {

	DescribeTable("examples", func(in expander.Interface, key string, expected types.GomegaMatcher) {
		Expect(in.Expand(key)).To(expected)
	},
		Entry("nominal", expander.ExpandSlice([]int{0, 1}), "1", Equal(1)),
		Entry("negative", expander.ExpandSlice([]int{0, 1, 2}), "-1", Equal(2)),
		Entry("out of range", expander.ExpandSlice([]int{0}), "10", BeNil()),
		Entry("non-numeric", expander.ExpandSlice([]int{0}), "x", BeNil()),
	)
	Describe("Expand", func() {
	})
})
