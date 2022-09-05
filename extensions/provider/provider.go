// Package provider implements a technique for identifying named providers with arguments.
// The main use case for providers is extensibility.  For example, in a command line tool that
// lets the user decide which output format to use, you could use providers to name the output
// format and their arguments.  For example, say a tool converts from one Go marshaler to another
// such as from YAML to JSON.  This would (say) enable the user to specify their desired output format:
//
//	conversiontool --format json --format-arg indent=2 --format-arg encoding=utf-16  inputFile.yaml
//
// Notice that --format names the desired output format (JSON) and --format-arg provides
// arguments that the JSON formatter presumably uses.
//
// Value is used to implement the provider specification, which is the name of the
// provider and optionally a shorthand for arg syntax.  In the case that you want a separate
// flag to provide the arguments, you use the SetArgument action.
//
// Usually, a registry of well-known provider names is used.  To support this, you can use a Registry.
package provider

import (
	"bytes"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

// Value implements a value which can be used as a flag that names a provider.
// A short syntax allows specifying parameters to the provider.  For example, given the flag
//
//	&cli.Flag{
//	   Name: "p"
//	   Value: new(provider.Provider),
//	}
//
// It becomes possible to specify the syntax -p Name,arg1=v,arg2=v, which provides the name
// of the provider to use and argument values to set.
type Value struct {
	// Name is the name of the provider to use
	Name string

	// Args provides the arguments to the provider.  Args should be any supported
	// flag type.  If unspecified, a map[string]string is used.
	Args interface{}

	setName bool
	rawArgs map[string]string
}

// Map provides a map that names the providers and their the default values.
type Map map[string]map[string]string

// Registry can be used to add validation to the Provider value and to determine what
// to be listed.  This value is used in the Uses
// pipeline of either the flag or its containing command.
type Registry struct {
	// Name of the registry, which is the same as the name of the flag
	Name string

	// Providers names each of the providers which are allowed with a mapping to
	// the provider's arguments' defaults.  For example, given the example in the
	// package overview, a valid initializer would be:
	//
	//  &provider.Registry{
	//      Providers: provider.Map{
	//          "json": {
	//              "indent":   "2",
	//              "encoding": "utf-16",
	//          },
	//      },
	//  }
	Providers map[string]map[string]string

	// AllowUnknown determines whether unknown provider names are allowed.
	// The default is false, which causes an error when trying to set an unknown
	// name.
	AllowUnknown bool
}

type providerData struct {
	Name     string
	Defaults defaultsMap
}

type defaultsMap map[string]string

var (
	listTemplate = `{{ range .Providers -}}
{{ .Name }}{{ "\t" }}{{ .Defaults }}
{{ end }}`
)

// ArgumentFlag obtains a conventions-based flag for setting an argument
func (v *Value) ArgumentFlag() cli.Prototype {
	return cli.Prototype{
		Name:     "arg",
		Value:    new(string),
		HelpText: "Sets an argument for %s",
		Setup: cli.Setup{
			Uses: cli.Bind(v.Set),
		},
	}
}

// SetArgument provides an action that can be used to set the argument for a provider.
// This enables you to have a dedicated flag to handle setting provider arguments:
//
//	 &cli.Flag{
//	    Name: "provider"
//	    Value: new(provider.Provider),
//	 },
//	 &cli.Flag {
//	    Name: "provider-arg",
//	    Uses: provider.SetArgument("provider"),
//	 }
//
//	Thus, the user could specify a provider using two flags as in:
//	     --provider download --provider-arg downloader=curl
//
//	If the action is set to initialize a flag that is unnamed, the suffix -arg is implied.
func SetArgument(name string) cli.Action {
	return cli.Prototype{
		Name:     name + "-arg",
		Value:    new(string),
		HelpText: fmt.Sprintf("Sets an argument for %s", name),
		Setup: cli.Setup{
			Action: cli.BindIndirect(name, (*Value).Set),
		},
	}
}

// ListProviders provides an action that can be used to display the list of providers.
// This requires that the provider registry was specified.
// If the action is set to initialize a flag that is unnamed, the prefix list- is implied.
// The template "providers" is used, which is set to a default if unspecified.
func ListProviders(name string) cli.Action {
	return cli.Prototype{
		Name:    "list-" + name,
		Value:   new(bool),
		Options: cli.Exits,
		Setup: cli.Setup{
			Optional: true,
			Uses:     cli.RegisterTemplate("Providers", listTemplate),
			Action: func(c *cli.Context) error {
				registry := Services(c).Registry(name)
				tpl := c.Template("Providers")
				data := struct {
					Providers []providerData
					Debug     bool
				}{
					Providers: toData(registry),
					Debug:     tpl.Debug,
				}

				return tpl.Execute(c.Stdout, data)
			},
		},
	}
}

// Set the text of the value.  Can be called successively to append.
func (v *Value) Set(arg string) error {
	if v.Args == nil {
		v.Args = &map[string]string{}
	}
	if !v.setName {
		args := strings.SplitN(arg, ",", 2)
		v.Name = args[0]
		v.setName = true

		if len(args) == 1 {
			return nil
		}
		arg = args[1]
	}
	cli.Set(&v.rawArgs, arg)
	return cli.Set(v.Args, arg)
}

// String obtains the textual representation
func (v *Value) String() string {
	var buf bytes.Buffer
	buf.WriteString(v.Name)

	switch val := v.Args.(type) {
	case map[string]string:
		buf.WriteString(",")
		buf.WriteString(support.FormatMap(val, ","))
	default:
		buf.WriteString(",")
		buf.WriteString(fmt.Sprint(val))
	}
	return buf.String()
}

func (v *Value) Initializer() cli.Action {
	return cli.Setup{
		Action: validateProviderExists,
	}
}

func validateProviderExists(c *cli.Context) error {
	v := c.Value("").(*Value)
	provider := strings.TrimLeft(c.Name(), "-")
	registry := Services(c).Registry(provider)
	if registry == nil {
		return nil
	}
	if registry.AllowUnknown {
		return nil
	}
	p, exists := registry.Providers[v.Name]
	if !exists {
		return fmt.Errorf("unknown %q %s", v.Name, provider)
	}
	for k := range v.rawArgs {
		if _, exists := p[k]; !exists {
			return fmt.Errorf("unknown argument %q for %q %s", k, v.Name, provider)
		}
	}

	return nil
}

func (r *Registry) Execute(c *cli.Context) error {
	return c.Before(cli.ActionFunc(func(c1 *cli.Context) error {
		Services(c1).registries[r.Name] = r
		return nil
	}))
}

func (d defaultsMap) String() string {
	return support.FormatMap(d, ", ")
}

func toData(r *Registry) []providerData {
	res := make([]providerData, 0)
	if r != nil {
		for n, d := range r.Providers {
			res = append(res, providerData{
				Name:     n,
				Defaults: d,
			})
		}
		sort.Slice(res, func(i, j int) bool {
			return res[i].Name < res[j].Name
		})
	}
	return res
}

var (
	_ flag.Value = (*Value)(nil)
	_ cli.Action = (*Registry)(nil)
)
