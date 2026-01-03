// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package bind_test

import (
	"context"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"testing/fstest"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Binder", func() {

	Describe("intitializer", func() {
		Describe("implicitly sets the type of the argument", func() {

			var factory = func(int) error {
				return nil
			}

			DescribeTable("examples", func(actual cli.Action, name string) {
				app := &cli.App{
					Flags: []*cli.Flag{
						{Name: "flag"},
					},
					Args: []*cli.Arg{
						{Name: "arg"},
					},
					Uses: actual,
					Action: cli.Pipeline(
						func(c *cli.Context) {
							Expect(c.Value(name)).To(BeAssignableToTypeOf(int(0)))
						},
					),
				}

				args, _ := cli.Split("app --flag 300 500")
				err := app.RunContext(context.Background(), args)
				Expect(err).NotTo(HaveOccurred())
			},
				Entry("arg by index", bind.Call(factory, bind.Int()), "arg"),
				Entry("flag by name", bind.Call(factory, bind.Int("flag")), "flag"),
			)
		})
	})
})

func callFactory[T any](t *T) func(T) error {
	return func(s T) error {
		*t = s
		return nil
	}
}

var _ = Describe("FileBinder", func() {

	It("delegates to bind properties", func() {
		var (
			exists                   bool
			dir, ext, name, basename string
		)

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "f",
					Value: new(cli.File),
					Uses: cli.Pipeline(
						bind.Call(callFactory(&name), bind.File().Name()),
						bind.Call(callFactory(&basename), bind.File().Base()),
						bind.Call(callFactory(&ext), bind.File().Ext()),
						bind.Call(callFactory(&dir), bind.File().Dir()),
						bind.Call(callFactory(&exists), bind.File().Exists()),
					),
				},
			},
			FS: fstest.MapFS{
				"V/filename.txt": {
					Data: []byte("data"),
				},
			},
		}
		args, _ := cli.Split("app -f V/filename.txt")
		_ = app.RunContext(context.Background(), args)

		Expect(name).To(Equal("V/filename.txt"))
		Expect(basename).To(Equal("filename.txt"))
		Expect(ext).To(Equal(".txt"))
		Expect(dir).To(Equal("V"))
		Expect(exists).To(BeTrue())
	})
})

var _ = Describe("ContextValue", func() {

	It("invokes the function with the value", func() {
		type contextKey string
		const key contextKey = "key"
		var actionCalledWith int

		fn := func(i int) cli.Action {
			actionCalledWith = i
			return nil
		}

		ctx := context.WithValue(context.Background(), key, 2)
		app := &cli.App{
			Action: bind.Action(fn, bind.ContextValue[int](key)),
		}
		app.RunContext(ctx, []string{"app"})
		Expect(actionCalledWith).To(Equal(2))
	})

})

var _ = Describe("FromContext", func() {

	It("invokes the function with the value", func() {
		var (
			actionCalledWith int
			called           bool
		)
		fn := func(context.Context) int {
			called = true
			return 2
		}
		action := func(i int) cli.Action {
			actionCalledWith = i
			return nil
		}
		app := &cli.App{
			Action: bind.Action(action, bind.FromContext(fn)),
		}
		app.RunContext(context.Background(), []string{"app"})

		Expect(called).To(BeTrue())
		Expect(actionCalledWith).To(Equal(2))
	})
})

var _ = Describe("Exact", func() {

	Describe("intitializer", func() {

		Describe("implicitly sets the type of the flag", func() {

			var factory = func(int) error {
				return nil
			}

			DescribeTable("examples", func(actual cli.Action, args string, expected any) {
				app := &cli.App{
					Flags: []*cli.Flag{
						{Name: "flag", Uses: actual},
					},
					Action: cli.Pipeline(
						func(c *cli.Context) {
							Expect(c.Value("flag")).To(Equal(expected))
						},
					),
				}

				arguments, _ := cli.Split(args)
				err := app.RunContext(context.Background(), arguments)
				Expect(err).NotTo(HaveOccurred())
			},
				Entry(
					"explicit value",
					bind.Call(factory, bind.Exact[int]()),
					"app --flag 300",
					300,
				),
				Entry(
					"value from flag",
					bind.Call(factory, bind.Exact(300)),
					"app --flag",
					true,
				),
			)
		})
	})

	It("invokes bind func with value from flag", func() {
		var value int
		call := func(r int) error {
			value = r
			return nil
		}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:   "memory",
					Value:  new(int),
					Action: bind.Call(call, bind.Exact[int]()),
				},
			},
		}
		args, _ := cli.Split("app --memory 33")
		_ = app.RunContext(context.Background(), args)
		Expect(value).To(Equal(33))
	})

	It("invokes bind func with static value", func() {
		var value int
		call := func(r int) error {
			value = r
			return nil
		}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "max-memory",
					Value: new(bool),
					Uses:  bind.Call(call, bind.Exact(1024)),
				},
			},
		}
		args, _ := cli.Split("app --max-memory")
		_ = app.RunContext(context.Background(), args)
		Expect(value).To(Equal(1024))
		Expect(app.Flags[0].Value).To(PointTo(BeTrue()))
	})

})

var _ = Describe("Value", func() {

	DescribeTable("examples", func(fn any) {
		var actual any

		rv := reflect.ValueOf(fn)
		Expect(func() {
			results := rv.Call(nil)
			actual = results[0].Interface()
		}).NotTo(Panic())
		Expect(actual).NotTo(BeNil())
	},
		Entry("Int", bind.Value[int]),
		Entry("Bool", bind.Value[bool]),
		Entry("String", bind.Value[string]),
		Entry("List", bind.Value[[]string]),
		Entry("Int", bind.Value[int]),
		Entry("Int8", bind.Value[int8]),
		Entry("Int16", bind.Value[int16]),
		Entry("Int32", bind.Value[int32]),
		Entry("Int64", bind.Value[int64]),
		Entry("Uint", bind.Value[uint]),
		Entry("Uint8", bind.Value[uint8]),
		Entry("Uint16", bind.Value[uint16]),
		Entry("Uint32", bind.Value[uint32]),
		Entry("Uint64", bind.Value[uint64]),
		Entry("Float32", bind.Value[float32]),
		Entry("Float64", bind.Value[float64]),
		Entry("Duration", bind.Value[time.Duration]),
		Entry("File", bind.Value[*cli.File]),
		Entry("FileSet", bind.Value[*cli.FileSet]),
		Entry("Map", bind.Value[map[string]string]),
		Entry("NameValue", bind.Value[*cli.NameValue]),
		Entry("NameValues", bind.Value[[]*cli.NameValue]),
		Entry("URL", bind.Value[*url.URL]),
		Entry("Regexp", bind.Value[*regexp.Regexp]),
		Entry("IP", bind.Value[net.IP]),
		Entry("BigInt", bind.Value[*big.Int]),
		Entry("BigFloat", bind.Value[*big.Float]),
		Entry("Bytes", bind.Value[[]byte]),
		Entry("Interface", bind.Value[any]),
	)
})

func box[T any](t T) *T {
	return &t
}

func unwrap[V any](v V, _ any) V {
	return v
}
