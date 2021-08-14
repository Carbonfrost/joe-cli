package cli_test

import (
	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("usage", func() {

	Describe("parse", func() {

		DescribeTable("extract placeholders",
			func(text string, expected []string) {
				Expect(cli.ParseUsage(text).Placeholders()).To(Equal(expected))
			},
			Entry("literal", "Literal text", []string{}),
			Entry("placeholder", "{PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("placeholder used twice", "{PLACEHOLDER} {PLACEHOLDER}", []string{"PLACEHOLDER"}),
			Entry("2 placeholders", "{A} {B}", []string{"A", "B"}),
			Entry("2 placeholders with indexes", "{1:A} {0:B}", []string{"B", "A"}),
		)
	})
})
