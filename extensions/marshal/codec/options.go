// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"errors"
)

// Options represents all currently available codec options as data.
// It implements Option and can be decomposed into a list of individual options.
type Options struct {
	DisallowUnknownFields bool `mapstructure:"disallow_unknown_fields"`
}

// List returns the individual Option values corresponding to the fields set on o.
func (o Options) List() []Option {
	var opts []Option

	if o.DisallowUnknownFields {
		opts = append(opts, DisallowUnknownFields())
	}
	return opts
}

func (o Options) apply(i Interface) error {
	for _, opt := range o.List() {
		err := opt.apply(i)
		if err != nil && !errors.Is(err, errors.ErrUnsupported) {
			return err
		}
	}
	return nil
}
