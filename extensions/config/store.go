// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/Carbonfrost/joe-cli"
)

// Store provides configuration storage based on the Lookup interface.
// It retrieves values by qualified names delimited by periods, using
// a "dig" algorithm to traverse hierarchical names.
type Store interface {
	cli.Lookup
	Has(name any) bool
}

// ReloadableStore provides a store which can be reloaded.
type ReloadableStore interface {
	Store

	// Reload will reload the store in place. Generally, clients don't call
	// this method directly. Use Config.Reload which will provide the appropriate
	// services to the context.
	Reload(context.Context) error
}

//counterfeiter:generate . Loader

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

// FromValues creates a store from the specified values
func FromValues(values ...Value) (Store, error) {
	lv := cli.LookupValues{}
	for _, v := range values {
		if v.FromEnv {
			var ok bool
			lv[v.Name], ok = os.LookupEnv(v.Value)
			if !ok {
				return nil, fmt.Errorf("env var not defined %s", v.Value)
			}
		} else {
			lv[v.Name] = v.Value
		}
	}
	return NewStore(lv, func(key string) bool {
		_, ok := lv[key]
		return ok
	}), nil
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

// WithLoader specifies a loader that can be used to generate the store.
func WithLoader(l Loader) Option {
	return optionFunc(func(c *Config) {
		c.store.fn = func(ctx context.Context) (Store, error) {
			return l.Load(ctx)
		}
	})
}

type storeCache struct {
	fn        func(context.Context) (Store, error)
	mu        sync.RWMutex
	store     Store
	once      sync.Once
	lastError error
}

func (c *storeCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = nil
	c.once = sync.Once{}
}

func (c *storeCache) ensureStore(ctx context.Context) Store {
	c.mu.RLock()

	if c.store != nil {
		c.mu.RUnlock()
		return c.store
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.once.Do(func() {
		if c.fn == nil {
			c.store = empty
			return
		}
		store, err := c.fn(ctx)
		if err != nil {
			c.lastError = err
			c.store = nil

		} else {
			c.store = store
		}
	})

	// If store creation failed, use the empty store
	if c.store == nil {
		c.store = empty
	}
	return c.store
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
