// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import (
	"context"
	"fmt"
	"io"
	"os"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	"github.com/Carbonfrost/joe-cli/extensions/provider"
	"github.com/Carbonfrost/joe-cli/extensions/structure"
)

// Codec identifies the support codecs. The JSON codec is supported by default.
// To add support for additional codecs, you must import them or register them.
// For example,
//
//	import _ "github.com/Carbonfrost/joe-cli/extensions/marshal/codec/toml"
type Codec int

// The available formats for marshaling and unmarshaling data
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

type key string

const (
	contextProviderKey key = "contextProvider"
)

// CodecProvider provides the context-bound provider that can be
// used as a codec.
type CodecProvider struct {
	cli.Action

	c codec.Interface
}

// Apply will apply the given options to the provider
func (c *CodecProvider) Apply(opts ...CodecProviderOption) {
	for _, o := range opts {
		o(c)
	}
}
func (c *CodecProvider) Pipeline() cli.Action {
	return c.Action
}

// MarshalWrite provides marshaling by delegating to the internal codec if it exists.
// JSON is used by default
func (c *CodecProvider) MarshalWrite(w io.Writer, in any) error {
	return c.impl().MarshalWrite(w, in)
}

// UnmarshalRead provides unmarshaling by delegating to the internal codec if it exists.
// JSON is used by default
func (c *CodecProvider) UnmarshalRead(r io.Reader, out any) error {
	return c.impl().UnmarshalRead(r, out)
}

func (c *CodecProvider) impl() codec.Interface {
	if c == nil || c.c == nil {
		return codec.NewJSONCodec()
	}
	return c.c
}

// NewCodecProvider provides a value that provides the codec to
// use when dumping. By default, adding the provider to the pipeline adds it as a
// context service which facilitates configuring the codec used
// by Dump
func NewCodecProvider(opts ...CodecProviderOption) *CodecProvider {
	c := &CodecProvider{}
	c.Apply(defaultOptions()...)
	c.Apply(opts...)
	return c
}

// CodecProviderOption provides options for the provider
type CodecProviderOption func(*CodecProvider)

func defaultOptions() []CodecProviderOption {
	return []CodecProviderOption{
		WithDefaultAction(),
	}
}

// WithAction sets the action to use with the codec
func WithAction(a cli.Action) CodecProviderOption {
	return CodecProviderOption(func(v *CodecProvider) {
		v.Action = a
	})
}

// WithDefaultAction sets the action to the default, which sets the
// CodecProvider into the context and sets up the flags:
// SetOutput, SetOutputArgument, and ListCodecs
func WithDefaultAction() CodecProviderOption {
	return CodecProviderOption(func(v *CodecProvider) {
		v.Action = cli.Pipeline(
			CodecRegistry,
			ContextValue(v),
			codecProviderFlagsAndArgs(),
		)
	})
}

// CodecProviderFromContext retrieves the codec provider from the context
func CodecProviderFromContext(ctx context.Context) *CodecProvider {
	res, err := tryFromContext(ctx)
	if err != nil {
		panic(err)
	}
	return res
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *CodecProvider.
func ContextValue(v *CodecProvider) cli.Action {
	return cli.WithContextValue(contextProviderKey, v)
}

func tryFromContext(ctx context.Context) (*CodecProvider, error) {
	var zero *CodecProvider
	res, ok := ctx.Value(contextProviderKey).(*CodecProvider)
	if ok {
		return res, nil
	}
	return zero, fmt.Errorf("expected %s value not present in context", contextProviderKey)
}

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
	defaults := map[string]string{
		"disallow_unknown_fields": "false",
		"indent_size":             "2",
		"indent_style":            "space",
	}
	if c.supportsEscapeHTML() {
		defaults["escape_html"] = "false"
	}
	return provider.Detail{
		Defaults: defaults,
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

// Dump provides an action which dumps out the specified values to
// stdout using the codec in the context. See DumpContext.
func Dump(v ...any) cli.Action {
	return cli.ActionOf(func(c context.Context) error {
		return DumpContext(c, v...)
	})
}

// DumpContext prints the specified values to stdout. If the context provides
// the CLI context, then the stdout writer specified by it will be used; otherwise,
// os.Stdout will be used. The codec to use will be retrieved from the context;
// however, as a special case, if the first
// value is itself is a [codec.Interface] (or merely implements its MarshalWrite method),
// this specifies the codec to use rather than the one in the context.
// New lines separate each item.
func DumpContext(ctx context.Context, v ...any) error {
	if len(v) == 0 {
		return nil
	}

	type marshalWriter interface {
		MarshalWrite(w io.Writer, in any) error
	}

	var writer io.Writer = os.Stdout
	var c marshalWriter

	if c, ok := cli.TryFromContext(ctx); ok {
		writer = c.Stdout
	}

	if s, ok := v[0].(marshalWriter); ok {
		c = s
		v = v[1:]
	}
	if c == nil {
		c, _ = tryFromContext(ctx)
	}
	if c == nil {
		c = codec.NewJSONCodec()
	}
	for _, value := range v {
		err := c.MarshalWrite(writer, value)
		if err != nil {
			return err
		}
		writer.Write([]byte("\n"))
	}
	return nil
}

func codecProviderFlagsAndArgs() cli.Action {
	return cli.AddFlags([]*cli.Flag{
		{Uses: SetOutput()},
		{Uses: SetOutputArgument()},
		{Uses: ListCodecs()},
	}...)
}

// SetOutput provides a flag which sets the codec to use for dumping
func SetOutput(v ...Codec) cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name: "output",
			Value: &provider.Value{
				Registry: "codecs",
				Args:     structure.Of(&codec.Options{}),
			},
		},
		bind.Call3(
			(*CodecProvider).setHelper,
			bind.FromContext(CodecProviderFromContext),
			provider.BindValue().Name(),
			bind.Seq(provider.BindValue().Args(), func(s any) (codec.Options, error) {
				opts := s.(*structure.Value).V.(*codec.Options)
				if opts == nil {
					return codec.Options{}, nil
				}
				return *opts, nil
			}),
		),
	)
}

func (c *CodecProvider) setHelper(name string, opts codec.Options) error {
	n, ok := codecByName(name)
	if !ok {
		return fmt.Errorf("")
	}

	res, err := n.New(opts)
	if err != nil {
		return err
	}
	c.SetCodec(res)
	return nil
}

// SetCodec sets the codec used internally by the provider
func (c *CodecProvider) SetCodec(in codec.Interface) {
	c.c = in
}

// Codec gets the codec used internally by the provider
func (c *CodecProvider) Codec() codec.Interface {
	return c.c
}

// SetOutputArgument provides a flag which sets an argument on the codec
// uses for dumping
func SetOutputArgument(v ...Codec) cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name: "output-arg",
		},
		provider.SetArgument("output"),
	)
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

func (c Codec) supportsEscapeHTML() bool {
	_, err := c.New(EscapeHTML())
	return err == nil
}

// DisallowUnknownFields affects unmarshaling and prevents unknown fields from
// being specified.
func DisallowUnknownFields() codec.Option {
	return codec.DisallowUnknownFields()
}

// EscapeHTML affects marshaling and generates escaped HTML within JSON.
// For other codecs, this option generates an error.
func EscapeHTML() codec.Option {
	return codec.EscapeHTML()
}

// WithIndent affects marshaling and sets the string used for each level of
// indentation in the encoded output.
func WithIndent(indent string) codec.Option {
	return codec.WithIndent(indent)
}

var (
	_ codec.Interface = (*CodecProvider)(nil)
)
