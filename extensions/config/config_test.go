// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"context"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Config", func() {

	It("sets up in the app", func() {
		app := &cli.App{
			Name: "myapp",
			Uses: config.New(),
		}
		ctx, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(func() {
			config.FromContext(ctx)
		}).NotTo(Panic())
	})

	Describe("Store", func() {

		It("returns empty store when Config is nil", func() {
			var c *config.Config
			store := c.Store()
			Expect(store).NotTo(BeNil())
			Expect(store.Has("anything")).To(BeFalse())
			Expect(store.String("anything")).To(Equal(""))
			Expect(store.Int("anything")).To(Equal(0))
		})

		It("returns empty store when store is nil", func() {
			c := &config.Config{}
			store := c.Store()
			Expect(store).NotTo(BeNil())
			Expect(store.Has("anything")).To(BeFalse())
			Expect(store.String("anything")).To(Equal(""))
			Expect(store.Int("anything")).To(Equal(0))
		})

		It("returns actual store when set", func() {
			c := config.New(config.WithStore(alwaysHas{cli.LookupValues{"key": "value"}}))
			store := c.Store()
			Expect(store).NotTo(BeNil())
			Expect(store.Has("key")).To(BeTrue())
			Expect(store.String("key")).To(Equal("value"))
		})

	})

	var _ = Describe("Lookup", func() {

		DescribeTableSubtree("examples",
			func(v any, lookup func(*config.Config) any, expected types.GomegaMatcher) {

				It("delegates to store nominal", func() {
					lk := config.New(config.WithStore(alwaysHas{cli.LookupValues{"a": v}}))
					Expect(lookup(lk)).To(expected)
					Expect(lk.Value("a")).To(expected)
				})

			},
			Entry(
				"bool",
				cli.Bool(),
				func(lk *config.Config) any { return lk.Bool("a") },
				Equal(false),
			),
			Entry(
				"File",
				&cli.File{},
				func(lk *config.Config) any { return lk.File("a") },
				Equal(&cli.File{}),
			),
			Entry(
				"FileSet",
				&cli.FileSet{},
				func(lk *config.Config) any { return lk.FileSet("a") },
				Equal(&cli.FileSet{}),
			),
			Entry(
				"Float32",
				cli.Float32(),
				func(lk *config.Config) any { return lk.Float32("a") },
				Equal(float32(0)),
			),
			Entry(
				"Float64",
				cli.Float64(),
				func(lk *config.Config) any { return lk.Float64("a") },
				Equal(float64(0)),
			),
			Entry(
				"Int",
				cli.Int(),
				func(lk *config.Config) any { return lk.Int("a") },
				Equal(int(0)),
			),
			Entry(
				"Int16",
				cli.Int16(),
				func(lk *config.Config) any { return lk.Int16("a") },
				Equal(int16(0)),
			),
			Entry(
				"Int32",
				cli.Int32(),
				func(lk *config.Config) any { return lk.Int32("a") },
				Equal(int32(0)),
			),
			Entry(
				"Int64",
				cli.Int64(),
				func(lk *config.Config) any { return lk.Int64("a") },
				Equal(int64(0)),
			),
			Entry(
				"Int8",
				cli.Int8(),
				func(lk *config.Config) any { return lk.Int8("a") },
				Equal(int8(0)),
			),
			Entry(
				"Duration",
				cli.Duration(),
				func(lk *config.Config) any { return lk.Duration("a") },
				Equal(time.Duration(0)),
			),
			Entry(
				"List",
				cli.List(),
				func(lk *config.Config) any { return lk.List("a") },
				BeAssignableToTypeOf([]string{}),
			),
			Entry(
				"Map",
				cli.Map(),
				func(lk *config.Config) any { return lk.Map("a") },
				BeAssignableToTypeOf(map[string]string{}),
			),
			Entry(
				"NameValue",
				&cli.NameValue{},
				func(lk *config.Config) any { return lk.NameValue("a") },
				BeAssignableToTypeOf(&cli.NameValue{}),
			),
			Entry(
				"NameValues",
				cli.NameValues(),
				func(lk *config.Config) any { return lk.NameValues("a") },
				BeAssignableToTypeOf(make([]*cli.NameValue, 0)),
			),
			Entry(
				"String",
				cli.String(),
				func(lk *config.Config) any { return lk.String("a") },
				Equal(""),
			),
			Entry(
				"Uint",
				cli.Uint(),
				func(lk *config.Config) any { return lk.Uint("a") },
				Equal(uint(0)),
			),
			Entry(
				"Uint16",
				cli.Uint16(),
				func(lk *config.Config) any { return lk.Uint16("a") },
				Equal(uint16(0)),
			),
			Entry(
				"Uint32",
				cli.Uint32(),
				func(lk *config.Config) any { return lk.Uint32("a") },
				Equal(uint32(0)),
			),
			Entry(
				"Uint64",
				cli.Uint64(),
				func(lk *config.Config) any { return lk.Uint64("a") },
				Equal(uint64(0)),
			),
			Entry(
				"Uint8",
				cli.Uint8(),
				func(lk *config.Config) any { return lk.Uint8("a") },
				Equal(uint8(0)),
			),
			Entry(
				"URL",
				cli.URL(),
				func(lk *config.Config) any { return lk.URL("a") },
				BeAssignableToTypeOf(&url.URL{}),
			),
			Entry(
				"Regexp",
				cli.Regexp(),
				func(lk *config.Config) any { return lk.Regexp("a") },
				BeAssignableToTypeOf(&regexp.Regexp{}),
			),
			Entry(
				"IP",
				cli.IP(),
				func(lk *config.Config) any { return lk.IP("a") },
				BeAssignableToTypeOf(net.IP{}),
			),
			Entry(
				"BigFloat",
				cli.BigFloat(),
				func(lk *config.Config) any { return lk.BigFloat("a") },
				BeAssignableToTypeOf(&big.Float{}),
			),
			Entry(
				"BigInt",
				cli.BigInt(),
				func(lk *config.Config) any { return lk.BigInt("a") },
				BeAssignableToTypeOf(&big.Int{}),
			),
		)

	})

})

type alwaysHas struct {
	cli.Lookup
}

func (alwaysHas) Has(any) bool {
	return true
}
