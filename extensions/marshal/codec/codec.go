// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package codec provides a model and conventions for marshaling and unmarshaling values to and
// from their encodings.
package codec

import (
	"io"
)

type Interface interface {
	MarshalWrite(w io.Writer, in any) error
	UnmarshalRead(r io.Reader, out any) error
}
