package cli

import (
	"github.com/pborman/getopt/v2"
)

// Flag represents a command line flag.  The only required attribute that must be set is Name.
// By default, the type of a flag is string; however, to use a more specific type you must
// either specify a pointer to a variable or use the built-in constants that identify the type to
// use:
//
//   &Flag{
//     Name: "age",
//     Value: &age, // var age int -- defined somewhere in scope
//   }
//
//   &Flag{
//     Name: "age",
//     Value: cli.Int(), // also sets int
//   }
//
// The corresponding, typed method to access the value of the flag by name is available from the Context.
// In this case, you can  obtain value of the --age=21 flag using Context.Int("flag"), which may be
// necessary when you don't use your own variable.
//
// A pointer to any of the following types is possible for Value: string, bool,
// int (and any variant of all sizes and/or unsigned), float32, float64, []string, time.Duration,
// and any implementer of the Value interface.
//
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

func names(name string) []string {
	return []string{name}
}

// flagName gets the long and short name for getopt given the name specified in the flag
func flagName(name string) (string, rune) {
	if len(name) == 1 {
		return "", []rune(name)[0]
	} else {
		return name, 0
	}
}
