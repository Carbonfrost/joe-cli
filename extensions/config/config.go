// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config provides a configuration system that registers a global
// service similar to the provider extension's context services. The config
// store is based on the Lookup interface and can retrieve values by qualified
// names delimited by periods, using a "dig" algorithm to traverse hierarchical
// names.
package config

import (
	"context"
	"fmt"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
)

type key string

type contextValue interface {
	contextValueSigil()
}

const (
	workspaceKey key = "workspace"
	configKey    key = "config"
)

// The various types that the configuration system supports
const (
	BigFloat   = marshal.BigFloat
	BigInt     = marshal.BigInt
	Bool       = marshal.Bool
	Bytes      = marshal.Bytes
	Duration   = marshal.Duration
	File       = marshal.File
	FileSet    = marshal.FileSet
	Float32    = marshal.Float32
	Float64    = marshal.Float64
	Int        = marshal.Int
	Int16      = marshal.Int16
	Int32      = marshal.Int32
	Int64      = marshal.Int64
	Int8       = marshal.Int8
	IP         = marshal.IP
	List       = marshal.List
	Map        = marshal.Map
	NameValue  = marshal.NameValue
	NameValues = marshal.NameValues
	Regexp     = marshal.Regexp
	String     = marshal.String
	Uint       = marshal.Uint
	Uint16     = marshal.Uint16
	Uint32     = marshal.Uint32
	Uint64     = marshal.Uint64
	Uint8      = marshal.Uint8
	URL        = marshal.URL
)

var (
	defaultOpts = []Option{
		WithDefaultAction(),
	}
)

// FromContext retrieves the configuration store from the context
func FromContext(ctx context.Context) *Config {
	return fromContext[*Config](ctx)
}

func fromContext[T contextValue](ctx context.Context) T {
	res, err := tryFromContext[T](ctx)
	if err != nil {
		panic(err)
	}
	return res
}

func tryFromContext[T contextValue](ctx context.Context) (T, error) {
	var zero T
	key := keyFor(zero)
	res, ok := ctx.Value(key).(T)
	if ok {
		return res, nil
	}
	return zero, fmt.Errorf("expected %s value not present in context", key)
}

func keyFor(v contextValue) key {
	switch v.(type) {
	case *Workspace:
		return workspaceKey
	case *Config:
		return configKey
	default:
		panic(fmt.Errorf("unexpected type for context: %T", v))
	}
}

// Option defines an option for initialization of the configuration.
type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) {
	f(c)
}

// Config provides the configuration system, typically retrieved from the context.
type Config struct {
	// Action specifies the action which defines the action to run when this value
	// is added to a pipeline. Typically, this is an initializer set via WithDefaultAction
	cli.Action

	store Store
}

// Pipeline retrieves the configuration action as a pipeline
func (c *Config) Pipeline() cli.Action {
	return cli.Pipeline(c.Action)
}

// New creates a new configuration within the context
func New(opts ...Option) *Config {
	c := new(Config)
	c.Apply(defaultOpts...)
	c.Apply(opts...)
	return c
}

// Store provides the configuration store
func (c *Config) Store() Store {
	return c.store
}

// Apply will apply the given options to the config
func (c *Config) Apply(opts ...Option) {
	for _, o := range opts {
		o.apply(c)
	}
}

// WithStore sets the configuration store to use
func WithStore(s Store) Option {
	return optionFunc(func(c *Config) {
		c.store = s
	})
}

// WithAction sets the Action to use
func WithAction(v cli.Action) Option {
	return optionFunc(func(c *Config) {
		c.Action = v
	})
}

// WithDefaultAction sets the default action
func WithDefaultAction() Option {
	return optionFunc(func(c *Config) {
		c.Action = cli.Pipeline(
			ContextValue(c),
			FlagsAndArgs(),
		)
	})
}

// FlagsAndArgs is an action which provides the default flags to the application.
// Despite its name, which is conventianal, this action provides no args.
func FlagsAndArgs() cli.Action {
	return cli.AddFlags([]*cli.Flag{}...)
}

func (*Config) contextValueSigil() {}
