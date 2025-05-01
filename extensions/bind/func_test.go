package bind_test

import (
	"context"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/color"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
	"github.com/Carbonfrost/joe-cli/extensions/expr/exprfakes"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Action", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder bind.Binder[string], expected string) {
			var (
				action     = new(joeclifakes.FakeAction)
				calledWith string
			)
			factory := func(c string) cli.Action {
				calledWith = c
				return action
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg"},
				},
				Uses: bind.Action(factory, binder),
			}

			args, _ := cli.Split("app arg_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(action.ExecuteCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), "flag_value"),
			Entry("implicit name", bind.String(), "arg_value"),
		)
	})

	It("binds implicitly on the current flag", func() {
		var (
			action     = new(joeclifakes.FakeAction)
			calledWith *big.Int
		)
		factory := func(c *big.Int) cli.Action {
			calledWith = c
			return action
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "flag",
					Uses: bind.Action(factory, bind.BigInt()),
				},
			},
		}

		args, _ := cli.Split("app --flag 8000")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(action.ExecuteCallCount()).To(Equal(1))
		Expect(calledWith.Int64()).To(Equal(int64(8000)))
	})

	It("can bind implicit names in Action pipeline", func() {
		var (
			action     = new(joeclifakes.FakeAction)
			calledWith string
		)
		factory := func(c string) cli.Action {
			calledWith = c
			return action
		}

		app := &cli.App{
			Args: []*cli.Arg{
				{Name: "arg"},
			},
			Action: bind.Action(factory, bind.String()),
		}

		args, _ := cli.Split("app arg_value")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(action.ExecuteCallCount()).To(Equal(1))
		Expect(calledWith).To(Equal("arg_value"))
	})
})

var _ = Describe("Action2", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2 bind.Binder[string], expected []string) {
			var (
				action     = new(joeclifakes.FakeAction)
				calledWith []string
			)
			factory := func(a, b string) cli.Action {
				calledWith = []string{a, b}
				return action
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
				},
				Uses: bind.Action2(factory, binder1, binder2),
			}

			args, _ := cli.Split("app arg1_value arg2_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(action.ExecuteCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), []string{"arg1_value", "arg2_value"}),
		)

	})
})

var _ = Describe("Action3", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2, binder3 bind.Binder[string], expected []string) {
			var (
				action     = new(joeclifakes.FakeAction)
				calledWith []string
			)
			factory := func(a, b, c string) cli.Action {
				calledWith = []string{a, b, c}
				return action
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
					{Name: "arg3", NArg: 1},
				},
				Uses: bind.Action3(factory, binder1, binder2, binder3),
			}

			args, _ := cli.Split("app arg1_value arg2_value arg3_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(action.ExecuteCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), bind.String(), []string{"arg1_value", "arg2_value", "arg3_value"}),
		)

	})
})

var _ = Describe("Call", func() {

	It("invokes the factory", func() {
		var (
			called     bool
			calledWith string
		)
		factory := func(c string) error {
			called = true
			calledWith = c
			return nil
		}

		actual := bind.Call(factory, bind.String("flag"))
		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "flag"},
			},
			Action: actual,
		}

		args, _ := cli.Split("app --flag=has_value")
		err := app.RunContext(context.Background(), args)

		Expect(err).NotTo(HaveOccurred())
		Expect(called).To(BeTrue())
		Expect(calledWith).To(Equal("has_value"))
	})
})

var _ = Describe("Call2", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2 bind.Binder[string], expected []string) {
			var (
				calledWith []string
			)
			factory := func(a, b string) error {
				calledWith = []string{a, b}
				return nil
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
				},
				Uses: bind.Call2(factory, binder1, binder2),
			}

			args, _ := cli.Split("app arg1_value arg2_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), []string{"arg1_value", "arg2_value"}),
		)
	})
})

