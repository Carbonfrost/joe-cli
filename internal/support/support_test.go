package support_test

import (
	"github.com/Carbonfrost/joe-cli/internal/support"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ParseMap", func() {

	DescribeTable("examples", func(text []string, expected types.GomegaMatcher) {
		Expect(support.ParseMap(text)).To(expected)
	},
		Entry("escaped comma", []string{"L=A\\,B"}, HaveKeyWithValue("L", "A,B")),
		Entry("escaped comma multiple", []string{"L=A\\,B", "M=Y\\,Z", "N=W\\,X"},
			And(
				HaveKeyWithValue("L", "A,B"),
				HaveKeyWithValue("M", "Y,Z"),
				HaveKeyWithValue("N", "W,X"),
			)),
		Entry("escaped comma trailing", []string{"L=A\\,"}, HaveKeyWithValue("L", "A,")),
		// Comma is implied as escaped because there is no other KVP after it
		Entry("implied escaped comma", []string{"L=A,B,C"}, HaveKeyWithValue("L", "A,B,C")),
		Entry("escaped equal", []string{"L\\=A=B"}, HaveKeyWithValue("L=A", "B")),
		Entry("escaped equal trailing", []string{"L\\==B"}, HaveKeyWithValue("L=", "B")),
	)
})
