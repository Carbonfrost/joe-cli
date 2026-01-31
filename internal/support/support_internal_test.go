// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package support

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("splitMapRegions", func() {

	DescribeTable("examples", func(text string, p, k, s types.GomegaMatcher) {
		prefix, kvps, suffix := splitMapRegions(text)
		Expect(prefix).To(p)
		Expect(kvps).To(k)
		Expect(suffix).To(s)
	},
		Entry("empty", "", BeEmpty(), BeEmpty(), BeEmpty()),
		Entry("nominal", "K1=V1,K2=V2", BeEmpty(), ConsistOf("K1=V1", "K2=V2"), BeEmpty()),
		Entry("just prefix", "P,Q,R", ConsistOf("P", "Q", "R"), BeEmpty(), BeEmpty()),
		Entry("just single prefix", "P", ConsistOf("P"), BeEmpty(), BeEmpty()),
		Entry("prefix and kvps", "P,Q,K=1", ConsistOf("P", "Q"), ConsistOf("K=1"), BeEmpty()),
		Entry("multiple eq", "K=1=2=3", BeEmpty(), ConsistOf("K=1=2=3"), BeEmpty()),
		Entry("prefix, kvps, suffix", "P,Q,K=1,J=2,S,T", ConsistOf("P", "Q"), ConsistOf("K=1", "J=2"), ConsistOf("S", "T")),

		Entry("special case with single kvp", "L=A,B,C", BeEmpty(), ConsistOf("L=A,B,C"), BeEmpty()),
		Entry("kvps and suffix", "K=A,L=B,C", BeEmpty(), ConsistOf("K=A", "L=B"), ConsistOf("C")),

		Entry("escaping and unquoting", `a="1,2",b='3\=4'`, BeEmpty(), ConsistOf(`a="1,2"`, `b='3\=4'`), BeEmpty()),
		Entry("escaped eq", "L\\=A=B", BeEmpty(), ConsistOf("L\\=A=B"), BeEmpty()),
		Entry("escaped eq trailing", "L\\==B", BeEmpty(), ConsistOf("L\\==B"), BeEmpty()),
		Entry("escaped comma", "L=A\\,B", BeEmpty(), ConsistOf("L=A\\,B"), BeEmpty()),
		Entry("escaped comma multiple", "L=A\\,B,M=Y\\,Z,N=W\\,X", BeEmpty(), ConsistOf("L=A\\,B", "M=Y\\,Z", "N=W\\,X"), BeEmpty()),
		Entry("escaped comma trailing", "L=A\\,", BeEmpty(), ConsistOf("L=A\\,"), BeEmpty()),
	)
})
