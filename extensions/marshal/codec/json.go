// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"encoding/json"
	"io"
)

type jsonCodec struct{}

func NewJSONCodec() Interface {
	return &jsonCodec{}
}

func (*jsonCodec) MarshalWrite(w io.Writer, in any) error {
	e := json.NewEncoder(w)
	return e.Encode(in)
}

func (*jsonCodec) UnmarshalRead(r io.Reader, out any) error {
	e := json.NewDecoder(r)
	return e.Decode(out)
}
