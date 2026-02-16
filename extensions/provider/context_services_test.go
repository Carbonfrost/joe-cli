// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/provider"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ContextServices", func() {

	var _ = Describe("LookupRegistry", func() {
		DescribeTable("examples", func(name any) {
			var actual *provider.Registry
			app := &cli.App{
				Uses: &provider.Registry{
					Name: "codecs",
				},
				Action: func(ctx context.Context) {
					actual, _ = provider.Services(ctx).LookupRegistry(name)
				},
			}
			app.RunContext(context.Background(), []string{"app"})
			Expect(actual).NotTo(BeNil())
		},
			Entry("string", "codecs"),
			Entry("Flag", &cli.Flag{Name: "codecs"}),
			Entry(
				"Flag registry",
				&cli.Flag{
					Name:  "c",
					Value: &provider.Value{Registry: "codecs"},
				}),
		)

	})
})
