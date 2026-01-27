// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"encoding/json"
	"io"
)

type jsonCodec struct{}

func NewJSONCodec() Codec {
	return &jsonCodec{}
}

func (*jsonCodec) NewDecoder(r io.Reader, _ ...DecodeOption) Decoder {
	return json.NewDecoder(r)
}

func (*jsonCodec) NewEncoder(w io.Writer, _ ...EncodeOption) Encoder {
	return json.NewEncoder(w)
}

func (*jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (*jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
