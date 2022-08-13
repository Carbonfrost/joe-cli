package structure_test

import (
	"math/big"
	"net"
	"net/url"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/structure"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Value", func() {

	It("unwraps value in lookup", func() {
		s := &struct {
			A string
		}{}
		lk := cli.LookupValues{"a": structure.Of(s)}
		Expect(lk.Value("a")).To(Equal(s))
	})

	Describe("String", func() {
		DescribeTable("examples",
			func(value interface{}, expected types.GomegaMatcher) {
				Expect(structure.Of(value).String()).To(expected)
			},
			Entry("map", map[string]string{"mass": "ive", "ly": "m"}, Equal("ly=m,mass=ive")),
			Entry("structure", &struct {
				Hello string
				World string
			}{
				Hello: "X",
				World: "Y",
			}, Equal("Hello=X,World=Y")),
		)
	})

	Describe("Set", func() {

		DescribeTable("examples",
			func(name string, value string, expected types.GomegaMatcher) {
				s := &struct {
					Vstring  string
					Vbool    bool
					Vint     int
					Vint16   int16
					Vint32   int32
					Vint64   int64
					Vint8    int8
					Vuint    uint
					Vuint16  uint16
					Vuint32  uint32
					Vuint64  uint64
					Vuint8   uint8
					Vfloat32 float32
					Vfloat64 float64

					VDuration time.Duration
					VbigFloat *big.Float
					VbigInt   *big.Int

					VIP     net.IP
					VList   []string
					Vmap    map[string]string
					VRegexp *regexp.Regexp

					VURL   *url.URL
					VValue cli.Value
				}{
					VValue: new(joeclifakes.FakeValue),
				}

				err := cli.Set(structure.Of(s), name+"="+value)
				Expect(err).NotTo(HaveOccurred())

				Expect(s).To(PointTo(MatchFields(IgnoreExtras, Fields{
					name: expected,
				})))
			},

			Entry("string", "Vstring", "true", Equal("true")),
			Entry("bool", "Vbool", "true", Equal(true)),
			Entry("int", "Vint", "480", Equal(480)),
			Entry("int16", "Vint16", "421", Equal(int16(421))),
			Entry("int32", "Vint32", "421", Equal(int32(421))),
			Entry("int64", "Vint64", "421", Equal(int64(421))),
			Entry("int8", "Vint8", "47", Equal(int8(47))),
			Entry("uint", "Vuint", "480", Equal(uint(480))),
			Entry("uint16", "Vuint16", "421", Equal(uint16(421))),
			Entry("uint32", "Vuint32", "421", Equal(uint32(421))),
			Entry("uint64", "Vuint64", "421", Equal(uint64(421))),
			Entry("uint8", "Vuint8", "42", Equal(uint8(42))),
			Entry("float32", "Vfloat32", "3.14", Equal(float32(3.14))),
			Entry("float64", "Vfloat64", "3.14", Equal(float64(3.14))),
			Entry("Duration", "VDuration", "500ms", Equal(500*time.Millisecond)),
			Entry("big.Float", "VbigFloat", "-150.2", WithTransform(unwrapBigFloat, Equal(float64(-150.2)))),
			Entry("big.Int", "VbigInt", "22", Equal(big.NewInt(22))),
			Entry("IP", "VIP", "127.0.0.1", Equal(net.ParseIP("127.0.0.1"))),
			Entry("List", "VList", "a,b,c", Equal([]string{"a", "b", "c"})),
			XEntry("map", "Vmap", "hello=world,goodbye=earth", Equal(map[string]string{
				"hello":   "world",
				"goodbye": "earth",
			})),
			XEntry("NameValues", "VNameValues", "hello,world=good", Equal([]*cli.NameValue{
				{Name: "hello", Value: "true"},
				{Name: "world", Value: "good"},
			})),
			Entry("Regexp", "VRegexp", "[CGAT]{512}", Equal(regexp.MustCompile("[CGAT]{512}"))),
			Entry("URL", "VURL", "https://localhost.example:1619", Equal(unwrap(url.Parse("https://localhost.example:1619")))),
			Entry("Value", "VValue", "value was set", WithTransform(fakeValueArg, Equal("value was set"))),
		)
	})
})

func unwrap(v, _ interface{}) interface{} {
	return v
}

func unwrapBigFloat(v interface{}) interface{} {
	f, _ := v.(*big.Float).Float64()
	return f
}

func fakeValueArg(v interface{}) interface{} {
	f := v.(*joeclifakes.FakeValue).SetArgsForCall(0)
	return f
}
