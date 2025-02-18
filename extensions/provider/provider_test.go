package provider_test

import (
	"bytes"
	"context"
	"strconv"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/Carbonfrost/joe-cli/extensions/structure"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Registry", func() {

	Describe("Execute", func() {
		It("sets up context with registry in services", func() {
			action := new(joeclifakes.FakeAction)
			registry := &provider.Registry{
				Name: "providers",
				Providers: provider.Map{
					"csv": {
						"comma":   "a",
						"useCRLF": "true",
					},
				},
			}

			app := &cli.App{
				Uses:   registry,
				Action: action,
			}

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			c := cli.FromContext(action.ExecuteArgsForCall(0))

			services := provider.Services(c)
			Expect(services.Registry("providers")).To(Equal(registry))
		})
	})

	Describe("New", func() {
		type csvProvider struct {
			Comma   string
			UseCRLF bool
		}

		It("creates provider given the factory and its defaults", func() {
			action := new(joeclifakes.FakeAction)
			registry := &provider.Registry{
				Name: "providers",
				Providers: provider.Details{
					"csv": {
						Defaults: map[string]string{
							"comma":   "a",
							"useCRLF": "true",
						},
						Factory: func(opts map[string]string) (any, error) {
							b, _ := strconv.ParseBool(opts["useCRLF"])
							return &csvProvider{
								Comma:   opts["comma"],
								UseCRLF: b,
							}, nil
						},
					},
				},
			}

			app := &cli.App{
				Uses:   registry,
				Action: action,
			}

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			c := cli.FromContext(action.ExecuteArgsForCall(0))

			actual, _ := provider.Services(c).Registry("providers").New("csv", nil)
			Expect(actual).To(Equal(&csvProvider{"a", true}))
		})

		It("returns an error on non-existent provider", func() {
			action := new(joeclifakes.FakeAction)
			registry := &provider.Registry{
				Name:      "providers",
				Providers: provider.Details{},
			}

			app := &cli.App{
				Uses:   registry,
				Action: action,
			}

			_ = app.RunContext(context.TODO(), []string{"app"})
			c := cli.FromContext(action.ExecuteArgsForCall(0))

			_, err := provider.Services(c).Registry("providers").New("csv", nil)
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("ProviderNames", func() {
		It("creates obtains the provider names", func() {
			action := new(joeclifakes.FakeAction)
			registry := &provider.Registry{
				Name: "providers",
				Providers: provider.Details{
					"csv":  {},
					"json": {},
					"yaml": {},
				},
			}

			app := &cli.App{
				Uses:   registry,
				Action: action,
			}

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			c := cli.FromContext(action.ExecuteArgsForCall(0))

			actual := provider.Services(c).Registry("providers").ProviderNames()
			Expect(actual).To(ConsistOf([]string{"csv", "json", "yaml"}))
		})
	})
})

var _ = Describe("Factory", func() {
	It("decodes options from map", func() {
		var called bool
		provider.Factory(func(o Options) any {
			Expect(o.A).To(Equal("A"))
			Expect(o.B).To(Equal("B"))
			called = true
			return nil
		})(map[string]string{
			"A": "A",
			"B": "B",
		})
		Expect(called).To(BeTrue())
	})

	It("can accept configuration options to return error", func() {
		var called bool
		_, err := provider.Factory(func(o Options) any {
			called = true
			return nil
		}, structure.ErrorUnused)(map[string]string{
			"Extra": "this field is not on the struct",
		})
		Expect(called).To(BeFalse())
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("has invalid keys: Extra")))
	})

	DescribeTable("examples", func(f any) {
		Expect(func() { provider.Factory(f) }).NotTo(Panic())
		actual, _ := provider.Factory(f)(nil)
		Expect(actual).To(Equal("provider"))
	},
		Entry("no-op", func(any) (any, error) { return "provider", nil }),
		Entry("no error", func(Options) any { return "provider" }),
		Entry("nominal", func(Options) (any, error) { return "provider", nil }),
	)

	DescribeTable("error", func(f any) {
		Expect(func() { provider.Factory(f) }).To(Panic())
	},
		Entry("need args", func() (any, error) { return "provider", nil }),
		Entry("need return value", func(Options) {}),
		Entry("too many to return", func(Options) (any, any, error) { return "", "", nil }),
	)
})

var _ = Describe("SetArgument", func() {

	It("sets up the argument by value", func() {
		value := new(provider.Value)
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:  "provider",
					Value: value,
					Uses:  cli.Accessory("-", (*provider.Value).ArgumentFlag),
				},
			},
		}

		arguments, _ := cli.Split("app --provider hello --provider-arg world=2")
		err := app.RunContext(context.TODO(), arguments)
		Expect(err).NotTo(HaveOccurred())
		Expect(value.Name).To(Equal("hello"))
		Expect(value.Args).To(Equal(&map[string]string{
			"world": "2",
		}))
	})

	It("implicitly sets up map[string]string", func() {
		v := new(provider.Value)
		err := cli.Set(v, "hello,world=2")
		Expect(err).NotTo(HaveOccurred())

		Expect(v.Name).To(Equal("hello"))
		Expect(v.Args).To(Equal(&map[string]string{
			"world": "2",
		}))
	})

	It("implicitly uses provider name and value", func() {
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Uses: provider.SetArgument("provider"),
				},
			},
		}

		_ = app.RunContext(context.TODO(), []string{"app"})
		Expect(app.Flags[0].Name).To(Equal("provider-arg"))
		Expect(app.Flags[0].Value).To(Equal(new(string)))
	})

	It("uses correct accessory flag HelpText", func() {
		// Addresses a bug where the incorrect HelpText was being used
		// since automatic %s expansion was removed in 3e976a
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:  "provider",
					Value: new(provider.Value),
					Uses:  cli.Accessory("-", (*provider.Value).ArgumentFlag),
				},
			},
		}

		_, _ = app.Initialize(context.TODO())
		flag, _ := app.Flag("provider-arg")
		Expect(flag.HelpText).To(Equal("Sets an argument for provider"))
	})
})

