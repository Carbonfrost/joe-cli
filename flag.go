package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Name string

	// Aliases provides a list of alternative names for the flag.  In general, Name should
	// be used for the long name of the flag, and Aliases should contain the short name.
	// If there are additional names for compatibility reasons, they should be included
	// with Aliases but listed after the preferred names. Note that only one short name
	// and one long name is displayed on help screens by default.
	Aliases []string

	// HelpText contains text which briefly describes the usage of the flag.  If it contains
	// placeholders in the form {PLACEHOLDER}, then these name the purpose of the flag's
	// value.  If a flag has multiple values, then placeholders can also specify the index of
	// the corresponding value using the syntax {0:PLACEHOLDER}; otherwise, the order is
	// inferred start to end.
	// For style, generally the usage text should be limited to about 40 characters.
	// Sentence case is recommended for the usage text.    Uppercase is recommended for the
	// text of placeholders.  The placeholder is used in the synoposis for the flag as well
	// as error messages.
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
	NArg        int

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
	Set(string) error
	name() string
	envVars() []string
	filePath() string
	before() ActionHandler
	action() ActionHandler
}

type optionWrapper struct {
	getopt.Option

	arg *Arg
}

type flagSynopsis struct {
	short string
	long  string
	sep   string
	value string
}

func (f *Flag) applyToSet(s *getopt.Set) {
	f.Value = ensureDestination(f.Value)
	for _, name := range f.Names() {
		long, short := flagName(name)
		f.option = s.FlagLong(f.Value, long, short, f.HelpText, name)
	}
}

// Synopsis contains the name of the flag, its aliases, and the value placeholder.  The text of synopsis
// is inferred from the HelpText.  Up to one short and one long name will be used.
func (f *Flag) Synopsis() string {
	return f.newSynopsis().formatString(false)
}

func (f *Flag) newSynopsis() *flagSynopsis {
	short := f.canonicalName(true)
	long := f.canonicalName(false)
	sep := "="

	if len(long) == 0 {
		sep = " "
	}

	return &flagSynopsis{
		short: short,
		long:  long,
		sep:   sep,
		value: valueSynopsis(f),
	}
}

func (f *flagSynopsis) formatString(hideAlternates bool) string {
	sepIfNeeded := ""
	if len(f.value) > 0 {
		sepIfNeeded = f.sep
	}
	return f.names(hideAlternates) + sepIfNeeded + f.value
}

func (f *flagSynopsis) names(hideAlternates bool) string {
	if len(f.long) == 0 {
		return fmt.Sprintf("-%s", f.short)
	}
	if len(f.short) == 0 {
		return fmt.Sprintf("--%s", f.long)
	}
	if hideAlternates {
		return fmt.Sprintf("--%s", f.long)
	}
	return fmt.Sprintf("-%s, --%s", f.short, f.long)
}

func valueSynopsis(f *Flag) string {
	usage := strings.Join(parseUsage(f.HelpText).Placeholders(), " ")
	if len(usage) > 0 {
		return usage
	}
	switch f.Value.(type) {
	case *bool:
		return ""
	case *int, *int8, *int16, *int32, *int64:
		return "NUMBER"
	case *uint, *uint8, *uint16, *uint32, *uint64:
		return "NUMBER"
	case *float32, *float64:
		return "NUMBER"
	case *string:
		return "STRING"
	case *[]string:
		return "VALUES"
	case *time.Duration:
		return "DURATION"
	default:
		return "VALUE"
	}
}

func (f *Flag) Seen() bool {
	if f.option == nil {
		return false
	}
	return f.option.Seen()
}

func (f *Flag) Occurrences() int {
	if f.option == nil {
		return 0
	}
	return f.option.Count()
}

func (f *Flag) Names() []string {
	return append([]string{f.Name}, f.Aliases...)
}

func (f *Flag) Set(arg string) error {
	return f.option.Value().Set(arg, f.option)
}

func (f *Flag) canonicalName(short bool) string {
	if short == (len(f.Name) == 1) {
		return f.Name
	}
	for _, s := range f.Aliases {
		if short == (len(s) == 1) {
			return s
		}
	}
	return ""
}

func (f *Flag) action() ActionHandler {
	return Action(f.Action)
}

func (f *Flag) before() ActionHandler {
	return Action(f.Before)
}

func (f *Flag) name() string {
	return f.Name
}

func (f *Flag) envVars() []string {
	return f.EnvVars
}

func (f *Flag) filePath() string {
	return f.FilePath
}

func hasOnlyShortName(f *Flag) bool {
	return len(f.Name) == 1
}

func hasNoValue(f *Flag) bool {
	if _, ok := f.Value.(*bool); ok {
		return true
	}
	return false
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

func (a *Arg) name() string {
	return a.Name
}

func (a *Arg) envVars() []string {
	return a.EnvVars
}

func (a *Arg) filePath() string {
	return a.FilePath
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

// flagName gets the long and short name for getopt given the name specified in the flag
func flagName(name string) (string, rune) {
	if len(name) == 1 {
		return "", []rune(name)[0]
	} else {
		return name, 0
	}
}

func loadFlagValueFromEnvironment(f option) (string, bool) {
	if f.Seen() {
		return "", false
	}

	envVars := f.envVars()
	for _, envVar := range envVars {
		envVar = strings.TrimSpace(envVar)
		if val, ok := os.LookupEnv(envVar); ok {
			return val, true
		}
	}

	filePath := f.filePath()
	if len(filePath) > 0 {
		for _, fileVar := range filepath.SplitList(filePath) {
			if data, err := ioutil.ReadFile(fileVar); err == nil {
				return string(data), true
			}
		}
	}
	return "", false
}

func findArgByName(items []*Arg, name string) (*Arg, bool) {
	for _, sub := range items {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}

func findFlagByName(items []*Flag, name string) (*Flag, bool) {
	for _, sub := range items {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}
