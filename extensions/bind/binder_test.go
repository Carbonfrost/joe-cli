// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bind_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"testing/fstest"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/internal/bindfakes"
	"github.com/Carbonfrost/joe-cli/internal/exprfakes"
	joeclifakes "github.com/Carbonfrost/joe-cli/internal/joe-clifakes"
	"github.com/onsi/gomega/types"

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
				Entry("via ActionBinder", bind.Call(factory, bind.NewActionBinder(nil, bind.Int("flag"))), "flag"),
			)
		})

		Describe("implicitly sets the type of the argument", func() {

			var factory = func(string) error {
				return nil
			}
			var factory2 = func(bool) error {
				return nil
			}

			DescribeTable("examples", func(actual cli.Action, expected OmegaMatcher) {
				app := &cli.App{
					Flags: []*cli.Flag{
						{Name: "flag", Uses: actual},
					},
					Action: cli.Pipeline(
						func(c *cli.Context) {
							Expect(c.Value("flag")).To(expected)
						},
					),
				}

				args, _ := cli.Split("app")
				err := app.RunContext(context.Background(), args)
				Expect(err).NotTo(HaveOccurred())
			},
				Entry("File name", bind.Call(factory, bind.File().Name()), BeAssignableToTypeOf(new(cli.File))),
				Entry("File base", bind.Call(factory, bind.File().Base()), BeAssignableToTypeOf(new(cli.File))),
				Entry("NameValue name", bind.Call(factory, bind.NameValue().Name()), BeAssignableToTypeOf(new(cli.NameValue))),
				Entry("Boolean negated", bind.Call(factory2, bind.Bool().Negated()), BeAssignableToTypeOf(true)),
				Entry("wrapped in ActionBinder", bind.Call(factory, bind.NewActionBinder(nil, bind.File().Base())), BeAssignableToTypeOf(new(cli.File))),

				Entry("File name", bind.Call(factory, bind.File("flag").Name()), BeAssignableToTypeOf(new(cli.File))),
				Entry("File base", bind.Call(factory, bind.File("flag").Base()), BeAssignableToTypeOf(new(cli.File))),
				Entry("NameValue name", bind.Call(factory, bind.NameValue("flag").Name()), BeAssignableToTypeOf(new(cli.NameValue))),
				Entry("Boolean negated", bind.Call(factory2, bind.Bool("flag").Negated()), BeAssignableToTypeOf(true)),
			)
		})
	})
})

var _ = Describe("ActionBinder", func() {

	It("invokes the binder", func() {
		fakeBinder := new(bindfakes.FakeBinder[int])
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: bind.Call(callFactory(new(0)), bind.NewActionBinder(nil, fakeBinder)),
				},
			},
		}

		args, _ := cli.Split("app --flag 2")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeBinder.BindCallCount()).To(Equal(1))
	})

	It("invokes the action", func() {
		fakeBinder := new(bindfakes.FakeBinder[int])
		fakeAction := new(joeclifakes.FakeAction)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: bind.NewActionBinder(fakeAction, fakeBinder),
				},
			},
		}

		args, _ := cli.Split("app --flag 2")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeAction.ExecuteCallCount()).To(Equal(1))
	})

	It("invokes the action as initializer in bind context", func() {
		fakeAction := new(joeclifakes.FakeAction)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: bind.Call(callFactory(new(0)), bind.NewActionBinder(fakeAction, new(bindfakes.FakeBinder[int]))),
				},
			},
		}

		args, _ := cli.Split("app --flag 2")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(fakeAction.ExecuteCallCount()).To(Equal(1))
	})
})

var _ = Describe("Elem", func() {

	It("applies element", func() {
		var actual cli.NameValue
		caller := func(v cli.NameValue) error {
			actual = v
			return nil
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "f",
					Uses: cli.Pipeline(
						bind.Call(caller, bind.Elem(bind.NameValue())),
					),
				},
			},
		}
		args, _ := cli.Split("app -f key=value")
		_ = app.RunContext(context.Background(), args)

		Expect(actual.Name).To(Equal("key"))
		Expect(actual.Value).To(Equal("value"))
	})
})

