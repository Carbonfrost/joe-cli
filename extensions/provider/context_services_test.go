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

	var _ = It("is unique per app", func() {

		var actual1, actual2 [2]bool
		app1 := &cli.App{
			Uses: cli.Pipeline(
				&provider.Registry{Name: "0"},
			),
			// TODO Bug: provider.Services is not ready to use within the Uses pipeline
			Before: func(c context.Context) {
				_, actual1[0] = provider.Services(c).LookupRegistry("0")
				_, actual1[1] = provider.Services(c).LookupRegistry("1")
			},
		}
		app2 := &cli.App{
			Uses: cli.Pipeline(
				&provider.Registry{Name: "1"},
			),
			Before: func(c context.Context) {
				_, actual2[0] = provider.Services(c).LookupRegistry("0")
				_, actual2[1] = provider.Services(c).LookupRegistry("1")
			},
		}

		_ = app1.RunContext(context.Background(), []string{"app"})
		_ = app2.RunContext(context.Background(), []string{"app"})

		Expect(actual1).To(Equal([2]bool{true, false}))
		Expect(actual2).To(Equal([2]bool{false, true}))
	})

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
