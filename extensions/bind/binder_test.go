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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("For", func() {

	DescribeTable("examples", func(fn func() any) {
		var actual any
		Expect(func() {
			actual = fn
		}).NotTo(Panic())
		Expect(actual).NotTo(BeNil())
	},
		Entry("Int", func() any { return bind.For[int]() }),
		Entry("Bool", func() any { return bind.For[bool]() }),
		Entry("String", func() any { return bind.For[string]() }),
		Entry("List", func() any { return bind.For[[]string]() }),
		Entry("Int", func() any { return bind.For[int]() }),
		Entry("Int8", func() any { return bind.For[int8]() }),
		Entry("Int16", func() any { return bind.For[int16]() }),
		Entry("Int32", func() any { return bind.For[int32]() }),
		Entry("Int64", func() any { return bind.For[int64]() }),
		Entry("Uint", func() any { return bind.For[uint]() }),
		Entry("Uint8", func() any { return bind.For[uint8]() }),
		Entry("Uint16", func() any { return bind.For[uint16]() }),
		Entry("Uint32", func() any { return bind.For[uint32]() }),
		Entry("Uint64", func() any { return bind.For[uint64]() }),
		Entry("Float32", func() any { return bind.For[float32]() }),
		Entry("Float64", func() any { return bind.For[float64]() }),
		Entry("Duration", func() any { return bind.For[time.Duration]() }),
		Entry("File", func() any { return bind.For[*cli.File]() }),
		Entry("FileSet", func() any { return bind.For[*cli.FileSet]() }),
		Entry("Map", func() any { return bind.For[map[string]string]() }),
		Entry("NameValue", func() any { return bind.For[*cli.NameValue]() }),
		Entry("NameValues", func() any { return bind.For[[]*cli.NameValue]() }),
		Entry("URL", func() any { return bind.For[*url.URL]() }),
		Entry("Regexp", func() any { return bind.For[*regexp.Regexp]() }),
		Entry("IP", func() any { return bind.For[net.IP]() }),
		Entry("BigInt", func() any { return bind.For[*big.Int]() }),
		Entry("BigFloat", func() any { return bind.For[*big.Float]() }),
		Entry("Bytes", func() any { return bind.For[[]byte]() }),
		Entry("Interface", func() any { return bind.For[any]() }),
	)
})

func box[T any](t T) *T {
	return &t
}

func unwrap[V any](v V, _ any) V {
	return v
}
