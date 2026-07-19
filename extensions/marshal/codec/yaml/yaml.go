// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaml provides the YAML codec
package yaml

import (
	"io"
	"strings"

	"github.com/Carbonfrost/joe-cli/extensions/marshal"
	"github.com/Carbonfrost/joe-cli/extensions/marshal/codec"
	goyaml "go.yaml.in/yaml/v3"
)

type yamlCodec struct {
	disallowUnknownFields bool
	indent                int
}

func init() {
	marshal.RegisterCodec(marshal.YAML, NewCodec)
}

// NewCodec creates a codec that supports yaml
func NewCodec() codec.Interface {
	return &yamlCodec{}
}

func (y *yamlCodec) MarshalWrite(w io.Writer, in any) error {
	e := goyaml.NewEncoder(w)
	if y.indent > 0 {
		e.SetIndent(y.indent)
	}
	return e.Encode(in)
}

func (y *yamlCodec) UnmarshalRead(r io.Reader, out any) error {
	e := goyaml.NewDecoder(r)
	if y.disallowUnknownFields {
		e.KnownFields(true)
	}
	return e.Decode(out)
}

func (y *yamlCodec) DisallowUnknownFields() {
	y.disallowUnknownFields = true
}

func (y *yamlCodec) SetIndent(indent string) {
	// This API only supports spaces, so we simply infer
	// from the _length_ of the indent, not its actual symbols
	indent = strings.ReplaceAll(indent, "\t", "    ")
	y.indent = len(indent)
}
