// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"net"
	"net/url"
	"regexp"

	"github.com/Carbonfrost/joe-cli/extensions/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Values", func() {

	Describe("conversion", func() {

		DescribeTable("examples",
			func(accessor func(config.Values, any) any, text string, expected types.GomegaMatcher) {
				lv := config.Values{"a": text}
				Expect(accessor(lv, "a")).To(expected)
			},
			Entry(
				"bool",
				func(lv config.Values, k any) any { return lv.Bool(k) },
				"true",
				Equal(true),
			),
			Entry(
				"Float32",
				func(lv config.Values, k any) any { return lv.Float32(k) },
				"2.0",
				Equal(float32(2.0)),
			),
			Entry(
				"Float64",
				func(lv config.Values, k any) any { return lv.Float64(k) },
				"2.0",
				Equal(float64(2.0)),
			),
			Entry(
				"Int",
				func(lv config.Values, k any) any { return lv.Int(k) },
				"16",
				Equal(int(16)),
			),
			Entry(
				"Int16",
				func(lv config.Values, k any) any { return lv.Int16(k) },
				"16",
				Equal(int16(16)),
			),
			Entry(
				"Int32",
				func(lv config.Values, k any) any { return lv.Int32(k) },
				"16",
				Equal(int32(16)),
			),
			Entry(
				"Int64",
				func(lv config.Values, k any) any { return lv.Int64(k) },
				"16",
				Equal(int64(16)),
			),
			Entry(
				"Int8",
				func(lv config.Values, k any) any { return lv.Int8(k) },
				"16",
				Equal(int8(16)),
			),
			Entry(
				"List",
				func(lv config.Values, k any) any { return lv.List(k) },
				"text,plus",
				Equal([]string{"text", "plus"}),
			),
			Entry(
				"Map",
				func(lv config.Values, k any) any { return lv.Map(k) },
				"key=value",
				Equal(map[string]string{"key": "value"}),
			),
			Entry(
				"String",
				func(lv config.Values, k any) any { return lv.String(k) },
				"text",
				Equal("text"),
			),
			Entry(
				"Uint",
				func(lv config.Values, k any) any { return lv.Uint(k) },
				"19",
				Equal(uint(19)),
			),
			Entry(
				"Uint16",
				func(lv config.Values, k any) any { return lv.Uint16(k) },
				"19",
				Equal(uint16(19)),
			),
			Entry(
				"Uint32",
				func(lv config.Values, k any) any { return lv.Uint32(k) },
				"19",
				Equal(uint32(19)),
			),
			Entry(
				"Uint64",
				func(lv config.Values, k any) any { return lv.Uint64(k) },
				"19",
				Equal(uint64(19)),
			),
			Entry(
				"Uint8",
				func(lv config.Values, k any) any { return lv.Uint8(k) },
				"19",
				Equal(uint8(19)),
			),
			Entry(
				"URL",
				func(lv config.Values, k any) any { return lv.URL(k) },
				"https://localhost",
				Equal(unwrap(url.Parse("https://localhost"))),
			),
			Entry(
				"Regexp",
				func(lv config.Values, k any) any { return lv.Regexp(k) },
				"blc",
				Equal(regexp.MustCompile("blc")),
			),
			Entry(
				"IP",
				func(lv config.Values, k any) any { return lv.IP(k) },
				"127.0.0.1",
				Equal(net.ParseIP("127.0.0.1")),
			),
		)
	})
})

func unwrap[V any](v V, _ any) V {
	return v
}
