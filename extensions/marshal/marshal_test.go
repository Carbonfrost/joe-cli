// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Type", func() {

	Describe("New", func() {
		DescribeTable("examples",
			func(t marshal.Type, expected any) {
				v := t.New()
				Expect(v).To(BeAssignableToTypeOf(expected))
			},
			Entry("Bool", marshal.Bool, cli.Bool()),
			Entry("File", marshal.File, &cli.File{}),
			Entry("FileSet", marshal.FileSet, &cli.FileSet{}),
			Entry("Float32", marshal.Float32, cli.Float32()),
			Entry("Float64", marshal.Float64, cli.Float64()),
			Entry("Int", marshal.Int, cli.Int()),
			Entry("Int16", marshal.Int16, cli.Int16()),
			Entry("Int32", marshal.Int32, cli.Int32()),
			Entry("Int64", marshal.Int64, cli.Int64()),
			Entry("Int8", marshal.Int8, cli.Int8()),
			Entry("Duration", marshal.Duration, cli.Duration()),
			Entry("List", marshal.List, cli.List()),
			Entry("Map", marshal.Map, cli.Map()),
			Entry("NameValue", marshal.NameValue, &cli.NameValue{}),
			Entry("NameValues", marshal.NameValues, cli.NameValues()),
			Entry("String", marshal.String, cli.String()),
			Entry("Uint", marshal.Uint, cli.Uint()),
			Entry("Uint16", marshal.Uint16, cli.Uint16()),
			Entry("Uint32", marshal.Uint32, cli.Uint32()),
			Entry("Uint64", marshal.Uint64, cli.Uint64()),
			Entry("Uint8", marshal.Uint8, cli.Uint8()),
			Entry("URL", marshal.URL, cli.URL()),
			Entry("Regexp", marshal.Regexp, cli.Regexp()),
			Entry("IP", marshal.IP, cli.IP()),
			Entry("BigFloat", marshal.BigFloat, cli.BigFloat()),
			Entry("BigInt", marshal.BigInt, cli.BigInt()),
		)
	})

	Describe("parsing", func() {
		DescribeTable("examples",
			func(text string, expected marshal.Type) {
				var t marshal.Type
				_ = (&t).UnmarshalText([]byte(text))

				Expect(t).To(Equal(expected))
			},
			Entry("Bool", "bool", marshal.Bool),
			Entry("File", "file", marshal.File),
			Entry("FileSet", "fileset", marshal.FileSet),
			Entry("Float32", "float32", marshal.Float32),
			Entry("Float64", "float64", marshal.Float64),
			Entry("Int", "int", marshal.Int),
			Entry("Int16", "int16", marshal.Int16),
			Entry("Int32", "int32", marshal.Int32),
			Entry("Int64", "int64", marshal.Int64),
			Entry("Int8", "int8", marshal.Int8),
			Entry("Duration", "duration", marshal.Duration),
			Entry("List", "list", marshal.List),
			Entry("Map", "map", marshal.Map),
			Entry("NameValue", "namevalue", marshal.NameValue),
			Entry("NameValues", "namevalues", marshal.NameValues),
			Entry("String", "string", marshal.String),
			Entry("Uint", "uint", marshal.Uint),
			Entry("Uint16", "uint16", marshal.Uint16),
			Entry("Uint32", "uint32", marshal.Uint32),
			Entry("Uint64", "uint64", marshal.Uint64),
			Entry("Uint8", "uint8", marshal.Uint8),
			Entry("URL", "url", marshal.URL),
			Entry("Regexp", "regexp", marshal.Regexp),
			Entry("IP", "ip", marshal.IP),
			Entry("BigFloat", "bigfloat", marshal.BigFloat),
			Entry("BigInt", "bigint", marshal.BigInt),
		)
	})
})
