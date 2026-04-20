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
}

func NewJSONCodec() Interface {
	return &jsonCodec{}
}

func (*jsonCodec) MarshalWrite(w io.Writer, in any) error {
	e := json.NewEncoder(w)
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
