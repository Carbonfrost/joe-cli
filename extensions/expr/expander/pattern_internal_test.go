// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expander

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("findAllSubmatchIndex", func() {

	var patternRegexp = regexp.MustCompile(`%\((.+?)\)`)

	DescribeTable("examples", func(text string, expected types.GomegaMatcher, tokens []string) {
		result := slices.Collect(
			findAllSubmatchIndex([]byte(text), []byte("%("), byte(')')),
		)

		// Regression between regexp's behavior only applies to non-nested tokens
		if !strings.Contains(text, "nested") {
			regression := patternRegexp.FindAllSubmatchIndex([]byte(text), -1)
			Expect(formatIndexes(result)).To(Equal(formatIndexes2(regression)))
		}

		actualTokens := make([]string, len(result))
		for i, loc := range result {
			key := text[loc[2]:loc[3]]
			actualTokens[i] = key
		}
		Expect(actualTokens).To(Equal(tokens))
		Expect(result).To(expected)
	},
		Entry("empty string", "", HaveLen(0), []string{}),
		Entry("no matches", ".........", HaveLen(0), []string{}),
		Entry("nominal", "%(token)", Equal([][4]int{
			{0, 8, 2, 7},
		}), []string{"token"}),
		Entry("prefix portion skipped", "%%%(token)", Equal([][4]int{
			{2, 10, 4, 9},
		}), []string{"token"}),

		Entry("zero-len token", "%()", HaveLen(0), []string{}),
		Entry("skips", "%(a)  %(b)  %(c)%(d) ", Equal([][4]int{
			{0, 4, 2, 3},
			{6, 10, 8, 9},
			{12, 16, 14, 15},
			{16, 20, 18, 19},
		}), []string{"a", "b", "c", "d"}),

		Entry("nested token", "%(token:%(nested_token))", HaveLen(1), []string{"token:%(nested_token)"}),
		Entry("nested illegal token", "%(%(illegal_nested)", Equal([][4]int{
			{2, 19, 4, 18},
		}), []string{"illegal_nested"}),
		Entry("unterminated token", "%(token", HaveLen(0), []string{}),
		Entry("unterminated nested token", "%(%(token", HaveLen(0), []string{}),
	)

})

func DebugPattern(p *Pattern) string {
	return p.debugExprs()
}

func formatIndexes(v [][4]int) string {
	var buf strings.Builder
	for i, val := range v {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprint(val))
	}
	return buf.String()
}

func formatIndexes2(v [][]int) string {
	var buf strings.Builder
	for i, val := range v {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprint(val))
	}
	return buf.String()
}
