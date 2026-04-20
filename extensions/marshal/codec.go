package marshal

import (
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
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
)

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