var _ = Describe("Pointer", func() {
	It("applies pointer", func() {
		var actual int
		caller := func(v *int) error {
			actual = *v
			return nil
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "f",
					Uses: cli.Pipeline(
						bind.Call(caller, bind.Pointer(bind.Int())),
					),
				},
			},
		}
		args, _ := cli.Split("app -f 9161")
		_ = app.RunContext(context.Background(), args)

		Expect(actual).To(Equal(9161))
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
			dataReader               io.Reader
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
						bind.Call(callFactory(&dataReader), bind.File().OpenReader()),
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

		data, _ := io.ReadAll(dataReader)
		Expect(string(data)).To(Equal("data"))
	})

	Describe("delegates via Seq applies type and value", func() {
		DescribeTable("arg examples", func(file *bind.FileBinder) {

			var name string
			app := &cli.App{
				Args: []*cli.Arg{
					{Name: "a"},
				},
				Uses: bind.Call(callFactory(&name), file.Name()),
			}

			args, _ := cli.Split("app V/filename.txt")
			_ = app.RunContext(context.Background(), args)

			Expect(name).To(Equal("V/filename.txt"))
			Expect(app.Args[0].Value).To(BeAssignableToTypeOf(new(cli.File)))
		},
			Entry("index", bind.File(0)),
			Entry("name", bind.File("a")),
			Entry("implicit", bind.File()),
		)

		DescribeTable("arg in expr examples", func(file *bind.FileBinder) {

			var name string
			var eval = new(exprfakes.FakeEvaluator)

			callFactory := func(s string) expr.Evaluator {
				name = s
				return eval
			}

			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name: "expression",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "name",
									Args: []*cli.Arg{
										{
											Name: "a",
										},
									},
									Uses: expr.BindEvaluator(callFactory, file.Name()),
								},
							},
						},
					},
				},
				Action: func(c *cli.Context) {
					expr.FromContext(c, "expression").Evaluate(c, 0)
				},
			}

			args, _ := cli.Split("app -- -name V/filename.txt")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("V/filename.txt"))
			Expect(app.Args[0].Value.(*expr.Expression).Exprs[0].Args[0].Value).To(BeAssignableToTypeOf(new(cli.File)))
		},
			Entry("index", bind.File(0)),
			Entry("name", bind.File("a")),
			Entry("implicit", bind.File()),
		)

	})
})

