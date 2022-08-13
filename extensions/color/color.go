// Package color provides template functions for adding color
// to the command line output.  It automatically detects whether the
// terminal supports color, and it contains conventions-based approaches
// for allowing the user to control whether color is used
package color

import (
	"github.com/Carbonfrost/joe-cli"
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
	flagWithMode = cli.Pipeline(
		cli.AddFlag(&cli.Flag{
			Name:  "color",
			Value: new(Mode),
			Uses: cli.Pipeline(
				cli.OptionalValue(Always),
				SetMode(),
				tagged,
			),
			Completion: cli.CompletionValues("auto", "always", "never"),
			HelpText:   helpText,
		}),
	)

	helpText = "Controls whether terminal color and styles are used"

	featureMap = cli.FeatureMap[Feature]{
		FlagFeature | NoFlagFeature | ModeFeature: cli.Pipeline(flagWithMode, standaloneNoFlag),
		FlagFeature | ModeFeature:                 flagWithMode,
		FlagFeature | NoFlagFeature:               cli.ActionFunc(bothFlags),
		ModeFeature:                               flagWithMode,
		NoFlagFeature:                             cli.ActionFunc(standaloneNoFlag),
		FlagFeature:                               cli.ActionFunc(standaloneFlag),

		TemplateFuncs: RegisterTemplateFuncs(),
	}

	tagged = cli.Data(SourceAnnotation())
)

// SourceAnnotation gets the name and value of the annotation added to the Data
// of all flags that are initialized from this package
func SourceAnnotation() (string, string) {
	return "Source", "joe-cli/color"
}

// RegisterTemplateFuncs sets up the template funcs which can be used
// to activate color and styles.  The common use is to pipe to the format
// function which has the same name as any of the Color and Style constants:
//
//	{{ "Text to make bold" | Bold }}
func RegisterTemplateFuncs() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		for k, v := range templateFuncs(c) {
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
	switch len(modeopt) {
	case 0:
		return cli.Setup{
			Uses: cli.Prototype{
				Value: new(Mode),
			},
			Action: func(c *cli.Context) error {
				switch v := c.Value("").(type) {
				case bool:
					c.SetColor(v)
					return nil
				case *Mode:
					return c.Do(SetMode(*v))
				default:
					c.SetColor(true)
					return nil
				}
			},
		}
	case 1:
		switch modeopt[0] {
		case Always:
			return cli.SetColor(true)
		case Never:
			return cli.SetColor(false)
		case Auto:
			fallthrough
		default:
			return cli.AutodetectColor()
		}
	default:
		panic("expected 0 or 1 argument")
	}
}

func bothFlags(c *cli.Context) error {
	return c.AddFlag(&cli.Flag{
		Name:     "color",
		Options:  cli.No,
		Value:    new(bool),
		Action:   setFromBoolean,
		Uses:     tagged,
		HelpText: helpText,
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
		Name:     "color",
		Value:    new(bool),
		Action:   setFromBoolean,
		Uses:     tagged,
		HelpText: helpText,
	})
}

func (o Options) Execute(c *cli.Context) error {
	return c.Do(o.Features.Pipeline())
}

func (f Feature) Pipeline() cli.Action {
	if f == 0 {
		f = AllFeatures
	}
	return featureMap.Pipeline(f)
}

func setFromBoolean(c *cli.Context) error {
	c.SetColor(c.Bool(""))
	return nil
}

func emojiByName(name string) Emoji {
	switch name {
	case "Tada":
		return Tada
	case "Fire":
		return Fire
	case "Sparkles":
		return Sparkles
	case "Exclamation":
		return Exclamation
	case "Bulb":
		return Bulb
	case "X":
		return X
	case "HeavyCheckMark":
		return HeavyCheckMark
	case "Warning":
		return Warning
	case "Play":
		return Play
	}
	return ""
}

var _ cli.Action = Options{}
