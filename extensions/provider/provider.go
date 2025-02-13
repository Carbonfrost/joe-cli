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
	"context"
	"flag"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/structure"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

// Value implements a value which can be used as a flag that names a provider.
// A short syntax allows specifying parameters to the provider.  For example, given the flag
//
//	&cli.Flag{
//	   Name: "p"
//	   Value: new(provider.Value),
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

// FactoryFunc describes the factory function for a provider
type FactoryFunc func(opts map[string]string) (any, error)

// Lookup defines how to obtain the provider or information about it from its name
type Lookup interface {
	ProviderNames() []string
	LookupProvider(name string) (Detail, bool)
}

// Details provides a lookup that provides information about a provider and a factory
// for instancing it.
type Details map[string]Detail

// Detail provides information about a provider
type Detail struct {
	// Defaults specifies the default values for the provider
	Defaults map[string]string

	// Factory is responsible for creating the provider given the options.
	// The value is a function, and must be one of the functions available to
	// FactoryFunc
	Factory FactoryFunc

	// Value is a discrete value to use for the provider.  Factory and Value
	// are designed to be mutually exclusive; however, when both as specified,
	// Factory is used.  The use of Value also implies that the provider has
	// no defaults.
	Value any

	// HelpText contains text which briefly describes the usage of the provider.
	HelpText string
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
	Providers Lookup

	// AllowUnknown determines whether unknown provider names are allowed.
	// The default is false, which causes an error when trying to set an unknown
	// name.
	AllowUnknown bool
}

type providerData struct {
	Name     string
	HelpText string
	Defaults defaultsMap
}

type defaultsMap map[string]string

var (
	listTemplate = `{{ range .Providers -}}
{{ .Name }}{{ "\t" }}{{ .Defaults }}
{{ end }}`
)

// Factory uses reflection to create a provider factory function.  In particular,
// the function argument must have a signature that takes one argument for
// options and return the provider and optionally an error.  The actual types can be
// more specific than interface{}.  When they are, type conversion is provided using
// the Decode method
func Factory(fn any, options ...structure.DecoderOption) FactoryFunc {
	if fn == nil {
		panic("cannot specify nil argument")
	}
	if !validFactory(fn) {
		panic("unexpected function signature")
	}

	v := reflect.ValueOf(fn)
	optType := reflect.TypeOf(fn).In(0)
	return FactoryFunc(func(opts map[string]string) (any, error) {
		if opts == nil {
			opts = map[string]string{}
		}

		value := reflect.New(optType).Interface()
		err := structure.Decode(opts, value, options...)
		if err != nil {
			return nil, err
		}
		o := reflect.ValueOf(value).Elem().Interface()

		out := v.Call([]reflect.Value{reflect.ValueOf(o)})
		if len(out) > 1 {
			err, _ = out[1].Interface().(error)
		}
		return out[0].Interface(), err
	})
}

func validFactory(fn any) bool {
	typ := reflect.TypeOf(fn)
	if typ.NumIn() != 1 {
		return false
	}
	if typ.NumOut() < 1 || typ.NumOut() > 2 {
		return false
	}
	var errorType = reflect.TypeFor[error]()
	if typ.NumOut() == 2 && typ.Out(1) != errorType {
		return false
	}
	return true
}

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
	return cli.Pipeline(
		&cli.Prototype{
			Name:     "list-" + name,
			Value:    new(bool),
			Options:  cli.Exits,
			HelpText: fmt.Sprintf("List available %s providers then exit", name),
		},
		cli.Initializer(cli.ActionFunc(fallbackTemplate)),
		cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) error {
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
		})),
	)
}

func fallbackTemplate(c *cli.Context) error {
	if c.Template("Providers") != nil {
		return nil
	}
	return c.Do(cli.RegisterTemplate("Providers", listTemplate))
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
	pro, exists := registry.LookupProvider(v.Name)
	p := pro.Defaults
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

func (r *Registry) ProviderNames() []string {
	return r.Providers.ProviderNames()
}

func (r *Registry) LookupProvider(name string) (Detail, bool) {
	return r.Providers.LookupProvider(name)
}

func (r *Registry) New(name string, opts map[string]string) (any, error) {
	pro, ok := r.LookupProvider(name)
	defaults := pro.Defaults
	fac := pro.Factory
	mergedOpts := map[string]string{}
	maps.Copy(mergedOpts, defaults)
	maps.Copy(mergedOpts, opts)

	if !ok {
		return nil, fmt.Errorf("provider not found: %q", name)
	}
	return fac(mergedOpts)
}

func (r *Registry) Execute(c context.Context) error {
	return cli.FromContext(c).Before(cli.ActionFunc(func(c1 *cli.Context) error {
		Services(c1).registries[r.Name] = r
		return nil
	}))
}

func (m Map) ProviderNames() []string {
	keys := make([]string, len(m))
	var i int
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

func (m Map) LookupProvider(name string) (d Detail, ok bool) {
	d.Defaults, ok = m[name]
	return
}

func (d Details) ProviderNames() []string {
	keys := make([]string, len(d))
	var i int
	for k := range d {
		keys[i] = k
		i++
	}
	return keys
}

func (d Details) LookupProvider(name string) (Detail, bool) {
	r, ok := d[name]
	return r, ok
}

func (d defaultsMap) String() string {
	return support.FormatMap(d, ", ")
}

func toData(r *Registry) []providerData {
	res := make([]providerData, 0)
	if r != nil {
		for _, n := range r.ProviderNames() {
			dd, _ := r.LookupProvider(n)
			res = append(res, providerData{
				Name:     n,
				Defaults: dd.Defaults,
				HelpText: dd.HelpText,
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
	_ Lookup     = (Map)(nil)
	_ Lookup     = (Details)(nil)
	_ Lookup     = (*Registry)(nil)
)
