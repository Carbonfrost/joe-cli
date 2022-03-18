package cli_test

import (
	"math/big"
	"net"
	"net/url"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Lookup", func() {
	Describe("accessors", func() {
		DescribeTable("examples",
			func(v interface{}, lookup func(cli.Lookup) interface{}, expected types.GomegaMatcher) {
				lk := cli.LookupValues{"a": v}
				Expect(lookup(lk)).To(expected)
				Expect(lk.Value("a")).To(expected)
			},
			Entry(
				"bool",
				cli.Bool(),
				func(lk cli.Lookup) interface{} { return lk.Bool("a") },
				Equal(false),
			),

			Entry(
				"File",
				&cli.File{},
				func(lk cli.Lookup) interface{} { return lk.File("a") },
				Equal(&cli.File{}),
			),
			Entry(
				"FileSet",
				&cli.FileSet{},
				func(lk cli.Lookup) interface{} { return lk.FileSet("a") },
				Equal(&cli.FileSet{}),
			),
			Entry(
				"Float32",
				cli.Float32(),
				func(lk cli.Lookup) interface{} { return lk.Float32("a") },
				Equal(float32(0)),
			),
			Entry(
				"Float64",
				cli.Float64(),
				func(lk cli.Lookup) interface{} { return lk.Float64("a") },
				Equal(float64(0)),
			),
			Entry(
				"Int",
				cli.Int(),
				func(lk cli.Lookup) interface{} { return lk.Int("a") },
				Equal(int(0)),
			),
			Entry(
				"Int16",
				cli.Int16(),
				func(lk cli.Lookup) interface{} { return lk.Int16("a") },
				Equal(int16(0)),
			),
			Entry(
				"Int32",
				cli.Int32(),
				func(lk cli.Lookup) interface{} { return lk.Int32("a") },
				Equal(int32(0)),
			),
			Entry(
				"Int64",
				cli.Int64(),
				func(lk cli.Lookup) interface{} { return lk.Int64("a") },
				Equal(int64(0)),
			),
			Entry(
				"Int8",
				cli.Int8(),
				func(lk cli.Lookup) interface{} { return lk.Int8("a") },
				Equal(int8(0)),
			),
			Entry(
				"Duration",
				cli.Duration(),
				func(lk cli.Lookup) interface{} { return lk.Duration("a") },
				Equal(time.Duration(0)),
			),
			Entry(
				"List",
				cli.List(),
				func(lk cli.Lookup) interface{} { return lk.List("a") },
				BeAssignableToTypeOf([]string{}),
			),
			Entry(
				"Map",
				cli.Map(),
				func(lk cli.Lookup) interface{} { return lk.Map("a") },
				BeAssignableToTypeOf(map[string]string{}),
			),
			Entry(
				"NameValue",
				&cli.NameValue{},
				func(lk cli.Lookup) interface{} { return lk.NameValue("a") },
				BeAssignableToTypeOf(&cli.NameValue{}),
			),
			Entry(
				"NameValues",
				cli.NameValues(),
				func(lk cli.Lookup) interface{} { return lk.NameValues("a") },
				BeAssignableToTypeOf(make([]*cli.NameValue, 0)),
			),
			Entry(
				"String",
				cli.String(),
				func(lk cli.Lookup) interface{} { return lk.String("a") },
				Equal(""),
			),
			Entry(
				"UInt",
				cli.UInt(),
				func(lk cli.Lookup) interface{} { return lk.UInt("a") },
				Equal(uint(0)),
			),
			Entry(
				"UInt16",
				cli.UInt16(),
				func(lk cli.Lookup) interface{} { return lk.UInt16("a") },
				Equal(uint16(0)),
			),
			Entry(
				"UInt32",
				cli.UInt32(),
				func(lk cli.Lookup) interface{} { return lk.UInt32("a") },
				Equal(uint32(0)),
			),
			Entry(
				"UInt64",
				cli.UInt64(),
				func(lk cli.Lookup) interface{} { return lk.UInt64("a") },
				Equal(uint64(0)),
			),
			Entry(
				"UInt8",
				cli.UInt8(),
				func(lk cli.Lookup) interface{} { return lk.UInt8("a") },
				Equal(uint8(0)),
			),

			Entry(
				"URL",
				cli.URL(),
				func(lk cli.Lookup) interface{} { return lk.URL("a") },
				BeAssignableToTypeOf(&url.URL{}),
			),
			Entry(
				"Regexp",
				cli.Regexp(),
				func(lk cli.Lookup) interface{} { return lk.Regexp("a") },
				BeAssignableToTypeOf(&regexp.Regexp{}),
			),
			Entry(
				"IP",
				cli.IP(),
				func(lk cli.Lookup) interface{} { return lk.IP("a") },
				BeAssignableToTypeOf(net.IP{}),
			),
			Entry(
				"BigFloat",
				cli.BigFloat(),
				func(lk cli.Lookup) interface{} { return lk.BigFloat("a") },
				BeAssignableToTypeOf(&big.Float{}),
			),
			Entry(
				"BigInt",
				cli.BigInt(),
				func(lk cli.Lookup) interface{} { return lk.BigInt("a") },
				BeAssignableToTypeOf(&big.Int{}),
			),
			Entry(
				"Value auto dereference",
				&hasDereference{v: &big.Int{}},
				func(lk cli.Lookup) interface{} { return lk.Value("a") },
				BeAssignableToTypeOf(&big.Int{}),
			),
		)
	})

	Describe("conversion", func() {
		DescribeTable("examples",
			func(v interface{}, text string, expected types.GomegaMatcher) {
				f := &cli.Flag{Value: v}
				c := cli.InitializeFlag(f)

				f.Set(text)
				Expect(c.Value("")).To(expected)
			},
			Entry(
				"bool",
				cli.Bool(),
				"true",
				Equal(true),
			),

			Entry(
				"Float32",
				cli.Float32(),
				"2.0",
				Equal(float32(2.0)),
			),
			Entry(
				"Float64",
				cli.Float64(),
				"2.0",
				Equal(float64(2.0)),
			),
			Entry(
				"Int",
				cli.Int(),
				"16",
				Equal(int(16)),
			),
			Entry(
				"Int16",
				cli.Int16(),
				"16",
				Equal(int16(16)),
			),
			Entry(
				"Int32",
				cli.Int32(),
				"16",
				Equal(int32(16)),
			),
			Entry(
				"Int64",
				cli.Int64(),
				"16",
				Equal(int64(16)),
			),
			Entry(
				"Int8",
				cli.Int8(),
				"16",
				Equal(int8(16)),
			),
			Entry(
				"List",
				cli.List(),
				"text,plus",
				Equal([]string{"text", "plus"}),
			),
			Entry(
				"Map",
				cli.Map(),
				"key=value",
				Equal(map[string]string{"key": "value"}),
			),
			Entry(
				"String",
				cli.String(),
				"text",
				Equal("text"),
			),
			Entry(
				"UInt",
				cli.UInt(),
				"19",
				Equal(uint(19)),
			),
			Entry(
				"UInt16",
				cli.UInt16(),
				"19",
				Equal(uint16(19)),
			),
			Entry(
				"UInt32",
				cli.UInt32(),
				"19",
				Equal(uint32(19)),
			),
			Entry(
				"UInt64",
				cli.UInt64(),
				"19",
				Equal(uint64(19)),
			),
			Entry(
				"UInt8",
				cli.UInt8(),
				"19",
				Equal(uint8(19)),
			),

			Entry(
				"URL",
				cli.URL(),
				"https://localhost",
				Equal(unwrap(url.Parse("https://localhost"))),
			),
			Entry(
				"Regexp",
				cli.Regexp(),
				"blc",
				Equal(regexp.MustCompile("blc")),
			),
			Entry(
				"IP",
				cli.IP(),
				"127.0.0.1",
				Equal(net.ParseIP("127.0.0.1")),
			),
		)
	})
})
