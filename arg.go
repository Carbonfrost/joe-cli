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

	option *optionWrapper
	flags  internalFlags
}

type argSynopsis struct {
	value string
	multi bool
}

func (a *Arg) Occurrences() int {
	return a.option.Count()
}

func (a *Arg) Seen() bool {
	return a.option.Count() > 0
}

func (a *Arg) Set(arg string) error {
	a.option.count += 1
	err := a.option.Value().Set(arg, a.option)
	return err
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

func (a *Arg) applyToSet(s *Set) {
	a.option = s.defineArg(a.Name, a.value())
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
