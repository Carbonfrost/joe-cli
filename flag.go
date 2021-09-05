package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	Value       interface{}
	DefaultText string
	Options     Option
	Category    string

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// After executes after the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	After interface{}

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses interface{}

	// Action executes if the flag was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action interface{}
	option *internalOption
	flags  internalFlags
}

// FlagsByName is a sortable slice for flags
type FlagsByName []*Flag
type FlagsByCategory []*FlagCategory
type FlagCategory struct {
	Category string
	Flags    []*Flag
}

type option interface {
	target
	Occurrences() int
	Seen() bool
	Set(string) error
	SetHidden()
	SetRequired()

	applyToSet(s *set)
	wrapAction(func(ActionHandler) ActionFunc)
	setInternalFlags(internalFlags)
	options() Option
	internalFlags() internalFlags
	value() interface{}
	name() string
	envVars() []string
	filePath() string
	helpText() string
	before() ActionHandler
	after() ActionHandler
	action() ActionHandler
}

type flagSynopsis struct {
	short string
	long  string
	sep   string
	value *valueSynopsis
}

type valueSynopsis struct {
	placeholder string
	helpText    string
	usage       *usage
}

func GroupFlagsByCategory(flags []*Flag) FlagsByCategory {
	res := FlagsByCategory{}
	all := map[string]*FlagCategory{}
	category := func(name string) *FlagCategory {
		if c, ok := all[name]; ok {
			return c
		}
		c := &FlagCategory{Category: name, Flags: []*Flag{}}
		all[name] = c
		res = append(res, c)
		return c
	}
	for _, f := range flags {
		cc := category(f.Category)
		cc.Flags = append(cc.Flags, f)
	}
	sort.Sort(res)
	return res
}

func (f *Flag) applyToSet(s *set) {
	for _, name := range f.Names() {
		f.option = s.defineFlag(f.Name, name, f.value())
	}
}

// Synopsis contains the name of the flag, its aliases, and the value placeholder.  The text of synopsis
// is inferred from the HelpText.  Up to one short and one long name will be used.
func (f *Flag) Synopsis() string {
	return textUsage.flag(f.newSynopsis(), false)
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
		value: getValueSynopsis(f),
	}
}

func (f *Flag) setData(name string, v interface{}) {
	f.ensureData()[name] = v
}

func (f *Flag) setCategory(name string) {
	f.Category = name
}

func (f *Flag) initialize(c *Context) error {
	return hookExecute(Action(f.Uses), nil, c)
}

func (f *Flag) ensureData() map[string]interface{} {
	if f.Data == nil {
		f.Data = map[string]interface{}{}
	}
	return f.Data
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

func getValueSynopsis(o option) *valueSynopsis {
	usage := parseUsage(o.helpText())
	placeholders := strings.Join(usage.Placeholders(), " ")
	if len(placeholders) > 0 {
		return &valueSynopsis{
			placeholder: placeholders,
			usage:       usage,
			helpText:    usage.WithoutPlaceholders(),
		}
	}
	return &valueSynopsis{
		placeholder: placeholder(o.value()),
		helpText:    usage.WithoutPlaceholders(),
		usage:       usage,
	}
}

func placeholder(v interface{}) string {
	switch v.(type) {
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

func (f *Flag) SetHidden() {
	f.flags |= internalFlagHidden
}

func (f *Flag) SetRequired() {
	f.flags |= internalFlagRequired
}

func (f *Flag) internalFlags() internalFlags {
	return f.flags
}

func (f *Flag) setInternalFlags(i internalFlags) {
	f.flags |= i
}

func (f *Flag) options() Option {
	return f.Options
}

func (f *Flag) wrapAction(fn func(ActionHandler) ActionFunc) {
	f.Action = fn(Action(f.Action))
}

func (f *Flag) action() ActionHandler {
	return Action(f.Action)
}

func (f *Flag) before() ActionHandler {
	return Action(f.Before)
}

func (f *Flag) after() ActionHandler {
	return Action(f.After)
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

func (f *Flag) value() interface{} {
	f.Value = ensureDestination(f.Value, false)
	return f.Value
}

func (f *Flag) helpText() string {
	return f.HelpText
}

func (f FlagsByName) Len() int {
	return len(f)
}

func (f FlagsByName) Less(i, j int) bool {
	return f[i].canonicalName(false) < f[j].canonicalName(false)
}

func (f FlagsByName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f *FlagCategory) VisibleFlags() []*Flag {
	res := make([]*Flag, 0, len(f.Flags))
	for _, o := range f.Flags {
		if o.flags.hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (e *FlagCategory) Undocumented() bool {
	for _, x := range e.Flags {
		if x.HelpText != "" {
			return false
		}
	}
	return true
}

func (f FlagsByCategory) Less(i, j int) bool {
	return f[i].Category < f[j].Category
}

func (f FlagsByCategory) Len() int {
	return len(f)
}

func (f FlagsByCategory) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
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

func ensureDestination(dest interface{}, multi bool) interface{} {
	if dest == nil {
		if !multi {
			return String()
		}
		return List()
	}
	return dest
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

func findFlagByName(items []*Flag, name string) (*Flag, bool) {
	for _, sub := range items {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}
