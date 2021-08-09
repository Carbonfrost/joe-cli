package cli

import (
	"github.com/pborman/getopt/v2"
)

type Flag struct {
	Name        string
	Aliases     []string
	HelpText    string
	EnvVars     []string
	FilePath    string
	Required    bool
	Hidden      bool
	Value       interface{}
	DefaultText string
	Action      ActionHandler
	option      getopt.Option
}

type Arg struct {
	Name        string
	EnvVars     []string
	FilePath    string
	Required    bool
	Hidden      bool
	Value       interface{}
	DefaultText string
	Action      ActionHandler

	internal *generic
	count    int
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

func (f *Flag) Names() []string {
	return names(f.Name)
}

func (a *Arg) Occurrences() int {
	return a.count
}

func (a *Arg) Set(arg string) error {
	a.Value = ensureDestination(a.Value)
	if a.internal == nil {
		a.internal = wrapGeneric(a.Value)
	}

	a.count = a.count + 1
	return a.internal.Set(arg, optionWrapper{arg: a})
}

func (o optionWrapper) Count() int {
	return o.arg.count
}

func ensureDestination(dest interface{}) interface{}{
	if dest == nil {
		// Default to using a string if it wasn't set
		var p string
		return &p
	}
	return dest
}
