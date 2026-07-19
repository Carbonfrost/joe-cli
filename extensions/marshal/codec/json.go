// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"encoding/json"
	"io"
)

type jsonCodec struct {
	disallowUnknownFields bool
	escapeHTML            bool
	indent                string
}

// NewJSONCodec creates a codec to support JSON
func NewJSONCodec() Interface {
	return &jsonCodec{}
}

func (j *jsonCodec) MarshalWrite(w io.Writer, in any) error {
	e := json.NewEncoder(w)
	if j.indent != "" {
		e.SetIndent("", j.indent)
	}
	e.SetEscapeHTML(j.escapeHTML)
	return e.Encode(in)
}

func (j *jsonCodec) UnmarshalRead(r io.Reader, out any) error {
	e := json.NewDecoder(r)
	if j.disallowUnknownFields {
		e.DisallowUnknownFields()
	}
	return e.Decode(out)
}

func (j *jsonCodec) DisallowUnknownFields() {
	j.disallowUnknownFields = true
}

func (j *jsonCodec) EscapeHTML() {
	j.escapeHTML = true
}

func (j *jsonCodec) SetIndent(indent string) {
	j.indent = indent
}

var _ escapeHTMLInterfaceOptioner = (*jsonCodec)(nil)