var _ = Describe("Call3", func() {

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder1, binder2, binder3 bind.Binder[string], expected []string) {
			var (
				calledWith []string
			)
			factory := func(a, b, c string) error {
				calledWith = []string{a, b, c}
				return nil
			}

			app := &cli.App{
				Flags: []*cli.Flag{
					{Name: "flag"},
				},
				Args: []*cli.Arg{
					{Name: "arg1", NArg: 1},
					{Name: "arg2", NArg: 1},
					{Name: "arg3", NArg: 1},
				},
				Uses: bind.Call3(factory, binder1, binder2, binder3),
			}

			args, _ := cli.Split("app arg1_value arg2_value arg3_value --flag flag_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("flag"), bind.String("flag"), bind.String("flag"), []string{"flag_value", "flag_value", "flag_value"}),
			Entry("implicit name", bind.String(), bind.String(), bind.String(), []string{"arg1_value", "arg2_value", "arg3_value"}),
		)
	})
})

var _ = Describe("Evaluator", func() {

	It("invokes the factory", func() {
		var called bool
		factory := func(string) expr.Evaluator {
			called = true
			return new(exprfakes.FakeEvaluator)
		}

		actual := bind.Evaluator(factory, bind.String())
		_ = actual.Evaluate(newContext(new(cli.App)), nil, nil)
		Expect(called).To(BeTrue())
	})

	It("delegates Evaluate calls the factory-produced evaluator", func() {
		fakeEvaluator := new(exprfakes.FakeEvaluator)
		factory := func(string) expr.Evaluator {
			return fakeEvaluator
		}

		actual := bind.Evaluator(factory, bind.String())
		_ = actual.Evaluate(newContext(new(cli.App)), nil, nil)
		Expect(fakeEvaluator.EvaluateCallCount()).To(Equal(1))
	})

	It("uses value from the context", func() {
		var varNames struct {
			string string
			bool   bool
			list   []string
			int    int
			int8   int8
			int16  int16
			int32  int32
			int64  int64
			uint   uint
			uint8  uint8
			uint16 uint16
			uint32 uint32
			uint64 uint64
		}
		evaluators := []expr.Evaluator{
			bind.Evaluator(evalFactory(&varNames.string), bind.String("string")),
			bind.Evaluator(evalFactory(&varNames.bool), bind.Bool("bool")),
			bind.Evaluator(evalFactory(&varNames.list), bind.List("list")),
			bind.Evaluator(evalFactory(&varNames.int), bind.Int("int")),
			bind.Evaluator(evalFactory(&varNames.int8), bind.Int8("int8")),
			bind.Evaluator(evalFactory(&varNames.int16), bind.Int16("int16")),
			bind.Evaluator(evalFactory(&varNames.int32), bind.Int32("int32")),
			bind.Evaluator(evalFactory(&varNames.int64), bind.Int64("int64")),
			bind.Evaluator(evalFactory(&varNames.uint), bind.Uint("uint")),
			bind.Evaluator(evalFactory(&varNames.uint8), bind.Uint8("uint8")),
			bind.Evaluator(evalFactory(&varNames.uint16), bind.Uint16("uint16")),
			bind.Evaluator(evalFactory(&varNames.uint32), bind.Uint32("uint32")),
			bind.Evaluator(evalFactory(&varNames.uint64), bind.Uint64("uint64")),
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "string",
					Value: box("value set"),
				},
				{
					Name:  "bool",
					Value: box(true),
				},
				{Name: "list", Value: box([]string{"list"})},
				{Name: "int", Value: box(2)},
				{Name: "int8", Value: box(int8(8))},
				{Name: "int16", Value: box(int16(16))},
				{Name: "int32", Value: box(int32(32))},
				{Name: "int64", Value: box(int64(64))},
				{Name: "uint", Value: box(uint(2))},
				{Name: "uint8", Value: box(uint8(8))},
				{Name: "uint16", Value: box(uint(16))},
				{Name: "uint32", Value: box(uint(32))},
				{Name: "uint64", Value: box(uint(64))},
			},
			Action: func(c context.Context) {
				for _, e := range evaluators {
					e.Evaluate(c, nil, nil)
				}
			},
		}

		app.RunContext(context.Background(), []string{})
		Expect(varNames.string).To(Equal("value set"))
		Expect(varNames.bool).To(Equal(true))
		Expect(varNames.list).To(Equal([]string{"list"}))
		Expect(varNames.int).To(Equal(2))
		Expect(varNames.int8).To(Equal(int8(8)))
		Expect(varNames.int16).To(Equal(int16(16)))
		Expect(varNames.int32).To(Equal(int32(32)))
		Expect(varNames.int64).To(Equal(int64(64)))
		Expect(varNames.uint).To(Equal(uint(2)))
		Expect(varNames.uint8).To(Equal(uint8(8)))
		Expect(varNames.uint16).To(Equal(uint16(16)))
		Expect(varNames.uint32).To(Equal(uint32(32)))
		Expect(varNames.uint64).To(Equal(uint64(64)))
	})

	It("uses value from the context (more obscure)", func() {
		var varNames struct {
			float32    float32
			float64    float64
			duration   time.Duration
			file       *cli.File
			fileSet    *cli.FileSet
			mmap       map[string]string
			nameValue  *cli.NameValue
			nameValues []*cli.NameValue
			url        *url.URL
			regexp     *regexp.Regexp
			ip         net.IP
			bigInt     *big.Int
			bigFloat   *big.Float
			bytes      []byte
		}
		evaluators := []expr.Evaluator{
			bind.Evaluator(evalFactory(&varNames.float32), bind.Float32("float32")),
			bind.Evaluator(evalFactory(&varNames.float64), bind.Float64("float64")),
			bind.Evaluator(evalFactory(&varNames.duration), bind.Duration("duration")),
			bind.Evaluator(evalFactory(&varNames.file), bind.File("file")),
			bind.Evaluator(evalFactory(&varNames.fileSet), bind.FileSet("fileSet")),
			bind.Evaluator(evalFactory(&varNames.mmap), bind.Map("mmap")),
			bind.Evaluator(evalFactory(&varNames.nameValue), bind.NameValue("nameValue")),
			bind.Evaluator(evalFactory(&varNames.nameValues), bind.NameValues("nameValues")),
			bind.Evaluator(evalFactory(&varNames.url), bind.URL("url")),
			bind.Evaluator(evalFactory(&varNames.regexp), bind.Regexp("regexp")),
			bind.Evaluator(evalFactory(&varNames.ip), bind.IP("ip")),
			bind.Evaluator(evalFactory(&varNames.bigInt), bind.BigInt("bigInt")),
			bind.Evaluator(evalFactory(&varNames.bigFloat), bind.BigFloat("bigFloat")),
			bind.Evaluator(evalFactory(&varNames.bytes), bind.Bytes("bytes")),
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "float32", Value: box(float32(32))},
				{Name: "float64", Value: box(float64(64))},
				{Name: "duration", Value: box(2 * time.Second)},
				{Name: "file", Value: &cli.File{}},
				{Name: "fileSet", Value: &cli.FileSet{}},
				{Name: "mmap", Value: box(map[string]string{"m": "a"})},
				{Name: "nameValue", Value: &cli.NameValue{Name: "n", Value: "v"}},
				{Name: "nameValues", Value: cli.NameValues("a", "b")},
				{Name: "url", Value: box(unwrap(url.Parse("http://go.com")))},
				{Name: "regexp", Value: box(regexp.MustCompile("[a-z]"))},
				{Name: "ip", Value: box(net.ParseIP("127.0.0.1"))},
				{Name: "bigInt", Value: box(big.NewInt(200))},
				{Name: "bigFloat", Value: box(big.NewFloat(100))},
				{Name: "bytes", Value: box([]byte{1, 2})},
			},
			Action: func(c context.Context) {
				for _, e := range evaluators {
					e.Evaluate(c, nil, nil)
				}
			},
		}

		app.RunContext(context.Background(), []string{})
		Expect(varNames.float32).To(Equal(float32(32)))
		Expect(varNames.float64).To(Equal(float64(64)))
		Expect(varNames.duration).To(Equal(2 * time.Second))
		Expect(varNames.file).To(BeAssignableToTypeOf(&cli.File{}))
		Expect(varNames.fileSet).To(BeAssignableToTypeOf(&cli.FileSet{}))
		Expect(varNames.mmap).To(Equal(map[string]string{"m": "a"}))
		Expect(varNames.nameValue).To(Equal(&cli.NameValue{Name: "n", Value: "v"}))
		Expect(varNames.nameValues).To(Equal(*cli.NameValues("a", "b")))
		Expect(varNames.url).To(Equal(unwrap(url.Parse("http://go.com"))))
		Expect(varNames.regexp).To(Equal(regexp.MustCompile("[a-z]")))
		Expect(varNames.ip).To(Equal(net.ParseIP("127.0.0.1")))
		Expect(varNames.bigInt).To(Equal(big.NewInt(200)))
		Expect(varNames.bigFloat).To(Equal(big.NewFloat(100)))
		Expect(varNames.bytes).To(Equal([]byte{1, 2}))
	})

	It("uses value from the context (interface{})", func() {
		var varNames struct {
			iface any
		}
		fakeValue := &joeclifakes.FakeValue{}
		evaluators := []expr.Evaluator{
			bind.Evaluator(evalFactory(&varNames.iface), bind.Interface("iface")),
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "iface", Value: fakeValue},
			},
			Action: func(c context.Context) {
				for _, e := range evaluators {
					e.Evaluate(c, nil, nil)
				}
			},
		}

		app.RunContext(context.Background(), []string{})
		Expect(varNames.iface).To(Equal(fakeValue))
	})

	It("uses value from the context (Value)", func() {
		var varNames struct {
			value *color.Mode
		}
		var mode color.Mode = color.Never
		evaluators := []expr.Evaluator{
			bind.Evaluator(evalFactory(&varNames.value), bind.Value[*color.Mode]("mode")),
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "mode", Value: &mode},
			},
			Action: func(c context.Context) {
				for _, e := range evaluators {
					e.Evaluate(c, nil, nil)
				}
			},
		}

		app.RunContext(context.Background(), []string{})
		Expect(*varNames.value).To(Equal(color.Never))
	})

	It("uses resetable value from the multiple bindings", func() {
		var seen []*cli.NameValue

		evaluatorThunk := func(nv *cli.NameValue) expr.Evaluator {
			c := *nv // Must copy the value so we have each instance that occurred
			seen = append(seen, &c)
			return expr.AlwaysTrue
		}
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name:     "more",
								Args:     cli.Args("i", new(cli.NameValue)),
								Evaluate: bind.Evaluator(evaluatorThunk, bind.NameValue("i")),
							},
						},
					},
				},
			},
			Action: func(c *cli.Context) {
				expr.FromContext(c, "expression").Evaluate(c, 0)
			},
		}
		args, _ := cli.Split("app -- -more 1 -more 2 -more 3")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(seen).To(Equal([]*cli.NameValue{
			{Name: "1", Value: "true"},
			{Name: "2", Value: "true"},
			{Name: "3", Value: "true"},
		}))
	})

	Describe("binding arguments", func() {
		DescribeTable("examples", func(binder bind.Binder[string], expected string) {
			var (
				eval       = new(exprfakes.FakeEvaluator)
				calledWith string
			)

			factory := func(s string) expr.Evaluator {
				calledWith = s
				return eval
			}
			app := &cli.App{
				Action: func(c *cli.Context) {
					expr.FromContext(c, "expression").Evaluate(c, 0)
				},
				Flags: []*cli.Flag{
					{
						Name: "flag",
					},
				},
				Args: []*cli.Arg{
					{
						Name: "start",
						NArg: -2,
					},
					{
						Name: "expression",
						Value: &expr.Expression{
							Exprs: []*expr.Expr{
								{
									Name: "name",
									Args: []*cli.Arg{
										{Name: "a"},
									},
									Uses: bind.Evaluator(factory, binder),
								},
							},
						},
					},
				},
			}

			args, _ := cli.Split("app --flag=f_value . -name a_value")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())

			Expect(eval.EvaluateCallCount()).To(Equal(1))
			Expect(calledWith).To(Equal(expected))
		},
			Entry("named", bind.String("a"), "a_value"),
			Entry("implicit name", bind.String(), "a_value"),
			Entry("flag name", bind.String("flag"), "f_value"),
		)
	})

	It("can bind implicit names in Evaluate", func() {
		var (
			eval       = new(exprfakes.FakeEvaluator)
			calledWith string
		)

		factory := func(s string) expr.Evaluator {
			calledWith = s
			return eval
		}
		app := &cli.App{
			Action: func(c *cli.Context) {
				expr.FromContext(c, "expression").Evaluate(c, 0)
			},
			Args: []*cli.Arg{{Name: "start", NArg: -2},
				{
					Name: "expression",
					Value: &expr.Expression{
						Exprs: []*expr.Expr{
							{
								Name:     "name",
								Args:     cli.Args("a", new(string)),
								Evaluate: bind.Evaluator(factory, bind.String()),
							},
						},
					},
				},
			},
		}

		args, _ := cli.Split("app . -name a_value")
		err := app.RunContext(context.Background(), args)
		Expect(err).NotTo(HaveOccurred())

		Expect(eval.EvaluateCallCount()).To(Equal(1))
		Expect(calledWith).To(Equal("a_value"))
	})
})

