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
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
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

type codecDir int

const (
	outputCodec codecDir = iota
	inputCodec
)

func (c codecDir) String() string {
	if c == inputCodec {
		return "input"
	}
	return "output"
}

// CodecProvider provides the context-bound provider that can be
// used as a codec.
// An input and output codecs can be set with SetInputCodec
// and SetOutputCodec, respectively. If no codec is specified, the
// default is JSON.
type CodecProvider struct {
	cli.Action

	in, out codec.Interface
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
	return c.impl(outputCodec).MarshalWrite(w, in)
}

// UnmarshalRead provides unmarshaling by delegating to the internal codec if it exists.
// JSON is used by default
func (c *CodecProvider) UnmarshalRead(r io.Reader, out any) error {
	return c.impl(inputCodec).UnmarshalRead(r, out)
}

func (c *CodecProvider) impl(dir codecDir) codec.Interface {
	if c == nil {
		return codec.NewJSONCodec()
	}
	var result codec.Interface
	if dir == inputCodec {
		result = c.InputCodec()
	} else {
		result = c.OutputCodec()
	}
	if result == nil {
		return codec.NewJSONCodec()
	}
	return result
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
// SetOutput, SetOutputArgument, and ListCodecs.
// However, SetInput and SetInputArgument are not added by the default
// action.
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

const codecRegistryName = "codec"

// CodecRegistry provides a provider.Registry that enumerates the supported codecs.
// Each codec is instanced from a factory that takes Options as its argument, and
// only codecs which have been registered are listed.  Use ListCodecs to expose the
// registry via a flag.
var CodecRegistry = &provider.Registry{
	Name:      codecRegistryName,
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
	defaults := map[string]any{
		"disallow_unknown_fields": false,
		"indent_size":             2,
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

// Dumper provides an expr.Evaluator which prints each value it is given using
// the codec configured within the context.  Printing is done by DumpContext,
// which means that the codec and writer are obtained from the context.  The
// value is always yielded to the rest of the expression pipeline, so a Dumper
// can be introduced anywhere within an expression to observe the values which
// flow through it.  The zero value is ready to use.
type Dumper struct{}

// Evaluate implements the Evaluator interface from the expr extension by
// dumping v and then yielding it.
func (Dumper) Evaluate(ctx context.Context, v any, yield func(any) error) error {
	if err := DumpContext(ctx, v); err != nil {
		return err
	}
	return yield(v)
}

func codecProviderFlagsAndArgs() cli.Action {
	return cli.AddFlags([]*cli.Flag{
		{Uses: SetOutput()},
		{Uses: SetOutputArgument()},
		{Uses: ListCodecs()},
	}...)
}

func (c *CodecProvider) setCodecHelper(name codecDir, v codec.Interface) error {
	if name == inputCodec {
		c.SetInputCodec(v)
	} else {
		c.SetOutputCodec(v)
	}
	return nil
}

// InputCodec gets the codec used internally by the provider for input
func (c *CodecProvider) InputCodec() codec.Interface {
	return c.in
}

// OutputCodec gets the codec used internally by the provider for output
func (c *CodecProvider) OutputCodec() codec.Interface {
	return c.out
}

func (c *CodecProvider) SetInputCodec(v codec.Interface) {
	c.in = v
}

func (c *CodecProvider) SetOutputCodec(v codec.Interface) {
	c.out = v
}

// SetOutput provides a flag which sets the codec to use for input
func SetOutput(v ...codec.Interface) cli.Action {
	return setCodec(outputCodec, v...)
}

// SetOutputArgument provides a flag which sets an argument on the codec
// uses for input
func SetOutputArgument() cli.Action {
	return setCodecArgument(outputCodec)
}

// SetInput provides a flag which sets the codec to use for input
func SetInput(v ...codec.Interface) cli.Action {
	return setCodec(inputCodec, v...)
}

// SetInputArgument provides a flag which sets an argument on the codec
// uses for input
func SetInputArgument() cli.Action {
	return setCodecArgument(inputCodec)
}

func setCodec(name codecDir, v ...codec.Interface) cli.Action {
	actualBind := provider.Bind[codec.Interface]()
	if len(v) > 0 {
		actualBind = bind.Exact(v...)
	}

	return cli.Pipeline(
		cli.Prototype{
			Name:      name.String(),
			HelpText:  "Set the output {FORMAT}",
			UsageText: synopsis.Choices(CodecRegistry.ProviderNames()),
			Value: &provider.Value{
				Registry: codecRegistryName,
				Args:     structure.Of(&codec.Options{}),
			},
		},
		bind.Call3(
			(*CodecProvider).setCodecHelper,
			bind.FromContext(CodecProviderFromContext),
			bind.Exact(name),
			actualBind,
		),
	)
}

func setCodecArgument(name codecDir) cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name: name.String() + "-arg",
		},
		provider.SetArgument(name.String()),
		// We must set the codec again after an argument because of its internal
		// provider factory (provider.Bind)
		bind.Call3(
			(*CodecProvider).setCodecHelper,
			bind.FromContext(CodecProviderFromContext),
			bind.Exact(name),
			provider.Bind[codec.Interface](name.String()),
		),
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
	if !c.Available() {
		return nil, fmt.Errorf("codec not available: %s", c)
	}
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
