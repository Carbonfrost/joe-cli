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

// Interface converts the given string key into its variable expansion
type Interface interface {
	Expand(key string) any
}

// Func converts the given string key into its variable expansion
type Func func(string) any

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

func Env() Interface {
	return Func(func(s string) any {
		result, ok := os.LookupEnv(s)
		if ok {
			return result
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

type Map map[string]any

func (m Map) Expand(k string) any {
	v, ok := m[k]
	if !ok {
		return nil
	}
	return v
}

func Colors() Interface {
	return Func(func(k string) any {
		if a, ok := colors[k]; ok {
			return fmt.Sprintf("\x1b[%dm", a)
		}
		return nil
	})
}

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

func Unknown() Interface {
	return Func(func(s string) any {
		return ErrUnknownToken(s)
	})
}

// Reflect provides a simple expander around a given value using
// reflection
func Reflect(v any) Interface {
	if v == nil {
		return Func(func(string) any {
			return nil
		})
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

type ErrUnknownToken string

func (e ErrUnknownToken) Error() string {
	return fmt.Sprintf("unknown: %s", string(e))
}

var _ error = (ErrUnknownToken)("")
