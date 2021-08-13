package cli

import (
	"github.com/pborman/getopt/v2"
)

type Flag struct {
	Name        string
	Aliases     []string
	HelpText    string
	UsageText   string
	EnvVars     []string
	FilePath    string
	Required    bool
	Hidden      bool
	Value       interface{}
	DefaultText string

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// Action executes if the flag was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action interface{}
	option getopt.Option
}

type Arg struct {
	Name        string
	EnvVars     []string
	FilePath    string
	HelpText    string
	UsageText   string
	Required    bool
	Hidden      bool
	Value       interface{}
	DefaultText string

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// Action executes if the flag was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action interface{}

	internal *generic
	count    int
}

type option interface {
	Occurrences() int
	Seen() bool
	before() ActionHandler
	action() ActionHandler
}

type optionWrapper struct {
	getopt.Option

	arg *Arg
}

func (c *Context) String(name string) string {
	return c.Value(name).(string)
}

func (f *Flag) applyToSet(s *getopt.Set) {
	f.Value = ensureDestination(f.Value)
	for _, name := range f.Names() {
		long, short := flagName(name)
		f.option = s.FlagLong(f.Value, long, short, f.HelpText, name)
	}
}

func (f *Flag) Seen() bool {
	return f.option.Seen()
}

func (f *Flag) Occurrences() int {
	return f.option.Count()
}

func (f *Flag) Names() []string {
	return names(f.Name)
}

func (f *Flag) action() ActionHandler {
	return Action(f.Action)
}

func (f *Flag) before() ActionHandler {
	return Action(f.Before)
}

func (a *Arg) Occurrences() int {
	return a.count
}

func (a *Arg) Seen() bool {
	return a.count > 0
}

func (a *Arg) Set(arg string) error {
	a.Value = ensureDestination(a.Value)
	if a.internal == nil {
		a.internal = wrapGeneric(a.Value)
	}

	a.count = a.count + 1
	return a.internal.Set(arg, optionWrapper{arg: a})
}

func (a *Arg) action() ActionHandler {
	return Action(a.Action)
}

func (a *Arg) before() ActionHandler {
	return Action(a.Before)
}

func (o optionWrapper) Count() int {
	return o.arg.count
}

func ensureDestination(dest interface{}) interface{} {
	if dest == nil {
		// Default to using a string if it wasn't set
		var p string
		return &p
	}
	return dest
}
