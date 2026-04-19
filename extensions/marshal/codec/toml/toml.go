package toml

import (
	"io"

	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	"github.com/pelletier/go-toml/v2"
)

type tomlCodec struct {
}

func init() {
	marshal.RegisterCodec(marshal.TOML, NewTOMLCodec)
}

// NewTOMLCodec creates the TOML codec
func NewTOMLCodec() codec.Interface {
	return &tomlCodec{}
}

func (*tomlCodec) MarshalWrite(w io.Writer, in any) error {
	e := toml.NewEncoder(w)
	return e.Encode(in)
}

func (*tomlCodec) UnmarshalRead(r io.Reader, out any) error {
	e := toml.NewDecoder(r)
	return e.Decode(out)
}
