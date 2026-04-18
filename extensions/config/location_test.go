// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"github.com/Carbonfrost/joe-cli/extensions/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Layer", func() {

	Describe("String", func() {
		DescribeTable("examples", func(l config.Layer, expected string) {
			Expect(l.String()).To(Equal(expected))
		},
			Entry("LayerUnspecified", config.LayerUnspecified, "UNSPECIFIED"),
			Entry("LayerIntrinsic", config.LayerIntrinsic, "INTRINSIC"),
			Entry("LayerSystem", config.LayerSystem, "SYSTEM"),
			Entry("LayerUser", config.LayerUser, "USER"),
			Entry("LayerWorkspace", config.LayerWorkspace, "WORKSPACE"),
			Entry("LayerProfile", config.LayerProfile, "PROFILE"),
			Entry("LayerAdditional", config.LayerAdditional, "ADDITIONAL"),
			Entry("in between", config.LayerSystem+1, "SYSTEM+1"),
			Entry("in between 2", config.LayerProfile+1, "PROFILE+1"),
			Entry("over bounds", config.Layer(11), "ADDITIONAL"),
			Entry("under bounds", config.Layer(-2), "UNSPECIFIED"),
		)

	})

})
