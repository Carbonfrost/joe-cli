package marshal

import (
	"fmt"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
)

// Codec identifies the support codecs. The JSON codec is supported by default.
// To add support for additional codecs, you must import them or register them.
// For example,
//
//	import _ "github.com/Carbonfrost/joe-cli/extensions/marshal/codec/toml"
type Codec int

const (
	JSON Codec = iota
	YAML
	TOML
	maxCodec
)

var (
	codecs = map[Codec]func() codec.Interface{
		JSON: codec.NewJSONCodec,
	}

	codecNames = [maxCodec]string{
		"json",
		"yaml",
		"toml",
	}

	codecHelpText = [maxCodec]string{
		JSON: "JSON format",
		YAML: "YAML format",
		TOML: "TOML format",
	}
)

// CodecRegistry provides a provider.Registry that enumerates the supported codecs.
// Each codec is instanced from a factory that takes Options as its argument, and
// only codecs which have been registered are listed.  Use ListCodecs to expose the
// registry via a flag.
var CodecRegistry = &provider.Registry{
	Name:      "codec",
	Providers: codecLookup{},
}

type codecLookup struct{}

func (codecLookup) ProviderNames() []string {
	names := make([]string, 0, len(codecs))
	for c := range codecs {
		names = append(names, c.String())
	}
	return names
}

func (codecLookup) LookupProvider(name string) (provider.Detail, bool) {
	c, ok := codecByName(name)
	if !ok || !c.Available() {
		return provider.Detail{}, false
	}
	return provider.Detail{
		Defaults: map[string]string{
			"disallow_unknown_fields": "false",
			"indent_size":             "2",
			"indent_style":            "space",
		},
		HelpText: codecHelpText[c],
		Factory: provider.FactoryOf(func(o codec.Options) (codec.Interface, error) {
			return c.New(o)
		}),
	}, true
}

// ListCodecs provides an action that lists the supported codecs then exits.
// This action only works if the Registry has been installed into the context;
// otherwise, it produces an error
func ListCodecs() cli.Action {
	return cli.Pipeline(
		cli.At(cli.ActionTiming, requireRegistry()),
		provider.ListProviders(CodecRegistry.Name),
	)
}

func requireRegistry() cli.ActionFunc {
	return func(c *cli.Context) error {
		if _, ok := provider.Services(c).LookupRegistry(CodecRegistry.Name); ok {
			return nil
		}
		return fmt.Errorf("no codecs registered")
	}
}

// codecByName resolves the Codec that corresponds to the given name.
func codecByName(name string) (Codec, bool) {
	for i, n := range codecNames {
		if n == name {
			return Codec(i), true
		}
	}
	return 0, false
}

var _ provider.Lookup = codecLookup{}

// RegisterCodec provides the behavior of registering a codec. This is expected
// to be called by implementations in their package initializer
func RegisterCodec(c Codec, f func() codec.Interface) {
	if c >= maxCodec {
		panic("marshal: RegisterCodec of unknown codec function")
	}
	codecs[c] = f
}

// Available indicates whether the codec type is registered
func (c Codec) Available() bool {
	_, ok := codecs[c]
	return ok
}

// New creates an instance of the given codec
func (c Codec) New(opts ...codec.Option) (codec.Interface, error) {
	return codec.WithOptions(codecs[c](), opts...)
}

// String provides the name of the codec
func (c Codec) String() string {
	return codecNames[c]
}

// DisallowUnknownFields affects unmarshaling and prevents unknown fields from
// being specified.
func DisallowUnknownFields() codec.Option {
	return codec.DisallowUnknownFields()
}

// WithIndent affects marshaling and sets the string used for each level of
// indentation in the encoded output.
func WithIndent(indent string) codec.Option {
	return codec.WithIndent(indent)
}
