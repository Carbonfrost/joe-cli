package cli

import (
	"fmt"
)

type Arg struct {
	Name        string
	EnvVars     []string
	FilePath    string
	HelpText    string
	UsageText   string
	Value       interface{}
	DefaultText string
	NArg        int
	Options     Option

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// Action executes if the flag was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action interface{}

	option *internalOption
	flags  internalFlags
}

type argSynopsis struct {
	value string
	multi bool
}

// Args provides a simple initializer for positional arguments.  You specify each argument name and value
// in order to this function.    It generates the corresponding list of required positional arguments.
// A panic occurs when this function is not called properly: when a name is blank, when names and values
// are not arranged in pairs, or when a supported type of value is used.
func Args(namevalue ...interface{}) []*Arg {
	if len(namevalue)%2 != 0 {
		panic("unexpected number of arguments")
	}
	res := make([]*Arg, 0, len(namevalue)/2)
	for i := 0; i < len(namevalue); i += 2 {
		res = append(res, &Arg{
			Name:  namevalue[i].(string),
			Value: namevalue[i+1],
		})
	}
	return res
}

func (a *Arg) Occurrences() int {
	return a.option.Count()
}

func (a *Arg) Seen() bool {
	return a.option.Count() > 0
}

func (a *Arg) Set(arg string) error {
	return a.option.Value().Set(arg, a.option)
}

func (a *Arg) SetHidden() {
	a.flags |= internalFlagHidden
}

func (a *Arg) SetRequired() {
	a.flags |= internalFlagRequired
}

// Synopsis contains the value placeholder
func (a *Arg) Synopsis() string {
	return textUsage.arg(a.newSynopsis())
}

func (a *Arg) newSynopsis() *argSynopsis {
	usage := a.UsageText
	if usage == "" {
		usage = fmt.Sprintf("<%s>", a.Name)
	}
	return &argSynopsis{
		value: usage,
		multi: a.NArg < 0 || a.NArg > 1,
	}
}

func (a *Arg) internalFlags() internalFlags {
	return a.flags
}

func (a *Arg) setInternalFlags(i internalFlags) {
	a.flags |= i
}

func (a *Arg) options() Option {
	return a.Options
}

func (a *Arg) wrapAction(fn func(ActionHandler) ActionFunc) {
	a.Action = fn(Action(a.Action))
}

func (a *Arg) applyToSet(s *set) {
	a.option = s.defineArg(a.Name, a.value(), a.NArg)
}

func (a *Arg) action() ActionHandler {
	return Action(a.Action)
}

func (a *Arg) before() ActionHandler {
	return Action(a.Before)
}

func (a *Arg) name() string {
	return a.Name
}

func (a *Arg) envVars() []string {
	return a.EnvVars
}

func (a *Arg) filePath() string {
	return a.FilePath
}

func (a *Arg) helpText() string {
	return a.HelpText
}

func (a *Arg) value() interface{} {
	a.Value = ensureDestination(a.Value, a.NArg)
	return a.Value
}

func findArgByName(items []*Arg, name string) (*Arg, bool) {
	for _, sub := range items {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}