var _ = Describe("ListProviders", func() {

	It("prints output", func() {
		var (
			capture bytes.Buffer
		)
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Uses: &provider.Registry{
				Name: "providers",
				Providers: provider.Map{
					"csv": {
						"comma":   "a",
						"useCRLF": "true",
					},
					"json": {
						"indent": "true",
					},
				},
			},
			Flags: []*cli.Flag{
				{
					Name:  "list-providers",
					Value: new(bool),
					Uses:  provider.ListProviders("providers"),
				},
			},
		}

		_ = app.RunContext(context.TODO(), []string{"app", "--list-providers"})
		Expect(capture.String()).To(Equal(
			"csv\tcomma=a, useCRLF=true\n" +
				"json\tindent=true\n",
		))
	})

	It("can use custom template", func() {
		var (
			capture bytes.Buffer
		)
		app := &cli.App{
			Name:   "app",
			Stdout: &capture,
			Uses: cli.Pipeline(&provider.Registry{
				Name: "providers",
				Providers: provider.Details{
					"csv": {
						Defaults: map[string]string{
							"comma":   "a",
							"useCRLF": "true",
						},
						HelpText: "Use comma-separated values",
					},
					"json": {
						Defaults: map[string]string{
							"indent": "true",
						},
						HelpText: "Use JSON values",
					},
				},
			},
				cli.RegisterTemplate("Providers", `{{ range .Providers -}}
{{ .Name }} {{ .HelpText }}
{{ end }}`),
			),
			Flags: []*cli.Flag{
				{
					Name:  "list-providers",
					Value: new(bool),
					Uses:  provider.ListProviders("providers"),
				},
			},
		}

		_ = app.RunContext(context.TODO(), []string{"app", "--list-providers"})
		Expect(capture.String()).To(Equal("csv Use comma-separated values\njson Use JSON values\n"))
	})

	It("implicitly uses provider name and value", func() {
		app := &cli.App{
			Name: "app",
			Uses: &provider.Registry{
				Name: "providers",
			},
			Flags: []*cli.Flag{
				{
					Uses: provider.ListProviders("providers"),
				},
			},
		}

		_ = app.RunContext(context.TODO(), []string{"app"})
		Expect(app.Flags[0].Name).To(Equal("list-providers"))
		Expect(app.Flags[0].Value).To(Equal(new(bool)))
	})
})

