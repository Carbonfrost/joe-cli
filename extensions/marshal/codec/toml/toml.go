// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package toml provides the TOML codec
package toml

import (
	"io"

	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	"github.com/pelletier/go-toml/v2"
)

type tomlCodec struct {
	disallowUnknownFields bool
	indent                string
}

func init() {
	marshal.RegisterCodec(marshal.TOML, NewTOMLCodec)
}

// NewTOMLCodec creates the TOML codec
func NewTOMLCodec() codec.Interface {
	return &tomlCodec{}
}

func (t *tomlCodec) MarshalWrite(w io.Writer, in any) error {
	e := toml.NewEncoder(w)
	if t.indent != "" {
		e.SetIndentSymbol(t.indent)
		e.SetIndentTables(true)
	}
	return e.Encode(in)
}

func (t *tomlCodec) UnmarshalRead(r io.Reader, out any) error {
	d := toml.NewDecoder(r)

	if t.disallowUnknownFields {
		d.DisallowUnknownFields()
	}
	return d.Decode(out)
}

func (t *tomlCodec) DisallowUnknownFields() {
	t.disallowUnknownFields = true
}

func (t *tomlCodec) SetIndent(indent string) {
	t.indent = indent
}
