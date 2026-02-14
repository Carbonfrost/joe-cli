// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/provider"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Describe("Bind", func() {

	It("allows expected value", func() {
		var calledWith formatProvider
		call := func(s formatProvider) error {
			calledWith = s
			return nil
		}

		registry := &provider.Registry{
			Name: "format",
			Providers: provider.Details{
				"csv": {
					Defaults: map[string]string{
						"comma":   "a",
						"useCRLF": "true",
					},
					Factory: provider.FactoryOf(func(opts csvProvider) (any, error) {
						return &opts, nil
					}),
				},
			},
		}

		app := &cli.App{
			Name: "app",
			Uses: registry,
			Flags: []*cli.Flag{
				{
					Name:   "format",
					Value:  &provider.Value{},
					Action: bind.Call(call, provider.Bind[formatProvider]()),
				},
			},
		}

		args, _ := cli.Split("app --format csv,comma=b,useCRLF=false")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(calledWith).To(BeAssignableToTypeOf(new(csvProvider)))
		Expect(calledWith).To(Equal(&csvProvider{"b", false}))
	})
})

var _ = Describe("ValueBinder", func() {

	It("allows expected value", func() {
		var calledWith struct {
			name string
			args any
		}
		call := func(name string, args any) error {
			calledWith.name = name
			calledWith.args = args
			return nil
		}

		registry := &provider.Registry{
			Name:         "format",
			AllowUnknown: true,
		}

		app := &cli.App{
			Name: "app",
			Uses: registry,
			Flags: []*cli.Flag{
				{
					Name:   "format",
					Value:  &provider.Value{},
					Action: bind.Call2(call, provider.BindValue().Name(), provider.BindValue().Args()),
				},
			},
		}

		args, _ := cli.Split("app --format csv,comma=b,useCRLF=false")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(calledWith.name).To(Equal("csv"))
		Expect(calledWith.args).To(gstruct.PointTo(Equal(map[string]string{"comma": "b", "useCRLF": "false"})))
	})
})
