// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package codec provides a model and conventions for marshaling and unmarshaling values to and
// from their encodings.
package codec

import (
	"io"
)

type Codec interface {
	NewDecoder(io.Reader, ...DecodeOption) Decoder
	NewEncoder(io.Writer, ...EncodeOption) Encoder
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

type Decoder interface {
	Decode(v any) error
}

type Encoder interface {
	Encode(v any) error
}

type DecodeOption interface {
	Apply(Decoder)
}

type EncodeOption interface {
	Apply(Encoder)
}
