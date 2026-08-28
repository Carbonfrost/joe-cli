// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bind_test

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type structOptions struct {
	ConfigFile string
	HTTPPort   int
	Verbose    bool
	notAFlag   string
}

type EmbeddedOptions struct {
	Verbose bool
}

type outerOptions struct {
	EmbeddedOptions
	ConfigFile string
}

var _ = Describe("Struct", func() {

	It("binds the exported fields from the flags", func() {
		var actual structOptions

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "config-file", Value: new(string)},
				{Name: "http-port", Value: new(int)},
				{Name: "verbose", Value: new(bool)},
			},
			Uses: bind.Call(callFactory(&actual), bind.Struct[structOptions]()),
		}

		args, _ := cli.Split("app --config-file f.toml --http-port 8080 --verbose")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(actual.ConfigFile).To(Equal("f.toml"))
		Expect(actual.HTTPPort).To(Equal(8080))
		Expect(actual.Verbose).To(BeTrue())
		Expect(actual.notAFlag).To(BeEmpty())
	})

	It("binds the exported fields from the args", func() {
		var actual structOptions

		app := &cli.App{
			Args: []*cli.Arg{
				{Name: "config-file", Value: new(string)},
				{Name: "http-port", Value: new(int)},
			},
			Uses: bind.Call(callFactory(&actual), bind.Struct[structOptions]()),
		}

		args, _ := cli.Split("app f.toml 8080")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(actual.ConfigFile).To(Equal("f.toml"))
		Expect(actual.HTTPPort).To(Equal(8080))
	})

	It("implicitly sets the type of the flags", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "config-file"},
				{Name: "http-port"},
				{Name: "verbose"},
			},
			Uses: bind.Call(callFactory(new(structOptions)), bind.Struct[structOptions]()),
		}

		args, _ := cli.Split("app")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(app.Flags[0].Value).To(BeAssignableToTypeOf(new(string)))
		Expect(app.Flags[1].Value).To(BeAssignableToTypeOf(new(int)))
		Expect(app.Flags[2].Value).To(BeAssignableToTypeOf(new(bool)))
	})

	It("retains the zero value when no flag or arg is defined", func() {
		var actual structOptions

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "config-file", Value: new(string)},
			},
			Uses: bind.Call(callFactory(&actual), bind.Struct[structOptions]()),
		}

		args, _ := cli.Split("app --config-file f.toml")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(actual.ConfigFile).To(Equal("f.toml"))
		Expect(actual.HTTPPort).To(Equal(0))
		Expect(actual.Verbose).To(BeFalse())
	})

	It("binds the promoted fields of an embedded struct", func() {
		var actual outerOptions

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "config-file", Value: new(string)},
				{Name: "verbose", Value: new(bool)},
			},
			Uses: bind.Call(callFactory(&actual), bind.Struct[outerOptions]()),
		}

		args, _ := cli.Split("app --config-file f.toml --verbose")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(actual.ConfigFile).To(Equal("f.toml"))
		Expect(actual.Verbose).To(BeTrue())
	})

	It("converts the value when the flag uses a different numeric type", func() {
		type options struct {
			Count int64
		}
		var actual options

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "count", Value: new(int)},
			},
			Uses: bind.Call(callFactory(&actual), bind.Struct[options]()),
		}

		args, _ := cli.Split("app --count 21")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(actual.Count).To(Equal(int64(21)))
	})

	It("provides an error when the value can't be converted", func() {
		type options struct {
			Count int
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "count", Value: new(string)},
			},
			Uses: bind.Call(callFactory(new(options)), bind.Struct[options]()),
		}

		args, _ := cli.Split("app --count nope")
		err := app.RunContext(context.Background(), args)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`cannot bind "count" to field Count: expected int, got string`))
	})

	It("panics when the type is not a struct", func() {
		Expect(func() {
			bind.Struct[int]()
		}).To(PanicWith("expected struct type for T, got int"))
	})
})
