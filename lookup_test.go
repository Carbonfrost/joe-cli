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
			func(v any, lookup func(cli.Lookup) any, expected types.GomegaMatcher) {
				lk := cli.LookupValues{"a": v}
				Expect(lookup(lk)).To(expected)
				Expect(lk.Value("a")).To(expected)
			},
			Entry(
				"bool",
				cli.Bool(),
				func(lk cli.Lookup) any { return lk.Bool("a") },
				Equal(false),
			),
			Entry(
				"File",
				&cli.File{},
				func(lk cli.Lookup) any { return lk.File("a") },
				Equal(&cli.File{}),
			),
			Entry(
				"FileSet",
				&cli.FileSet{},
				func(lk cli.Lookup) any { return lk.FileSet("a") },
				Equal(&cli.FileSet{}),
			),
			Entry(
				"Float32",
				cli.Float32(),
				func(lk cli.Lookup) any { return lk.Float32("a") },
				Equal(float32(0)),
			),
			Entry(
				"Float64",
				cli.Float64(),
				func(lk cli.Lookup) any { return lk.Float64("a") },
				Equal(float64(0)),
			),
			Entry(
				"Int",
				cli.Int(),
				func(lk cli.Lookup) any { return lk.Int("a") },
				Equal(int(0)),
			),
			Entry(
				"Int16",
				cli.Int16(),
				func(lk cli.Lookup) any { return lk.Int16("a") },
				Equal(int16(0)),
			),
			Entry(
				"Int32",
				cli.Int32(),
				func(lk cli.Lookup) any { return lk.Int32("a") },
				Equal(int32(0)),
			),
			Entry(
				"Int64",
				cli.Int64(),
				func(lk cli.Lookup) any { return lk.Int64("a") },
				Equal(int64(0)),
			),
			Entry(
				"Int8",
				cli.Int8(),
				func(lk cli.Lookup) any { return lk.Int8("a") },
				Equal(int8(0)),
			),
			Entry(
				"Duration",
				cli.Duration(),
				func(lk cli.Lookup) any { return lk.Duration("a") },
				Equal(time.Duration(0)),
			),
			Entry(
				"List",
				cli.List(),
				func(lk cli.Lookup) any { return lk.List("a") },
				BeAssignableToTypeOf([]string{}),
			),
			Entry(
				"Map",
				cli.Map(),
				func(lk cli.Lookup) any { return lk.Map("a") },
				BeAssignableToTypeOf(map[string]string{}),
			),
			Entry(
				"NameValue",
				&cli.NameValue{},
				func(lk cli.Lookup) any { return lk.NameValue("a") },
				BeAssignableToTypeOf(&cli.NameValue{}),
			),
			Entry(
				"NameValues",
				cli.NameValues(),
				func(lk cli.Lookup) any { return lk.NameValues("a") },
				BeAssignableToTypeOf(make([]*cli.NameValue, 0)),
			),
			Entry(
				"String",
				cli.String(),
				func(lk cli.Lookup) any { return lk.String("a") },
				Equal(""),
			),
			Entry(
				"UInt",
				cli.UInt(),
				func(lk cli.Lookup) any { return lk.UInt("a") },
				Equal(uint(0)),
			),
			Entry(
				"UInt16",
				cli.UInt16(),
				func(lk cli.Lookup) any { return lk.UInt16("a") },
				Equal(uint16(0)),
			),
			Entry(
				"UInt32",
				cli.UInt32(),
				func(lk cli.Lookup) any { return lk.UInt32("a") },
				Equal(uint32(0)),
			),
			Entry(
				"UInt64",
				cli.UInt64(),
				func(lk cli.Lookup) any { return lk.UInt64("a") },
				Equal(uint64(0)),
			),
			Entry(
				"UInt8",
				cli.UInt8(),
				func(lk cli.Lookup) any { return lk.UInt8("a") },
				Equal(uint8(0)),
			),
			Entry(
				"URL",
				cli.URL(),
				func(lk cli.Lookup) any { return lk.URL("a") },
				BeAssignableToTypeOf(&url.URL{}),
			),
			Entry(
				"Regexp",
				cli.Regexp(),
				func(lk cli.Lookup) any { return lk.Regexp("a") },
				BeAssignableToTypeOf(&regexp.Regexp{}),
			),
			Entry(
				"IP",
				cli.IP(),
				func(lk cli.Lookup) any { return lk.IP("a") },
				BeAssignableToTypeOf(net.IP{}),
			),
			Entry(
				"BigFloat",
				cli.BigFloat(),
				func(lk cli.Lookup) any { return lk.BigFloat("a") },
				BeAssignableToTypeOf(&big.Float{}),
			),
			Entry(
				"BigInt",
				cli.BigInt(),
				func(lk cli.Lookup) any { return lk.BigInt("a") },
				BeAssignableToTypeOf(&big.Int{}),
			),
			Entry(
				"Value auto dereference",
				&hasDereference{v: &big.Int{}},
				func(lk cli.Lookup) any { return lk.Value("a") },
				BeAssignableToTypeOf(&big.Int{}),
			),
		)

		DescribeTable("implicit conversion examples",
			func(v any, lookup func(cli.Lookup) any, expected types.GomegaMatcher) {
				lk := cli.LookupValues{"a": v}
				Expect(lookup(lk)).To(expected)
			},
			Entry(
				"like bool",
				new(likeBool),
				func(lk cli.Lookup) any { return lk.Bool("a") },
				Equal(false),
			),
			Entry(
				"like Float32",
				new(likeFloat32),
				func(lk cli.Lookup) any { return lk.Float32("a") },
				Equal(float32(0)),
			),
			Entry(
				"like Float64",
				new(likeFloat64),
				func(lk cli.Lookup) any { return lk.Float64("a") },
				Equal(float64(0)),
			),
			Entry(
				"like Int",
				new(likeInt),
				func(lk cli.Lookup) any { return lk.Int("a") },
				Equal(int(0)),
			),
			Entry(
				"like Int16",
				new(likeInt16),
				func(lk cli.Lookup) any { return lk.Int16("a") },
				Equal(int16(0)),
			),
			Entry(
				"like Int32",
				new(likeInt32),
				func(lk cli.Lookup) any { return lk.Int32("a") },
				Equal(int32(0)),
			),
			Entry(
				"like Int64",
				new(likeInt64),
				func(lk cli.Lookup) any { return lk.Int64("a") },
				Equal(int64(0)),
			),
			Entry(
				"like Int8",
				new(likeInt8),
				func(lk cli.Lookup) any { return lk.Int8("a") },
				Equal(int8(0)),
			),
			Entry(
				"like UInt",
				new(likeUInt),
				func(lk cli.Lookup) any { return lk.UInt("a") },
				Equal(uint(0)),
			),
			Entry(
				"like UInt16",
				new(likeUInt16),
				func(lk cli.Lookup) any { return lk.UInt16("a") },
				Equal(uint16(0)),
			),
			Entry(
				"like UInt32",
				new(likeUInt32),
				func(lk cli.Lookup) any { return lk.UInt32("a") },
				Equal(uint32(0)),
			),
			Entry(
				"like UInt64",
				new(likeUInt64),
				func(lk cli.Lookup) any { return lk.UInt64("a") },
				Equal(uint64(0)),
			),
			Entry(
				"like UInt8",
				new(likeUInt8),
				func(lk cli.Lookup) any { return lk.UInt8("a") },
				Equal(uint8(0)),
			),
		)
	})

	Describe("conversion", func() {
		DescribeTable("examples",
			func(v any, text string, expected types.GomegaMatcher) {
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

type (
	likeBool    bool
	likeInt     int
	likeInt8    int8
	likeInt16   int16
	likeInt32   int32
	likeInt64   int64
	likeUInt    uint
	likeUInt8   uint8
	likeUInt16  uint16
	likeUInt32  uint32
	likeUInt64  uint64
	likeFloat64 float64
	likeFloat32 float32
)
