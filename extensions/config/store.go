// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"fmt"

	"github.com/Carbonfrost/joe-cli"
)

// Store provides configuration storage based on the Lookup interface.
// It retrieves values by qualified names delimited by periods, using
// a "dig" algorithm to traverse hierarchical names.
type Store interface {
	cli.Lookup
	Has(name any) bool
}

// Loader loads the configuration system
type Loader interface {
	Load(context.Context) (Store, error)
}

// NewStore provides a store that wraps a lookup
func NewStore(base cli.Lookup, has func(string) bool) Store {
	return wrapperStore{
		Lookup: base, has: has,
	}
}

type wrapperStore struct {
	cli.Lookup
	has func(string) bool
}

var empty = wrapperStore{
	Lookup: cli.EmptyLookup,
	has: func(string) bool {
		return false
	},
}

func (w wrapperStore) Has(v any) bool {
	return w.has(nameToString(v))
}

func nameToString(name any) string {
	switch v := name.(type) {
	case rune:
		return string(v)
	case string:
		return v
	case nil:
		return ""
	case *cli.Arg:
		return v.Name
	case *cli.Flag:
		return v.Name
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}
}
