package color_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/color"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mode", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(arg string, expected int) {
				actual := new(color.Mode)
				err := actual.Set(arg)

				Expect(err).NotTo(HaveOccurred())
				Expect(*actual).To(Equal(color.Mode(expected)))
			},
			Entry("nominal", "auto", color.Auto),
			Entry("bool true", "true", color.Always),
			Entry("bool on", "on", color.Always),
			Entry("always", "always", color.Always),
		)
	})

})
