package marshal

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
)

// value is a value that can be serialized
type value interface {
	valueSigil()
}

// From creates the schema serialization value for the given Joe value.
// The following types are supported:
//   - *cli.App
//   - *cli.Arg
//   - *cli.Command
//   - *cli.Flag
//
// Any other type of value specified will panic
func From(v any) any {
	return fromInternal(v)
}

func fromInternal(v any) value {
	switch t := v.(type) {
	case *cli.App:
		return newAppMarshal(t)
	case *cli.Command:
		return newCommandMarshal(t)
	case *cli.Flag:
		return newFlagMarshal(t)
	case *cli.Arg:
		return newArgMarshal(t)
	case value:
		return t
	}
	panic(fmt.Errorf("unsupported type %T", v))
}

func newAppMarshal(v *cli.App) App {
	return App{
		Name:       v.Name,
		Version:    v.Version,
		BuildDate:  v.BuildDate,
		Author:     v.Author,
		Copyright:  v.Copyright,
		Comment:    v.Comment,
		Commands:   commandsMarshal(v.Commands),
		Flags:      flagsMarshal(v.Flags),
		Args:       argsMarshal(v.Args),
		Data:       v.Data,
		Options:    v.Options,
		HelpText:   v.HelpText,
		ManualText: v.ManualText,
		UsageText:  v.UsageText,
		License:    v.License,
	}
}

func commandsMarshal(v []*cli.Command) []Command {
	res := make([]Command, len(v))
	for i := range v {
		res[i] = newCommandMarshal(v[i])
	}
	return res
}

func flagsMarshal(v []*cli.Flag) []Flag {
	res := make([]Flag, len(v))
	for i := range v {
		res[i] = newFlagMarshal(v[i])
	}
	return res
}

func argsMarshal(v []*cli.Arg) []Arg {
	res := make([]Arg, len(v))
	for i := range v {
		res[i] = newArgMarshal(v[i])
	}
	return res
}

func newCommandMarshal(v *cli.Command) Command {
	return Command{
		Name:        v.Name,
		Subcommands: commandsMarshal(v.Subcommands),
		Flags:       flagsMarshal(v.Flags),
		Args:        argsMarshal(v.Args),
		Aliases:     v.Aliases,
		Category:    v.Category,
		Comment:     v.Comment,
		Data:        v.Data,
		Options:     v.Options,
		HelpText:    v.HelpText,
		ManualText:  v.ManualText,
		UsageText:   v.UsageText,
	}
}

func newFlagMarshal(v *cli.Flag) Flag {
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
		Data:        v.Data,
	}
}

func newArgMarshal(v *cli.Arg) Arg {
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
		Data:        v.Data,
	}
}

func (App) valueSigil()     {}
func (Flag) valueSigil()    {}
func (Arg) valueSigil()     {}
func (Command) valueSigil() {}
