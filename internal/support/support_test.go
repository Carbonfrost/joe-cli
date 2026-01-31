// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package support_test

import (
	"github.com/Carbonfrost/joe-cli/internal/support"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ParseMap", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		Expect(support.FlattenValues(support.ParseMap(text))).To(expected)
	},
		Entry("nominal", "K1=V1,K2=V2", Equal(map[string]string{"K1": "V1", "K2": "V2"})),
		Entry("empty", nil, BeEmpty()),
		Entry("escaped comma", "L=A\\,B", HaveKeyWithValue("L", "A,B")),
		Entry("escaped comma multiple", "L=A\\,B,M=Y\\,Z,N=W\\,X",
			And(
				HaveKeyWithValue("L", "A,B"),
				HaveKeyWithValue("M", "Y,Z"),
				HaveKeyWithValue("N", "W,X"),
			)),
		Entry("escaped comma trailing", "L=A\\,", HaveKeyWithValue("L", "A,")),
		// Comma is implied as escaped because there is a single KVP
		Entry("implied escaped comma", "L=A,B,C", HaveKeyWithValue("L", "A,B,C")),

		// No commas implies just a value with key=""
		Entry("no commas", "NoCommas", HaveKeyWithValue("", "NoCommas")),

		// No key is interpretted as a value with key=""
		Entry("no key and follower", "NoKey,L=A", Equal(
			map[string]string{
				"":  "NoKey",
				"L": "A",
			}),
		),

		Entry("no key and follower implied escaped comma", "NoKey,L=A,B,C", Equal(
			map[string]string{
				"":  "NoKey",
				"L": "A",
				"B": "",
				"C": "",
			}),
		),

		Entry("implied escaped comma and follower", "L=A,B,C=D", Equal(
			map[string]string{
				"L": "A,B",
				"C": "D",
			}),
		),

		Entry("escaped equal", "L\\=A=B", HaveKeyWithValue("L=A", "B")),
		Entry("escaped equal trailing", "L\\==B", HaveKeyWithValue("L=", "B")),
	)
})

var _ = Describe("SplitList", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		Expect(support.SplitList(text, ",", -1)).To(expected)
	},
		Entry("empty string", "", BeEmpty()),
		Entry("only whitespace string", "  ", BeEmpty()),
		Entry("nominal", "P,Q,R", ConsistOf("P", "Q", "R")),
		Entry("escaping and unquoting", `a="1,2",b='3\=4'`, ConsistOf(`a="1,2"`, `b='3\=4'`)),
		Entry("escaped comma", "L=A\\,B", ConsistOf("L=A\\,B")),
		Entry("escaped comma multiple", "L=A\\,B,M=Y\\,Z,N=W\\,X", ConsistOf("L=A\\,B", "M=Y\\,Z", "N=W\\,X")),
		Entry("comma inside quotes", `1,"L,B"`, Equal([]string{"1", `"L,B"`})),
	)
})
