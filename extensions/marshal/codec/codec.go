// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package codec provides a model and conventions for marshaling and unmarshaling values to and
// from their encodings.
package codec

import (
	"bytes"
	"errors"
	"io"
)

// Interface defines the interface for reading and writing from data
type Interface interface {
	MarshalWrite(w io.Writer, in any) error
	UnmarshalRead(r io.Reader, out any) error
}

// Codec provides a wrapper around the basic interface to facilitate
// common codec operations
type Codec struct {
	Interface
}

// Marshal returns the encoding of v.
func (c Codec) Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	err := c.Interface.MarshalWrite(&buf, v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal parses the encoded data and stores the result in the
// value pointed to by v.
func (c Codec) Unmarshal(data []byte, v any) error {
	return c.Interface.UnmarshalRead(bytes.NewReader(data), v)
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

type commonInterfaceOptioner interface {
	DisallowUnknownFields()
}

// DisallowUnknownFields affects unmarshaling and prevents unknown fields from
// being specified.
func DisallowUnknownFields() Option {
	return booleanOption((commonInterfaceOptioner).DisallowUnknownFields)
}

func booleanOption[C any](fn func(C)) Option {
	return optionFunc(func(i Interface) error {
		c, ok := i.(C)
		if !ok {
			return errors.ErrUnsupported
		}
		fn(c)
		return nil
	})
}

type optionFunc func(i Interface) error

func (o optionFunc) apply(i Interface) error {
	return o(i)
}