var _ = Describe("Indirect", func() {

	It("copies the implied value of the function", func() {
		fs := &cli.FileSet{Recursive: true}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "no-recursive",
					Uses: bind.Indirect("files", (*cli.FileSet).SetRecursive, false),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		app.Initialize(context.Background())
		Expect(app.Flags[0].Value).To(Equal(new(bool)))
	})

	It("invokes bind func with static value", func() {
		fs := &cli.FileSet{Recursive: true}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "no-recursive",
					Value: new(bool),
					Uses:  bind.Indirect("files", (*cli.FileSet).SetRecursive, false),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app --no-recursive .")
		_ = app.RunContext(context.Background(), args)
		Expect(fs.Recursive).To(BeFalse())
	})

	It("invokes bind func with corresponding value", func() {
		var calledWith string
		fs := new(cli.FileSet)
		act := new(joeclifakes.FakeAction)
		call := func(_ *cli.FileSet, s string) error {
			calledWith = s
			return nil
		}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:   "recursive",
					Value:  new(string),
					Action: act,
					Uses:   bind.Indirect("files", call),
				},
			},
			Args: []*cli.Arg{
				{
					Name:  "files",
					Value: fs,
				},
			},
		}
		args, _ := cli.Split("app --recursive YES .")
		_ = app.RunContext(context.Background(), args)
		Expect(act.ExecuteCallCount()).To(Equal(1), "action should still be called")
		Expect(calledWith).To(Equal("YES"))
	})
})

var _ = Describe("Redirect", func() {

	It("copies the implied value of the function", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "t",
				},
				{
					Name: "u",
					Uses: bind.Redirect[uint64]("t"),
				},
			},
		}
		app.Initialize(context.Background())
		Expect(app.Flags[0].Value).To(Equal(new(uint64)))
		Expect(app.Flags[1].Value).To(Equal(new(uint64)))
	})

	It("invokes bind func with static value", func() {
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name: "t",
				},
				{
					Name: "u",
					Uses: bind.Redirect[uint64]("t", 420),
				},
			},
		}

		args, _ := cli.Split("app -u")
		_ = app.RunContext(context.Background(), args)
		Expect(app.Flags[0].Value).To(PointTo(Equal(uint64(420))))
		Expect(app.Flags[1].Value).To(PointTo(Equal(true)))
	})
})

func evalFactory[T any](t *T) func(T) expr.Evaluator {
	return func(s T) expr.Evaluator {
		*t = s
		return new(exprfakes.FakeEvaluator)
	}
}

func newContext(app *cli.App) context.Context {
	ctxt, _ := app.Initialize(context.Background())
	return ctxt
}
