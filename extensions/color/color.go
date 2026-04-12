// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package color provides template functions for adding color
// to the command line output.  It automatically detects whether the
// terminal supports color, and it contains conventions-based approaches
// for allowing the user to control whether color is used
package color

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

// Options is the configuration for the color extension and provides the initializer for the
// app initialization pipeline
type Options struct {
	// Features enumerates which features to use
	Features Feature
}

// Feature provides a name for each feature in the extension
type Feature int

// Emoji represents emoji
type Emoji string

const (
	// FlagFeature enables the flag --color={auto|never|always} for enabling color.
	// The flag's value is optional, in which case its value is always
	FlagFeature = Feature(1 << iota)

	// NoFlagFeature enables the flag --no-color for disabling color
	NoFlagFeature

	// ModeFeature specifies that the --color flag accepts an optional value
	// that corresponds to the Mode value, either "auto", "always", or "never"
	// indicating how to handle color.
	ModeFeature

	// TemplateFuncs enables the template funcs feature, which provides template funcs
	// for colors and styles
	TemplateFuncs

	// AllFeatures enables all of the features.  This is the default
	AllFeatures = -1
)

const (
	// Defines provides a context filter for flags defined by this package
	Defines ContextFilter = 0
)

// Emoji constants
const (
	Tada           Emoji = "🎉"
	Fire           Emoji = "🔥"
	Sparkles       Emoji = "✨"
	Exclamation    Emoji = "❗"
	Bulb           Emoji = "💡"
	X              Emoji = "❌"
	HeavyCheckMark Emoji = "✔️"
	Warning        Emoji = "⚠️"
	Play           Emoji = "▶"
)

var (
	helpText = "Controls whether terminal color and styles are used"

	featureMap = cli.FeatureMap[Feature]{
		FlagFeature | NoFlagFeature | ModeFeature: cli.Pipeline(flagWithMode, standaloneNoFlag),
		FlagFeature | ModeFeature:                 cli.ActionFunc(flagWithMode),
		FlagFeature | NoFlagFeature:               cli.ActionFunc(bothFlags),
		ModeFeature:                               cli.ActionFunc(flagWithMode),
		NoFlagFeature:                             cli.ActionFunc(standaloneNoFlag),
		FlagFeature:                               cli.ActionFunc(standaloneFlag),

		TemplateFuncs: RegisterTemplateFuncs(),
	}

	pkgPath = reflect.TypeFor[Emoji]().PkgPath()
	tagged  = cli.Data(SourceAnnotation())

	emojiByName = map[string]Emoji{
		"Tada":           Tada,
		"Fire":           Fire,
		"Sparkles":       Sparkles,
		"Exclamation":    Exclamation,
		"Bulb":           Bulb,
		"X":              X,
		"HeavyCheckMark": HeavyCheckMark,
		"Warning":        Warning,
		"Play":           Play,
	}

	definesImpl = cli.HasData(SourceAnnotation())
)

// ContextFilter provides the context filters in this package
type ContextFilter int

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", pkgPath
}

// RegisterTemplateFuncs sets up the template funcs which can be used
// to activate color and styles.  The common use is to pipe to the format
// function which has the same name as any of the Color and Style constants:
//
//	{{ "Text to make bold" | Bold }}
func RegisterTemplateFuncs(modeopt ...Mode) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		var thunk func() bool

		if len(modeopt) == 0 || modeopt[0] == Auto {
			thunk = func() bool {
				return support.ColorEnabled(c.Stdout)
			}
		} else {
			thunk = func() bool {
				return modeopt[0] == Always
			}
		}

		for k, v := range templateFuncs(thunk) {
			c.RegisterTemplateFunc(k, v)
		}
		return nil
	})
}

// SetMode returns an action that sets the color mode.
// If specified on a flag or argument, it provides the action for a Boolean or Mode value
// that controls whether color is set.  The flag or arg must have Value that is either
// *bool or *Mode.  The initializer sets *Mode if it is unset.
// If the argument modeopt is specified, the value will be used; otherwise, it will be
// obtained from the context.
func SetMode(modeopt ...Mode) cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Value: new(Mode),
		},
		bind.Call2(setMode, bind.Context(), bind.Exact(untyped(modeopt)...)),
	)
}

func untyped[T any](values []T) []any {
	res := make([]any, len(values))
	for i := range values {
		res[i] = values[i]
	}
	return res
}

func setMode(c *cli.Context, f any) error {
	switch fb := f.(type) {
	case bool:
		c.SetColor(fb)
		return nil
	case Mode:
		if fb == Auto {
			c.AutodetectColor()
		} else {
			c.SetColor(fb == Always)
		}
		return nil
	case *Mode:
		return setMode(c, *fb)
	}

	return fmt.Errorf("not supported type %T", f)
}

func flagWithMode(c *cli.Context) error {
	return c.AddFlag(&cli.Flag{
		Name:  "color",
		Value: new(Mode),
		Uses: cli.Pipeline(
			cli.OptionalValue(Always),
			SetMode(),
			tagged,
		),
		Completion: cli.CompletionValues("auto", "always", "never"),
		HelpText:   helpText,
	})
}

func bothFlags(c *cli.Context) error {
	return c.AddFlag(&cli.Flag{
		Name:    "color",
		Options: cli.No,
		Uses:    booleanFlag(),
	})
}

func standaloneNoFlag(c *cli.Context) error {
	return c.AddFlag(&cli.Flag{
		Name:     "no-color",
		Value:    new(bool),
		Action:   SetMode(Never),
		Uses:     tagged,
		HelpText: "Disable terminal color and styles",
	})
}

func standaloneFlag(c *cli.Context) error {
	return c.AddFlag(&cli.Flag{
		Name: "color",
		Uses: booleanFlag(),
	})
}

func booleanFlag() cli.Action {
	return cli.Pipeline(
		bind.Call2(setFromBoolean, bind.Context(), bind.Bool("")),
		tagged,
		cli.HelpText(helpText),
	)
}

// Execute implements the action interface
func (o Options) Execute(c context.Context) error {
	return o.Pipeline().Execute(c)
}

// Pipeline converts the value into a pipeline
func (o Options) Pipeline() cli.Action {
	return o.Features.Pipeline()
}

// Pipeline converts the value into a pipeline
func (f Feature) Pipeline() cli.Action {
	if f == 0 {
		f = AllFeatures
	}
	return featureMap.Pipeline(f)
}

func (f ContextFilter) Matches(ctx context.Context) bool {
	if f == Defines {
		return definesImpl.Matches(ctx)
	}
	return false
}

// String produces a textual representation of the context filter
func (f ContextFilter) String() string {
	return "color.DEFINES"
}

// MarshalText provides the textual representation
func (f ContextFilter) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText converts the textual representation
func (f *ContextFilter) UnmarshalText(b []byte) error {
	token := strings.TrimSpace(string(b))
	if token == "color.DEFINES" {
		*f = Defines
		return nil
	}
	return nil
}

// Describe produces a representation of the context filter suitable for use in messages
func (f ContextFilter) Describe() string {
	return "defined in joe-cli/color pkg"
}

var (
	_ encoding.TextMarshaler   = (*ContextFilter)(nil)
	_ encoding.TextUnmarshaler = (*ContextFilter)(nil)
	_ cli.ContextFilter        = (*ContextFilter)(nil)
)

func setFromBoolean(c *cli.Context, f bool) error {
	c.SetColor(f)
	return nil
}

var _ cli.Action = Options{}