var _ = Describe("Value", func() {

	type providerOptions struct {
		Comma   string
		UseCRLF bool
	}

	Describe("validation", func() {

		It("allows expected value", func() {
			app := &cli.App{
				Name: "app",
				Uses: &provider.Registry{
					Name:         "encoding",
					AllowUnknown: false,
					Providers: provider.Map{
						"utf8": {},
					},
				},
				Flags: []*cli.Flag{
					{
						Name: "encoding",
						Value: &provider.Value{
							Args: &map[string]string{},
						},
					},
				},
			}

			args, _ := cli.Split("app --encoding utf8")
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())
			Expect(app.Flags[0].Value.(*provider.Value).Name).To(Equal("utf8"))
		})

		DescribeTable("errors", func(arguments string, expected types.GomegaMatcher) {
			app := &cli.App{
				Name: "app",
				Uses: &provider.Registry{
					Name: "provider",
					Providers: provider.Map{
						"csv": {
							"comma": "default",
						},
					},
				},
				Flags: []*cli.Flag{
					{
						Name: "provider",
						Value: &provider.Value{
							Args: &map[string]string{},
						},
					},
				},
			}

			args, _ := cli.Split(arguments)
			err := app.RunContext(context.TODO(), args)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expected))
		},
			Entry(
				"invalid provider",
				"app --provider invalid",
				Equal(`unknown "invalid" provider`),
			),
			Entry(
				"invalid provider argument",
				"app --provider csv,unknownProperty=true",
				Equal(`unknown argument "unknownProperty" for "csv" provider`),
			),
		)

		It("allows unknown when specified", func() {
			app := &cli.App{
				Name: "app",
				Uses: &provider.Registry{
					Name:         "encoding",
					AllowUnknown: true,
					Providers:    provider.Map{},
				},
				Flags: []*cli.Flag{
					{
						Name: "encoding",
						Value: &provider.Value{
							Args: &map[string]string{},
						},
					},
				},
			}

			args, _ := cli.Split("app --encoding unknown")
			err := app.RunContext(context.TODO(), args)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("String", func() {
		DescribeTable("examples", func(v *provider.Value, expected string) {
			Expect(v.String()).To(Equal(expected))
		},
			Entry("map",
				&provider.Value{Name: "orson", Args: map[string]string{"w": "ean", "d": "ells"}},
				"orson,d=ells,w=ean"),
			Entry("structure",
				&provider.Value{Name: "orson", Args: structure.Of(&providerOptions{Comma: "true"})},
				"orson,Comma=true"),
		)
	})

	DescribeTable("examples", func(arguments string, expectedName string, expectedOpts types.GomegaMatcher) {
		opts := &providerOptions{}
		po := &provider.Value{
			Args: structure.Of(opts),
		}
		app := &cli.App{
			Name: "app",
			Flags: []*cli.Flag{
				{
					Name:  "provider",
					Value: po,
				},
			},
		}

		args, _ := cli.Split(arguments)
		err := app.RunContext(context.TODO(), args)
		Expect(err).NotTo(HaveOccurred())
		Expect(po.Name).To(Equal(expectedName))
		Expect(opts).To(expectedOpts)
	},
		Entry(
			"name only",
			"app --provider csv",
			"csv",
			Equal(&providerOptions{
				Comma:   "",
				UseCRLF: false,
			}),
		),
		Entry(
			"inline format",
			"app --provider csv,comma=A,useCRLF=true",
			"csv",
			Equal(&providerOptions{
				Comma:   "A",
				UseCRLF: true,
			}),
		),
		Entry(
			"repeated",
			"app --provider csv --provider comma=A --provider useCRLF=true",
			"csv",
			Equal(&providerOptions{
				Comma:   "A",
				UseCRLF: true,
			}),
		),
	)
})

type Options struct {
	A, B string
}
