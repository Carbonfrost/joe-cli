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
	"github.com/Carbonfrost/joe-cli/extensions/marshal"
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
