// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package codec provides a model and conventions for marshaling and unmarshaling values to and
// from their encodings.
package codec

import (
	"io"
)

// Interface defines the interface for reading and writing from data
type Interface interface {
	MarshalWrite(w io.Writer, in any) error
	UnmarshalRead(r io.Reader, out any) error
}

// Option implements options for codecs
type Option interface {
	apply(Interface) error
}

// WithOptions applies options to the given codec
func WithOptions(i Interface, opts ...Option) (Interface, error) {
	for _, o := range opts {
		err := o.apply(i)
		if err != nil {
			return nil, err
		}
	}
	return i, nil
}
