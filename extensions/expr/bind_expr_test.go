package expr_test

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
)

var _ = Describe("Evaluator", func() {

	It("invokes the factory", func() {
		var called bool
		factory := func(string) expr.Evaluator {
			called = true
			return new(exprfakes.FakeEvaluator)
		}

		actual := expr.BindEvaluator(factory, bind.String())
		_ = actual.Evaluate(newContext(new(cli.App)), nil, nil)
		Expect(called).To(BeTrue())
	})

	It("delegates Evaluate calls the factory-produced evaluator", func() {
		fakeEvaluator := new(exprfakes.FakeEvaluator)
		factory := func(string) expr.Evaluator {
			return fakeEvaluator
		}

		actual := expr.BindEvaluator(factory, bind.String())
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
			expr.BindEvaluator(evalFactory(&varNames.string), bind.String("string")),
			expr.BindEvaluator(evalFactory(&varNames.bool), bind.Bool("bool")),
			expr.BindEvaluator(evalFactory(&varNames.list), bind.List("list")),
			expr.BindEvaluator(evalFactory(&varNames.int), bind.Int("int")),
			expr.BindEvaluator(evalFactory(&varNames.int8), bind.Int8("int8")),
			expr.BindEvaluator(evalFactory(&varNames.int16), bind.Int16("int16")),
			expr.BindEvaluator(evalFactory(&varNames.int32), bind.Int32("int32")),
			expr.BindEvaluator(evalFactory(&varNames.int64), bind.Int64("int64")),
			expr.BindEvaluator(evalFactory(&varNames.uint), bind.Uint("uint")),
			expr.BindEvaluator(evalFactory(&varNames.uint8), bind.Uint8("uint8")),
			expr.BindEvaluator(evalFactory(&varNames.uint16), bind.Uint16("uint16")),
			expr.BindEvaluator(evalFactory(&varNames.uint32), bind.Uint32("uint32")),
			expr.BindEvaluator(evalFactory(&varNames.uint64), bind.Uint64("uint64")),
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "string",
					Value: new("value set"),
				},
				{
					Name:  "bool",
					Value: new(true),
				},
				{Name: "list", Value: new([]string{"list"})},
				{Name: "int", Value: new(2)},
				{Name: "int8", Value: new(int8(8))},
				{Name: "int16", Value: new(int16(16))},
				{Name: "int32", Value: new(int32(32))},
				{Name: "int64", Value: new(int64(64))},
				{Name: "uint", Value: new(uint(2))},
				{Name: "uint8", Value: new(uint8(8))},
				{Name: "uint16", Value: new(uint(16))},
				{Name: "uint32", Value: new(uint(32))},
				{Name: "uint64", Value: new(uint(64))},
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
			expr.BindEvaluator(evalFactory(&varNames.float32), bind.Float32("float32")),
			expr.BindEvaluator(evalFactory(&varNames.float64), bind.Float64("float64")),
			expr.BindEvaluator(evalFactory(&varNames.duration), bind.Duration("duration")),
			expr.BindEvaluator(evalFactory(&varNames.file), bind.File("file")),
			expr.BindEvaluator(evalFactory(&varNames.fileSet), bind.FileSet("fileSet")),
			expr.BindEvaluator(evalFactory(&varNames.mmap), bind.Map("mmap")),
			expr.BindEvaluator(evalFactory(&varNames.nameValue), bind.NameValue("nameValue")),
			expr.BindEvaluator(evalFactory(&varNames.nameValues), bind.NameValues("nameValues")),
			expr.BindEvaluator(evalFactory(&varNames.url), bind.URL("url")),
			expr.BindEvaluator(evalFactory(&varNames.regexp), bind.Regexp("regexp")),
			expr.BindEvaluator(evalFactory(&varNames.ip), bind.IP("ip")),
			expr.BindEvaluator(evalFactory(&varNames.bigInt), bind.BigInt("bigInt")),
			expr.BindEvaluator(evalFactory(&varNames.bigFloat), bind.BigFloat("bigFloat")),
			expr.BindEvaluator(evalFactory(&varNames.bytes), bind.Bytes("bytes")),
		}

		app := &cli.App{
			Flags: []*cli.Flag{
				{Name: "float32", Value: new(float32(32))},
				{Name: "float64", Value: new(float64(64))},
				{Name: "duration", Value: new(2 * time.Second)},
				{Name: "file", Value: &cli.File{}},
				{Name: "fileSet", Value: &cli.FileSet{}},
				{Name: "mmap", Value: new(map[string]string{"m": "a"})},
				{Name: "nameValue", Value: &cli.NameValue{Name: "n", Value: "v"}},
				{Name: "nameValues", Value: cli.NameValues("a", "b")},
				{Name: "url", Value: new(unwrap(url.Parse("http://go.com")))},
				{Name: "regexp", Value: new(regexp.MustCompile("[a-z]"))},
				{Name: "ip", Value: new(net.ParseIP("127.0.0.1"))},
				{Name: "bigInt", Value: new(big.NewInt(200))},
				{Name: "bigFloat", Value: new(big.NewFloat(100))},
				{Name: "bytes", Value: new([]byte{1, 2})},
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
			expr.BindEvaluator(evalFactory(&varNames.iface), bind.Interface("iface")),
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
			expr.BindEvaluator(evalFactory(&varNames.value), bind.Value[*color.Mode]("mode")),
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
								Evaluate: expr.BindEvaluator(evaluatorThunk, bind.NameValue("i")),
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
									Uses: expr.BindEvaluator(factory, binder),
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
								Evaluate: expr.BindEvaluator(factory, bind.String()),
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

func unwrap[V any](v V, _ any) V {
	return v
}
