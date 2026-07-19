// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package expander provides an extension for representing expandable
// expressions in command line arguments.  The main use case is
// for dynamically evaluating these expressions with values that
// occur during the execution of the evaluators.
package expander

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	// vt100 ansi codes
	colors = map[string]int{
		"default":       39,
		"black":         30,
		"red":           31,
		"green":         32,
		"yellow":        33,
		"blue":          34,
		"magenta":       35,
		"cyan":          36,
		"gray":          37,
		"darkGray":      90,
		"brightRed":     91,
		"brightGreen":   92,
		"brightYellow":  93,
		"brightBlue":    94,
		"brightMagenta": 95,
		"brightCyan":    96,
		"white":         97,

		"default.bg": 49,
		"black.bg":   40,
		"red.bg":     41,
		"green.bg":   42,
		"yellow.bg":  43,
		"blue.bg":    44,
		"magenta.bg": 45,
		"cyan.bg":    46,
		"gray.bg":    47,

		"reset":           0,
		"bold":            1,
		"faint":           2,
		"italic":          3,
		"underline":       4,
		"slow":            5,
		"fast":            6,
		"reverse":         7,
		"erase":           8,
		"strikethrough":   9,
		"doubleUnderline": 21,

		"bold.off":            22,
		"italic.off":          23,
		"underline.off":       24,
		"doubleUnderline.off": 24, // same
		"slow.off":            25,
		"fast.off":            26,
		"reverse.off":         27,
		"erase.off":           28,
		"strikethrough.off":   29,
	}
)

// Nil provides an expander which always provides nil
var Nil = Func(nilImpl)

// Interface converts the given string key into its variable expansion
type Interface interface {
	Expand(key string) any
}

// Func converts the given string key into its variable expansion
type Func func(string) any

// Expand implements Interface by invoking f with key.
func (f Func) Expand(key string) any {
	return f(key)
}

// Prefix provides an expander which looks for and cuts a given prefix
// and delegates the result to the underlying expander
func Prefix(p string, e Interface) Interface {
	prefix := p + "."
	return Func(func(k string) any {
		if name, ok := strings.CutPrefix(k, prefix); ok {
			return e.Expand(name)
		}
		return nil
	})
}

// Env provides an expander which looks up environment variables by name,
// returning nil when the variable is not set.
func Env() Interface {
	return Func(func(s string) any {
		result, ok := os.LookupEnv(s)
		if ok {
			return result
		}
		return nil
	})
}

// Runtime provides an expander for Go runtime variables: numCPU, os, arch, version.
// Intended for use with Prefix("go", Runtime()).
func Runtime() Interface {
	return Func(func(k string) any {
		switch k {
		case "numCPU":
			return runtime.NumCPU()
		case "os":
			return runtime.GOOS
		case "arch":
			return runtime.GOARCH
		case "version":
			return runtime.Version()
		}
		return nil
	})
}

// Time obtains an expander for the specified time
func Time(t time.Time) Interface {
	return Func(func(k string) any {
		switch k {
		case "timestamp", "unix":
			return t.Unix()
		case "timestampNanos", "timestampNano", "unixNanos", "unixNano":
			return t.UnixNano()
		case "year":
			return t.Year()
		case "yearDay":
			return t.YearDay()
		case "month":
			return t.Month()
		case "hour12":
			return (t.Hour() + 12) % 12
		case "hour":
			return t.Hour()
		case "day":
			return t.Day()
		case "minute":
			return t.Minute()
		case "second":
			return t.Second()
		case "nano", "nanosecond":
			return t.Nanosecond()
		case "weekday":
			return t.Weekday()
		case "zone":
			z, _ := t.Zone()
			return z
		case "zoneOffset":
			_, offset := t.Zone()
			return offset
		}
		return nil
	})
}

// Map provides an expander backed by a map, resolving keys directly to
// their corresponding values.
type Map map[string]any

// Expand returns the value stored under k, or nil when k is absent.
func (m Map) Expand(k string) any {
	v, ok := m[k]
	if !ok {
		return nil
	}
	return v
}

// ExpandSlice creates an expander which uses the underlying
// slice as its input. Keys resolve as the index into the slice, including
// supporting negative indexes to reference from the end of the slice.
func ExpandSlice[T any](slice []T) Interface {
	return Func(func(key string) any {
		i, err := strconv.Atoi(key)
		if err != nil {
			return nil
		}

		if i < 0 {
			i += len(slice)
		}

		if i < 0 || i >= len(slice) {
			return nil
		}

		return slice[i]
	})
}

// Colors provides an expander which resolves color and style names to their
// VT100 ANSI escape sequences, returning nil for unrecognized names.
func Colors() Interface {
	return Func(func(k string) any {
		if a, ok := colors[k]; ok {
			return fmt.Sprintf("\x1b[%dm", a)
		}
		return nil
	})
}

// Compose combines the given expanders into a single expander, returning the
// first non-nil expansion produced by them in order.
func Compose(expanders ...Interface) Interface {
	return Func(func(k string) any {
		for _, x := range expanders {
			v := x.Expand(k)
			if v != nil {
				return v
			}
		}
		return nil
	})
}

// Unknown provides an expander which resolves every key to an ErrUnknownToken
// error. It is typically composed last so that unrecognized keys are reported
// rather than silently expanding to nil.
func Unknown() Interface {
	return Func(func(s string) any {
		return ErrUnknownToken(s)
	})
}

// Reflect provides a simple expander around a given value using
// reflection
func Reflect(v any) Interface {
	if v == nil {
		return Nil
	}
	return &reflectExpander{
		reflect.ValueOf(v),
	}
}

type reflectExpander struct {
	v reflect.Value
}

func (r *reflectExpander) Expand(k string) any {
	v := r.v
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	field := v.FieldByNameFunc(func(name string) bool {
		return strings.EqualFold(name, k)
	})

	if !field.IsValid() {
		return nil
	}
	return field.Interface()
}

// ErrUnknownToken is an error reporting a key that no expander recognized.
// Its string value is the unrecognized key.
type ErrUnknownToken string

// Error implements the error interface.
func (e ErrUnknownToken) Error() string {
	return fmt.Sprintf("unknown: %s", string(e))
}

func nilImpl(string) any {
	return nil
}

var _ error = (ErrUnknownToken)("")
