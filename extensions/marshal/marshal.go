// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package marshal provides a representation of the command line model in a
// serializable data model.
package marshal

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

type App struct {
	Name       string         `json:"name"`
	Commands   []Command      `json:"commands,omitempty"`
	Flags      []Flag         `json:"flags,omitempty"`
	Args       []Arg          `json:"args,omitempty"`
	HelpText   string         `json:"helpText,omitempty"`
	ManualText string         `json:"manualText,omitempty"`
	UsageText  string         `json:"usageText,omitempty"`
	Version    string         `json:"version,omitempty"`
	BuildDate  time.Time      `json:"buildDate,omitempty"`
	Author     string         `json:"author,omitempty"`
	Copyright  string         `json:"copyright,omitempty"`
	License    string         `json:"license,omitempty"`
	Comment    string         `json:"comment,omitempty"`
	Options    Option         `json:"options,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
}

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
	Options     Option         `json:"options,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

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
	Options     Option         `json:"options,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

type Arg struct {
	Name        string         `json:"name"`
	EnvVars     []string       `json:"envVars,omitempty"`
	FilePath    string         `json:"filePath,omitempty"`
	HelpText    string         `json:"helpText,omitempty"`
	ManualText  string         `json:"manualText,omitempty"`
	Category    string         `json:"category,omitempty"`
	UsageText   string         `json:"usageText,omitempty"`
	DefaultText string         `json:"defaultText,omitempty"`
	Options     Option         `json:"options,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

type Option = cli.Option

// Type identifies the built-in supported types
type Type int

const (
	UnknownType Type = iota

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

func (t Type) New() any {
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

func (t *Type) Set(arg string) error {
	arg = strings.ToLower(strings.TrimSpace(arg))
	for i, v := range typeStrings {
		if v == arg {
			*t = Type(i)
			return nil
		}
	}
	return fmt.Errorf("unexpected %q", arg)
}

func (t Type) String() string {
	return typeStrings[int(t)]
}

// MarshalText provides the textual representation
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText converts the textual representation
func (t *Type) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	for k, y := range typeStrings {
		if token == y {
			*t = Type(k)
			return nil
		}
	}
	return nil
}

var _ flag.Value = (*Type)(nil)
