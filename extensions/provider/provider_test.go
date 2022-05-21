package provider_test

import (
	"bytes"
	"context"

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

			c := action.ExecuteArgsForCall(0)

			services := provider.Services(c)
			Expect(services.Registry("providers")).To(Equal(registry))
		})
	})
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
		Expect(capture.String()).To(Equal("csv\tcomma=a, useCRLF=true\n"))
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

		DescribeTable("errors", func(arguments string, expected types.GomegaMatcher) {
			app := &cli.App{
				Name: "app",
				Uses: &provider.Registry{
					Name: "providers",
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
			XEntry(
				"invalid provider",
				"app --provider invalid",
				Equal("unexpected argument: invalid"),
			),
			XEntry(
				"invalid provider argument",
				"app --provider csv,unknownProperty=true",
				Equal("unexpected argument: unknownProperty"),
			),
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
