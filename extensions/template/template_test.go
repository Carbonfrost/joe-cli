package template_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/joe-cli/extensions/template/templatefakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sequence", func() {
	It("invokes all dependent generators", func() {
		g1 := new(templatefakes.FakeGenerator)
		g2 := new(templatefakes.FakeGenerator)
		seq := template.Sequence([]template.Generator{
			g1, g2,
		})
		seq.Generate(nil)

		Expect(g1.GenerateCallCount()).To(Equal(1))
		Expect(g2.GenerateCallCount()).To(Equal(1))
	})

	It("skips nil generators", func() {
		g1 := new(templatefakes.FakeGenerator)
		seq := template.Sequence([]template.Generator{
			g1, nil,
		})
		seq.Generate(nil)

		Expect(g1.GenerateCallCount()).To(Equal(1))
	})
})
