// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	_ "github.com/Carbonfrost/joe-cli/extensions/marshal/codec/toml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Codec", func() {

	Describe("New", func() {

		Describe("option support", func() {

			DescribeTable("examples",
				func(co marshal.Codec, option codec.Option, expected types.GomegaMatcher) {
					_, err := co.New(option)
					Expect(err).To(expected)
				},
				Entry(
					"JSON DisallowUnknownFields",
					marshal.JSON,
					marshal.DisallowUnknownFields(),
					Not(HaveOccurred()),
				),
				Entry(
					"TOML DisallowUnknownFields",
					marshal.TOML,
					marshal.DisallowUnknownFields(),
					Not(HaveOccurred()),
				),
			)
		})

	})

})
