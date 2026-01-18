// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package marshal

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/Carbonfrost/joe-cli"
)

// value is a value that can be serialized
type value interface {
	valueSigil()
}

// Option specifies an option for creating marshal values
type Option interface {
	apply(*converter)
}

type converter struct {
	privateData bool
}

type optionFunc func(*converter)

// From creates the schema serialization value for the given Joe value.
// The following types are supported:
//   - *cli.App
//   - *cli.Arg
//   - *cli.Command
//   - *cli.Flag
//
// Any other type of value specified will panic
func From(v any, opts ...Option) any {
	c := new(converter)
	for _, o := range opts {
		o.apply(c)
	}
	return c.fromInternal(v)
}

func (c *converter) fromInternal(v any) value {
	switch t := v.(type) {
	case *cli.App:
		return c.newAppMarshal(t)
	case *cli.Command:
		return c.newCommandMarshal(t)
	case *cli.Flag:
		return c.newFlagMarshal(t)
	case *cli.Arg:
		return c.newArgMarshal(t)
	case value:
		return t
	}
	panic(fmt.Errorf("unsupported type %T", v))
}

func (c *converter) newAppMarshal(v *cli.App) App {
	app := App{
		Name:       v.Name,
		Version:    v.Version,
		BuildDate:  v.BuildDate,
		Author:     v.Author,
		Copyright:  v.Copyright,
		Comment:    v.Comment,
		Options:    v.Options,
		HelpText:   v.HelpText,
		ManualText: v.ManualText,
		UsageText:  v.UsageText,
		License:    v.License,
	}

	cmd, ok := v.Command("")
	if ok && cmd != nil {
		app.Commands = c.commandsMarshal(cmd.Subcommands)
		app.Flags = c.flagsMarshal(cmd.Flags)
		app.Args = c.argsMarshal(cmd.Args)
		app.Data = c.cleanDataMap(v.Data)
	} else {
		app.Commands = c.commandsMarshal(v.Commands)
		app.Flags = c.flagsMarshal(v.Flags)
		app.Args = c.argsMarshal(v.Args)
		app.Data = c.cleanDataMap(v.Data)
	}

	return app
}

func (c *converter) commandsMarshal(v []*cli.Command) []Command {
	res := make([]Command, len(v))
	for i := range v {
		res[i] = c.newCommandMarshal(v[i])
	}
	return res
}

func (c *converter) flagsMarshal(v []*cli.Flag) []Flag {
	res := make([]Flag, len(v))
	for i := range v {
		res[i] = c.newFlagMarshal(v[i])
	}
	return res
}

func (c *converter) argsMarshal(v []*cli.Arg) []Arg {
	res := make([]Arg, len(v))
	for i := range v {
		res[i] = c.newArgMarshal(v[i])
	}
	return res
}

func (c *converter) newCommandMarshal(v *cli.Command) Command {
	return Command{
		Name:        v.Name,
		Subcommands: c.commandsMarshal(v.Subcommands),
		Flags:       c.flagsMarshal(v.Flags),
		Args:        c.argsMarshal(v.Args),
		Aliases:     v.Aliases,
		Category:    v.Category,
		Comment:     v.Comment,
		Data:        c.cleanDataMap(v.Data),
		Options:     v.Options,
		HelpText:    v.HelpText,
		ManualText:  v.ManualText,
		UsageText:   v.UsageText,
	}
}

func (c *converter) newFlagMarshal(v *cli.Flag) Flag {
	return Flag{
		Name:        v.Name,
		Aliases:     v.Aliases,
		HelpText:    v.HelpText,
		ManualText:  v.ManualText,
		UsageText:   v.UsageText,
		EnvVars:     v.EnvVars,
		FilePath:    v.FilePath,
		DefaultText: v.DefaultText,
		Options:     v.Options,
		Category:    v.Category,
		Data:        c.cleanDataMap(v.Data),
	}
}

func (c *converter) newArgMarshal(v *cli.Arg) Arg {
	return Arg{
		Name:        v.Name,
		HelpText:    v.HelpText,
		ManualText:  v.ManualText,
		UsageText:   v.UsageText,
		EnvVars:     v.EnvVars,
		FilePath:    v.FilePath,
		DefaultText: v.DefaultText,
		Options:     v.Options,
		Category:    v.Category,
		Data:        c.cleanDataMap(v.Data),
	}
}

// WithPrivateData provides an option that causes private data, which is
// any data added to a Data map whose key starts with understcore, is included
// in the marshal representation of a target.  By default, private data is excluded
func WithPrivateData() Option {
	return optionFunc(func(c *converter) {
		c.privateData = true
	})
}

func (c *converter) cleanDataMap(in map[string]any) map[string]any {
	return cleanDataMap(in, !c.privateData)
}

func cleanDataMap(in map[string]any, skipPrivate bool) map[string]any {
	result := make(map[string]any, len(in))

	var ok bool
	for k, v := range in {
		if v, ok = cleanDataMapValue(k, v, skipPrivate); ok {
			result[k] = v
		}

	}
	return result
}

func cleanDataMapValue(k string, v any, skipPrivate bool) (any, bool) {
	if skipPrivate && strings.HasPrefix(k, "_") {
		return nil, false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Func,
		reflect.Chan,
		reflect.UnsafePointer,
		reflect.Complex64,
		reflect.Complex128:
		return nil, false

	case reflect.Map:
		rt := rv.Type()

		if rt.Key().Kind() == reflect.String {
			return cleanDataMap(rv.Interface().(map[string]any), skipPrivate), true
		}

		return nil, false

	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, false
		}

		// TODO This should look at structs, pointer to structs
	}
	return v, true
}

func (App) valueSigil()     {}
func (Flag) valueSigil()    {}
func (Arg) valueSigil()     {}
func (Command) valueSigil() {}

func (f optionFunc) apply(c *converter) {
	f(c)
}
