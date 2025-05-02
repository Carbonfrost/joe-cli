// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package template_test

import (
	"context"

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
		seq.Generate(context.Background(), nil)

		Expect(g1.GenerateCallCount()).To(Equal(1))
		Expect(g2.GenerateCallCount()).To(Equal(1))
	})

	It("skips nil generators", func() {
		g1 := new(templatefakes.FakeGenerator)
		seq := template.Sequence([]template.Generator{
			g1, nil,
		})
		seq.Generate(context.Background(), nil)

		Expect(g1.GenerateCallCount()).To(Equal(1))
	})
})