var _ = Describe("BoolBinder", func() {

	It("delegates to bind properties", func() {
		var negated bool

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "f",
					Value: new(bool),
					Uses: cli.Pipeline(
						bind.Call(callFactory(&negated), bind.Bool().Negated()),
					),
				},
			},
		}

		args, _ := cli.Split("app -f")
		_ = app.RunContext(context.Background(), args)
		Expect(negated).To(BeFalse())
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

var _ = Describe("FS", func() {

	It("retrieves the FS from context", func() {
		var actionCalledWith fs.FS
		fn := func(f cli.FS) cli.Action {
			actionCalledWith = f
			return nil
		}

		fake := new(joeclifakes.FakeFS)
		app := &cli.App{
			FS:     fake,
			Action: bind.Action(fn, bind.FS()),
		}
		app.RunContext(context.Background(), []string{"app"})
		Expect(actionCalledWith).To(Equal(fake))
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
					"value from flag",
					bind.Call(factory, bind.Exact[int]()),
					"app --flag 300",
					300,
				),
				Entry(
					"explicit value",
					bind.Call(factory, bind.Exact(300)),
					"app --flag",
					true,
				),
				Entry(
					"wrapped in ActionBinder",
					bind.Call(factory, bind.NewActionBinder(nil, bind.Exact[int]())),
					"app --flag 300",
					300,
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

	Describe("composite binder types", func() {
		DescribeTable("examples", func(binder any, expected types.GomegaMatcher) {
			Expect(binder).To(expected)
		},
			Entry("File", bind.Exact(new(cli.File)), BeAssignableToTypeOf(new(bind.FileBinder))),
			Entry("Boolean", bind.Exact(true), BeAssignableToTypeOf(new(bind.BoolBinder))),
			Entry("NameValue", bind.Exact(new(cli.NameValue)), BeAssignableToTypeOf(new(bind.NameValueBinder))),
		)
	})

})

var _ = Describe("Occurrences", func() {

	Describe("intitializer", func() {

		It("implicitly sets the type of the flag", func() {
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "v",
						Uses: bind.Call(callFactory(new(0)), bind.Occurrences("", 1)),
					},
				},
			}

			_, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(app.Flags[0].Value).To(Equal(new(false)))
		})
	})

	It("returns internal error if name is undefined", func() {
		var value1 bool
		app := &cli.App{
			Name: "b",
			Flags: []*cli.Flag{
				{
					Name: "m",
					Uses: bind.Call(callFactory(&value1), bind.Seen("z")),
				},
			},
		}
		args, _ := cli.Split("app -m _")
		err := app.RunContext(context.Background(), args)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(`internal error, at "b -m" (action timing): flag or arg named in binding but not defined "z"`))
	})

	It("invokes bind func with value from flag", func() {
		var value1, value2, value3 int
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "m",
					Uses: bind.Call(callFactory(&value1), bind.Occurrences("", -1, 33, -2)),
				},
				{
					Name: "n",
					Uses: bind.Call(callFactory(&value2), bind.Occurrences("n", -1, 39, -2)),
				},
				{
					Name:  "z",
					Value: new(bool),
				},
				{
					Name:  "o",
					Value: new(bool),
					Uses:  bind.Call(callFactory(&value3), bind.Occurrences("z", -1, 40, -2)), // unusual but allowed
				},
			},
		}
		args, _ := cli.Split("app -zmno")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(value1).To(Equal(33))
		Expect(value2).To(Equal(39))
		Expect(value3).To(Equal(40))
	})

	Describe("composite binder types", func() {
		DescribeTable("examples", func(binder any, expected types.GomegaMatcher) {
			Expect(binder).To(expected)
		},
			Entry("File", bind.Occurrences("", new(cli.File)), BeAssignableToTypeOf(new(bind.FileBinder))),
			Entry("Boolean", bind.Occurrences("", true), BeAssignableToTypeOf(new(bind.BoolBinder))),
			Entry("NameValue", bind.Occurrences("", new(cli.NameValue)), BeAssignableToTypeOf(new(bind.NameValueBinder))),
		)
	})

	Context("when using bind func count", func() {

		It("binds int binder", func() {
			var actual int
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "m",
						Uses: bind.Call(callFactory(&actual), bind.Occurrences(nil, 0)),
					},
				},
			}
			args, _ := cli.Split("app -mmm")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(3))
		})

		It("does not bind int with additional values", func() {
			var actual int
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "m",
						Uses: bind.Call(callFactory(&actual), bind.Occurrences(nil, 0, 2, 6)),
					},
				},
			}
			args, _ := cli.Split("app -mmm")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(int(6)))
		})

		It("does not bind non-int binder", func() {
			var actual int32
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "m",
						Uses: bind.Call(callFactory(&actual), bind.Occurrences(nil, int32(0))),
					},
				},
			}
			args, _ := cli.Split("app -mmm")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})

		It("does not bind int alias binder", func() {
			type rType int
			var actual rType
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name: "m",
						Uses: bind.Call(callFactory(&actual), bind.Occurrences(nil, rType(0))),
					},
				},
			}
			args, _ := cli.Split("app -mmm")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(rType(0)))
		})

	})

})

var _ = Describe("Seen", func() {

	It("invokes with bind func with value from flag", func() {
		var value1, value2, value3, value4 bool
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "m",
					Uses: bind.Call(callFactory(&value1), bind.Seen("")),
				},
				{
					Name: "n",
					Uses: bind.Call(callFactory(&value2), bind.Seen("n")),
				},
				{
					Name:  "z",
					Value: new(true),
					Uses:  bind.AfterCall(callFactory(&value3), bind.Seen("z")),
				},
				{
					Name: "o",
					Uses: bind.Call(callFactory(&value4), bind.Seen()),
				},
			},
		}
		args, _ := cli.Split("app -mno")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(value1).To(BeTrue())
		Expect(value2).To(BeTrue())
		Expect(value3).To(BeFalse())
		Expect(value4).To(BeTrue())
	})

	It("panics on invalid arguments", func() {
		Expect(func() { bind.Seen(1, 2, 3) }).To(Panic())
	})

})

var _ = Describe("Stdout", func() {

	It("invokes with bind func with output buffer", func() {
		testFunc := func(stdout, stderr io.Writer) error {
			fmt.Fprintln(stdout, "hello out")
			fmt.Fprintln(stderr, "hello err")
			return nil
		}
		var stdout, stderr bytes.Buffer

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "f",
					Uses: bind.Call2(testFunc, bind.Stdout(), bind.Stderr()),
				},
			},
			Stdout: &stdout,
			Stderr: &stderr,
		}

		args, _ := cli.Split("app -f r")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(stdout.String()).To(Equal("hello out\n"))
		Expect(stderr.String()).To(Equal("hello err\n"))
	})

})

