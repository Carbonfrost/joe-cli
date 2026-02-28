// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expander_test

import (
	"bytes"
	"os"

	"github.com/Carbonfrost/joe-cli/extensions/expr/expander"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("CompilePattern", func() {

	DescribeTable("expected output",
		func(start, end, pattern, expected string) {
			pat := expander.CompilePattern(pattern, start, end)
			actual := pat.Expand(expander.Map(map[string]any{"hello": "world"}))
			Expect(actual).To(Equal(expected))
		},
		Entry("quote with percent sign", "%(", ")", "hello %(hello)", "hello world"),
	)

	Describe("SyntaxRecursive", func() {

		DescribeTable("examples", func(pattern, expected string) {
			pat := expander.SyntaxRecursive.Compile(pattern)

			actual := pat.Expand(
				expander.Map(map[string]any{
					"hello":   "world",
					"goodbye": "earth",
					"foo":     "bar",
				}),
			)
			Expect(actual).To(Equal(expected), "Expected: debug pattern %s", expander.DebugPattern(pat))
		},
			Entry("nominal", "hello %(hello)", "hello world"),
			Entry("fallback to literal", "bonjour %(missing:le monde)", "bonjour le monde"),
			Entry("fallback to var", "hello %(missing:%(goodbye))", "hello earth"),
			Entry("fallback var 2", "hello %(missing:%(missing:%(foo)))", "hello bar"),
			Entry("fallback literal 2", "hello %(missing:%(missing:baz))", "hello baz"),
			Entry("redundant literal fallback", "hello %(missing:literal:bar)", "hello literal"),
		)
	})
})

var _ = Describe("Compile", func() {

	DescribeTable("expected output",
		func(pattern, expected string) {
			pat := expander.Compile(pattern)
			actual := pat.Expand(expander.Map(map[string]any{"hello": "world"}))
			Expect(actual).To(Equal(expected))
		},
		Entry("nominal", "hello %(hello)", "hello world"),
		Entry("missing value", "hello %(planet)", "hello <nil>"),
		Entry("whitespace: empty expansion newline", "%(empty)%(newline)", "\n"),
		Entry("whitespace: nominal expansion newline", "%(hello)%(newline)", "world\n"),
		Entry("whitespace: nominal multiple newlines", "%(hello)%(newline)%(newline)", "world\n\n"),
		Entry("whitespace: literal expansion newline", "literal%(newline)", "literal\n"),
		Entry("whitespace: literal multiple newlines", "literal%(newline)%(newline)", "literal\n\n"),

		// Starting with these whitespace tokens treats as if a literal
		Entry("whitespace: empty literal newline", "%(newline)", "\n"),
		Entry("whitespace: adjacent newlines", "%(newline)%(newline)", "\n\n"),
	)

	Context("when using colors", func() {
		DescribeTable("example",
			func(pattern, expected string) {
				pat := expander.Compile(pattern)
				actual := pat.Expand(expander.Prefix("color", expander.Colors()))
				Expect(actual).To(Equal(expected))
			},
			Entry("yellow", "%(color.yellow)", "\x1b[33m"),
		)
	})

	Context("when using renderer", func() {
		DescribeTable("examples",
			func(pattern, expectedOut, expectedErr string) {
				var out, err bytes.Buffer

				pat := expander.Compile(pattern)
				renderer := expander.NewRenderer(&out, &err)
				_, _ = expander.Fprint(renderer, pat, expander.Map(map[string]any{"hello": "world"}))
				Expect(out.String()).To(Equal(expectedOut))
				Expect(err.String()).To(Equal(expectedErr))
			},
			Entry("default to out", "abc", "abc", ""),
			Entry("switch to err from start", "%(stderr)abc", "", "abc"),
			Entry("switch to err", "abc%(stderr)xyz", "abc", "xyz"),
			Entry("switch back to out", "abc%(stderr)xyz%(stdout)bar", "abcbar", "xyz"),
		)
	})
})

var _ = Describe("String", func() {

	DescribeTable("examples",
		func(pattern, expected string) {
			pat := expander.Compile(pattern)
			Expect(pat.String()).To(Equal(expected))
		},
		Entry("literal", "hello", "hello"),
		Entry("expansion", "hello %(planet)", "hello %(planet)"),
		Entry("untruncated expansion", "hello %(p", "hello %(p"),

		Entry("whitespace", "%(newline)%(tab)%(space)", "%(newline)%(tab)%(space)"),
	)

	It("expands meta reference", func() {
		meta := expander.Compile("%(a) %(b)")
		pat := expander.Compile("%(meta) %(c)").WithMeta("meta", meta)
		Expect(pat.String()).To(Equal("%(a) %(b) %(c)"))
	})
})

var _ = Describe("Env", func() {

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
