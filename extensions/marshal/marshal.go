// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package marshal provides a representation of the command line model in a
// serializable data model.
package marshal

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

// App provides a representation of cli.App for use as data
type App struct {
	Name       string         `json:"name"`
	Commands   []Command      `json:"commands,omitempty"`
	Flags      []Flag         `json:"flags,omitempty"`
	Args       []Arg          `json:"args,omitempty"`
	HelpText   string         `json:"helpText,omitempty"`
	ManualText string         `json:"manualText,omitempty"`
	UsageText  string         `json:"usageText,omitempty"`
	Version    string         `json:"version,omitempty"`
	BuildDate  time.Time      `json:"buildDate"`
	Author     string         `json:"author,omitempty"`
	Copyright  string         `json:"copyright,omitempty"`
	License    string         `json:"license,omitempty"`
	Comment    string         `json:"comment,omitempty"`
	Options    Options        `json:"options,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

// Command provides a representation of cli.Command for use as data
type Command struct {
	Name        string         `json:"name"`
	Aliases     []string       `json:"aliases,omitempty"`
	Subcommands []Command      `json:"subcommands,omitempty"`
	Flags       []Flag         `json:"flags,omitempty"`
	Args        []Arg          `json:"args,omitempty"`
	HelpText    string         `json:"helpText,omitempty"`
	ManualText  string         `json:"manualText,omitempty"`
	UsageText   string         `json:"usageText,omitempty"`
	Comment     string         `json:"comment,omitempty"`
	Category    string         `json:"category,omitempty"`
	Options     Options        `json:"options,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

// Flag provides a representation of cli.Flag for use as data
type Flag struct {
	Name        string         `json:"name"`
	Aliases     []string       `json:"aliases,omitempty"`
	EnvVars     []string       `json:"envVars,omitempty"`
	FilePath    string         `json:"filePath,omitempty"`
	HelpText    string         `json:"helpText,omitempty"`
	ManualText  string         `json:"manualText,omitempty"`
	Category    string         `json:"category,omitempty"`
	UsageText   string         `json:"usageText,omitempty"`
	DefaultText string         `json:"defaultText,omitempty"`
	Options     Options        `json:"options,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

// Arg provides a representation of cli.Arg for use as data
type Arg struct {
	Name        string         `json:"name"`
	EnvVars     []string       `json:"envVars,omitempty"`
	FilePath    string         `json:"filePath,omitempty"`
	HelpText    string         `json:"helpText,omitempty"`
	ManualText  string         `json:"manualText,omitempty"`
	Category    string         `json:"category,omitempty"`
	UsageText   string         `json:"usageText,omitempty"`
	DefaultText string         `json:"defaultText,omitempty"`
	Options     Options        `json:"options,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

// Options provides a representation of cli.Option for use as data
type Options = cli.Option

// Type represents a type that can be used in the CLI
type Type interface {
	New() any
	String() string
	MarshalText() ([]byte, error)
}

// BuiltinType identifies the built-in supported types
type BuiltinType int

// The various types that the CLI supports
const (
	UnknownType BuiltinType = iota

	BigFloat
	BigInt
	Bool
	Bytes
	Duration
	File
	FileSet
	Float32
	Float64
	Int
	Int16
	Int32
	Int64
	Int8
	IP
	List
	Map
	NameValue
	NameValues
	Regexp
	String
	Uint
	Uint16
	Uint32
	Uint64
	Uint8
	URL

	maxType
)

var (
	typeStrings = [maxType]string{
		"",
		BigFloat:   "bigfloat",
		BigInt:     "bigint",
		Bool:       "bool",
		Bytes:      "bytes",
		Duration:   "duration",
		File:       "file",
		FileSet:    "fileset",
		Float32:    "float32",
		Float64:    "float64",
		Int:        "int",
		Int16:      "int16",
		Int32:      "int32",
		Int64:      "int64",
		Int8:       "int8",
		IP:         "ip",
		List:       "list",
		Map:        "map",
		NameValue:  "namevalue",
		NameValues: "namevalues",
		Regexp:     "regexp",
		String:     "string",
		Uint:       "uint",
		Uint16:     "uint16",
		Uint32:     "uint32",
		Uint64:     "uint64",
		Uint8:      "uint8",
		URL:        "url",
	}
)

func (t BuiltinType) New() any {
	switch t {
	case NameValues:
		return cli.NameValues()
	case NameValue:
		return new(cli.NameValue)
	case List:
		return cli.List()
	case BigFloat:
		return cli.BigFloat()
	case BigInt:
		return cli.BigInt()
	case Bool:
		return cli.Bool()
	case Float32:
		return cli.Float32()
	case Float64:
		return cli.Float64()
	case Int:
		return cli.Int()
	case Int16:
		return cli.Int16()
	case Int32:
		return cli.Int32()
	case Int64:
		return cli.Int64()
	case Int8:
		return cli.Int8()
	case Map:
		return cli.Map()
	case IP:
		return cli.IP()
	case Regexp:
		return cli.Regexp()
	case String:
		return cli.String()
	case Duration:
		return cli.Duration()
	case Uint:
		return cli.Uint()
	case Uint16:
		return cli.Uint16()
	case Uint32:
		return cli.Uint32()
	case Uint64:
		return cli.Uint64()
	case Uint8:
		return cli.Uint8()
	case URL:
		return cli.URL()
	case File:
		return new(cli.File)
	case FileSet:
		return new(cli.FileSet)
	}
	panic(fmt.Sprintf("unexpected value %q", t))
}

func (t *BuiltinType) Set(arg string) error {
	arg = strings.ToLower(strings.TrimSpace(arg))
	for i, v := range typeStrings {
		if v == arg {
			*t = BuiltinType(i)
			return nil
		}
	}
	return fmt.Errorf("unexpected %q", arg)
}

func (t BuiltinType) String() string {
	return typeStrings[int(t)]
}

// MarshalText provides the textual representation
func (t BuiltinType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText converts the textual representation
func (t *BuiltinType) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, y := range typeStrings {
		if token == y {
			*t = BuiltinType(k)
			return nil
		}
	}
	return nil
}

// New creates a new instance based on the schema
func (s Schema) New() any {
	result := make(map[string]any, len(s))
	for name, typ := range s {
		result[name] = typ.New()
	}
	return result
}

// String returns a string representation of the schema
func (s Schema) String() string {
	if len(s) == 0 {
		return "schema{}"
	}
	var parts []string
	for name, typ := range s {
		parts = append(parts, fmt.Sprintf("%s:%s", name, typ.String()))
	}
	return fmt.Sprintf("schema{%s}", strings.Join(parts, ","))
}

// Schema represents a structural type with named fields
type Schema map[string]Type

// MarshalText provides the textual representation
func (s Schema) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// MarshalJSON provides JSON representation of the schema
func (s Schema) MarshalJSON() ([]byte, error) {
	result := make(map[string]any, len(s))
	for name, typ := range s {
		switch t := typ.(type) {
		case BuiltinType:
			result[name] = t.String()
		case Schema:
			result[name] = t
		default:
			result[name] = typ.String()
		}
	}
	return json.Marshal(result)
}

// UnmarshalJSON parses JSON representation into a schema
func (s *Schema) UnmarshalJSON(b []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*s = make(Schema, len(raw))
	for name, value := range raw {
		switch v := value.(type) {
		case string:
			// Parse as BuiltinType
			var bt BuiltinType
			if err := bt.UnmarshalText([]byte(v)); err != nil {
				return err
			}
			(*s)[name] = bt
		case map[string]any:
			// Parse as nested Schema
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return err
			}
			var nested Schema
			if err := nested.UnmarshalJSON(jsonBytes); err != nil {
				return err
			}
			(*s)[name] = nested
		default:
			return fmt.Errorf("unexpected type for field %q: %T", name, value)
		}
	}
	return nil
}

var _ Type = Schema(nil)
var _ flag.Value = (*BuiltinType)(nil)
var _ Type = BuiltinType(0)