var _ = Describe("Seq", func() {

	var src = bind.Func[any](func(*cli.Context) (any, error) {
		return nil, nil
	})

	Describe("composite binder types", func() {
		DescribeTable("examples", func(actual any, expected types.GomegaMatcher) {
			Expect(actual).To(expected)
		},
			Entry("File", bind.Seq(src, func(any) (*cli.File, error) { return nil, nil }), BeAssignableToTypeOf(new(bind.FileBinder))),
			Entry("Boolean", bind.Seq(src, func(any) (bool, error) { return false, nil }), BeAssignableToTypeOf(new(bind.BoolBinder))),
			Entry("NameValue", bind.Seq(src, func(any) (*cli.NameValue, error) { return nil, nil }), BeAssignableToTypeOf(new(bind.NameValueBinder))),
		)
	})
})

var _ = Describe("Value", func() {

	DescribeTable("examples", func(fn any) {
		Expect(fn).To(WithTransform(calledWithReflection, Not(BeNil())))
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

	It("supports all the binder built-ins", func() {
		var fix struct {
			Bool       bool
			String     string
			List       []string
			Int        int
			Int8       int8
			Int16      int16
			Int32      int32
			Int64      int64
			Uint       uint
			Uint8      uint8
			Uint16     uint16
			Uint32     uint32
			Uint64     uint64
			Float32    float32
			Float64    float64
			Duration   time.Duration
			File       *cli.File
			FileSet    *cli.FileSet
			Map        map[string]string
			NameValue  *cli.NameValue
			NameValues []*cli.NameValue
			URL        *url.URL
			Regexp     *regexp.Regexp
			IP         net.IP
			BigInt     *big.Int
			BigFloat   *big.Float
			Bytes      []byte
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "bool", Uses: bind.SetPointer(&fix.Bool, bind.Bool())},
				{Name: "string", Uses: bind.SetPointer(&fix.String, bind.String())},
				{Name: "list", Uses: bind.SetPointer(&fix.List, bind.List())},
				{Name: "int", Uses: bind.SetPointer(&fix.Int, bind.Int())},
				{Name: "int8", Uses: bind.SetPointer(&fix.Int8, bind.Int8())},
				{Name: "int16", Uses: bind.SetPointer(&fix.Int16, bind.Int16())},
				{Name: "int32", Uses: bind.SetPointer(&fix.Int32, bind.Int32())},
				{Name: "int64", Uses: bind.SetPointer(&fix.Int64, bind.Int64())},
				{Name: "uint", Uses: bind.SetPointer(&fix.Uint, bind.Uint())},
				{Name: "uint8", Uses: bind.SetPointer(&fix.Uint8, bind.Uint8())},
				{Name: "uint16", Uses: bind.SetPointer(&fix.Uint16, bind.Uint16())},
				{Name: "uint32", Uses: bind.SetPointer(&fix.Uint32, bind.Uint32())},
				{Name: "uint64", Uses: bind.SetPointer(&fix.Uint64, bind.Uint64())},
				{Name: "float32", Uses: bind.SetPointer(&fix.Float32, bind.Float32())},
				{Name: "float64", Uses: bind.SetPointer(&fix.Float64, bind.Float64())},
				{Name: "duration", Uses: bind.SetPointer(&fix.Duration, bind.Duration())},
				{Name: "file", Uses: bind.SetPointer(&fix.File, bind.File())},
				{Name: "fileset", Uses: bind.SetPointer(&fix.FileSet, bind.FileSet())},
				{Name: "map", Uses: bind.SetPointer(&fix.Map, bind.Map())},
				{Name: "namevalue", Uses: bind.SetPointer(&fix.NameValue, bind.NameValue())},
				{Name: "namevalues", Uses: bind.SetPointer(&fix.NameValues, bind.NameValues())},
				{Name: "url", Uses: bind.SetPointer(&fix.URL, bind.URL())},
				{Name: "regexp", Uses: bind.SetPointer(&fix.Regexp, bind.Regexp())},
				{Name: "ip", Uses: bind.SetPointer(&fix.IP, bind.IP())},
				{Name: "bigint", Uses: bind.SetPointer(&fix.BigInt, bind.BigInt())},
				{Name: "bigfloat", Uses: bind.SetPointer(&fix.BigFloat, bind.BigFloat())},
				{Name: "bytes", Uses: bind.SetPointer(&fix.Bytes, bind.Bytes())},
			},
		}

		arguments, _ := cli.Split(`app --bool=true
										--string=string
										--list=a,b
										--int=300
										--int8=60
										--int16=1000
										--int32=2000
										--int64=3000
										--uint=301
										--uint8=61
										--uint16=1001
										--uint32=2001
										--uint64=3001
										--float32=0.32
										--float64=0.64
										--duration=8s
										--file=filename
										--fileset=./filename
										--map=k=v
										--namevalue=n=v
										--namevalues=n=v,o=w
										--url=https://example.com
										--regexp=^hello$
										--ip=127.0.0.1
										--bigint=808080
										--bigfloat=1.08
										--bytes=deadbeef`)
		err := app.RunContext(context.Background(), arguments)
		Expect(err).NotTo(HaveOccurred())
		Expect(fix.Int).To(Equal(300))
		Expect(fix.Bool).To(Equal(true))
		Expect(fix.String).To(Equal("string"))
		Expect(fix.List).To(Equal([]string{"a", "b"}))
		Expect(fix.Int).To(Equal(300))
		Expect(fix.Int8).To(Equal(int8(60)))
		Expect(fix.Int16).To(Equal(int16(1000)))
		Expect(fix.Int32).To(Equal(int32(2000)))
		Expect(fix.Int64).To(Equal(int64(3000)))
		Expect(fix.Uint).To(Equal(uint(301)))
		Expect(fix.Uint8).To(Equal(uint8(61)))
		Expect(fix.Uint16).To(Equal(uint16(1001)))
		Expect(fix.Uint32).To(Equal(uint32(2001)))
		Expect(fix.Uint64).To(Equal(uint64(3001)))
		Expect(fix.Float32).To(Equal(float32(0.32)))
		Expect(fix.Float64).To(Equal(float64(0.64)))
		Expect(fix.Duration).To(Equal(8 * time.Second))
		Expect(fix.File.Name).To(Equal("filename"))
		Expect(fix.FileSet.Files).To(Equal([]string{"./filename"}))
		Expect(fix.Map).To(HaveKeyWithValue("k", "v"))
		Expect(fix.NameValue).To(Equal(&cli.NameValue{Name: "n", Value: "v"}))
		Expect(fix.NameValues).To(Equal([]*cli.NameValue{{Name: "n", Value: "v"}, {Name: "o", Value: "w"}}))
		Expect(fix.URL).To(Equal(must(url.Parse("https://example.com"))))
		Expect(fix.Regexp).To(Equal(regexp.MustCompile("^hello$")))
		Expect(fix.IP).To(Equal(net.ParseIP("127.0.0.1")))
		Expect(fix.BigInt).To(Equal(big.NewInt(808080)))
		Expect(fix.BigFloat.String()).To(Equal("1.08"))
		Expect(fix.Bytes).To(Equal([]byte{0xde, 0xad, 0xbe, 0xef}))

	})

	Describe("composite binder types", func() {
		DescribeTable("examples", func(fn any, expected types.GomegaMatcher) {
			Expect(fn).To(WithTransform(calledWithReflection, expected))
		},
			Entry("File", bind.Value[*cli.File], BeAssignableToTypeOf(new(bind.FileBinder))),
			Entry("Boolean", bind.Value[bool], BeAssignableToTypeOf(new(bind.BoolBinder))),
			Entry("NameValue", bind.Value[*cli.NameValue], BeAssignableToTypeOf(new(bind.NameValueBinder))),
		)
	})

	It("unwraps pointer to Value", func() {
		var actual element

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: bind.Call(callFactory(&actual), bind.Value[element]()),
				},
			},
		}

		arguments, _ := cli.Split("app --flag 300")
		err := app.RunContext(context.Background(), arguments)
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(element(300)))
	})
})

func calledWithReflection(fn any) any {
	var actual any

	rv := reflect.ValueOf(fn)
	Expect(func() {
		results := rv.Call(nil)
		actual = results[0].Interface()
	}).NotTo(Panic())

	return actual
}

// element is not Value, but *element is
type element int

var _ cli.Value = (*element)(nil)

func (e *element) Set(s string) error {
	var value int
	err := cli.Set(&value, s)
	*e = element(value)
	return err
}

func (*element) String() string { return "" }
