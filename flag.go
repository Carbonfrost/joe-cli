package cli

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
type Flag struct {
	pipelinesSupport

	// Name provides the name of the flag. This value must be set, and it is used to access
	// the flag's value via the context
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
	HelpText string

	// UsageText provides the usage for the flag.  If left blank, a succint synopsis
	// is generated from the type of the flag's value
	UsageText string

	// EnvVars specifies the name of environment variables that are read to provide the
	// default value of the flag.
	EnvVars []string

	// FilePath specifies a file that is loaded to provide the default value of the flag.
	FilePath string

	// Value provides the value of the flag.  Any of the following types are valid for the
	// value:
	//
	//   * *bool
	//   * *time.Duration
	//   * *float32
	//   * *float64
	//   * *int
	//   * *int16
	//   * *int32
	//   * *int64
	//   * *int8
	//   * *net.IP
	//   * *[]string
	//   * *map[string]string
	//   * **regexp.Regexp
	//   * *string
	//   * *uint
	//   * *uint16
	//   * *uint32
	//   * *uint64
	//   * *uint8
	//   * **url.URL
	//   * an implementation of Value interface
	//
	// If unspecified, the value will be a string pointer.
	// For more information about Values, see the Value type
	Value interface{}

	// DefaultText provides a description of the detault value for the flag.  This is displayed
	// on help screens but is otherwise unused
	DefaultText string

	// Options sets various options about how to treat the flag.  For example, options can
	// hide the flag or make its value optional.
	Options Option

	// Category specifies the flag category.  When categories are used, flags are grouped
	// together on the help screen
	Category string

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

	option internalOption
}

// FlagsByName is a sortable slice for flags
type FlagsByName []*Flag

// FlagsByCategory provides a slice that can sort on category names and the flags
// themselves
type FlagsByCategory []*FlagCategory

// FlagCategory names a category and the flags it contains
type FlagCategory struct {
	// Category is the name of the category
	Category string

	// Flags in the category
	Flags []*Flag
}

type option interface {
	target
	Occurrences() int
	Seen() bool
	Set(string) error
	SetHidden()
	SetRequired()

	applyToSet(s *set)
	wrapAction(func(Action) ActionFunc)
	value() interface{}
	name() string
	envVars() []string
	filePath() string
	helpText() string
}

type flagContext struct {
	option  *Flag
	argList []string
}

type wrapLookupContext struct {
	*flagContext
	actual *Flag
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

// GroupFlagsByCategory groups together flags by category and sorts the groupings.
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
	s.defineFlag(&f.option)
}

// Synopsis contains the name of the flag, its aliases, and the value placeholder.  The text of synopsis
// is inferred from the HelpText.  Up to one short and one long name will be used.
func (f *Flag) Synopsis() string {
	return textUsage.flag(f.synopsis(), false)
}

func (f *Flag) cacheSynopsis(syn *flagSynopsis) *flagSynopsis {
	f.SetData("_Synopsis", syn)
	return syn
}

func (f *Flag) synopsis() *flagSynopsis {
	if f.Data != nil {
		if a, ok := f.Data["_Synopsis"]; ok {
			return a.(*flagSynopsis)
		}
	}
	return f.cacheSynopsis(f.newSynopsis())
}

func (f *Flag) newSynopsis() *flagSynopsis {
	long, short := canonicalNames(f.Name, f.Aliases)
	sep := "="

	if len(long) == 0 {
		sep = " "
	}

	return &flagSynopsis{
		short: shortName(short),
		long:  longName(long),
		sep:   sep,
		value: getValueSynopsis(f),
	}
}

// SetData sets the specified metadata on the flag
func (f *Flag) SetData(name string, v interface{}) {
	f.ensureData()[name] = v
}

func (f *Flag) setCategory(name string) {
	f.Category = name
}

func (f *Flag) setOptional() {
	f.setOptionalValue(f.option.value.smartOptionalDefault())
}

func (f *Flag) setOptionalValue(v interface{}) {
	f.option.optional = true
	f.option.optionalValue = v
}

func (f *Flag) ensureData() map[string]interface{} {
	if f.Data == nil {
		f.Data = map[string]interface{}{}
	}
	return f.Data
}

func (*Flag) hookAfter(string, Action) error {
	return cantHookError
}

func (*Flag) hookBefore(string, Action) error {
	return cantHookError
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

func (o *flagContext) initialize(c *Context) error {
	f := o.option
	p := f.value()
	long, short := canonicalNames(f.Name, f.Aliases)
	f.option = internalOption{
		short: short,
		long:  long,
		value: wrapGeneric(p),
		uname: f.Name,
		flag:  isFlagType(p),
	}

	rest := newPipelines(ActionOf(f.Uses), &f.Options)
	f.setPipelines(rest)
	return executeAll(c, rest.Initializers, defaultOption.Initializers)
}

func (o *flagContext) executeBefore(ctx *Context) error {
	tt := o.option
	return executeAll(ctx, tt.uses().Before, ActionOf(tt.Before), defaultOption.Before)
}

func (o *flagContext) executeBeforeDescendent(ctx *Context) error { return nil }
func (o *flagContext) executeAfterDescendent(ctx *Context) error  { return nil }
func (o *flagContext) executeAfter(ctx *Context) error {
	tt := o.option
	return executeAll(ctx, tt.uses().After, ActionOf(tt.After), defaultOption.After)
}
func (o *flagContext) execute(ctx *Context) error {
	return executeAll(ctx, o.option.uses().Action, ActionOf(o.option.Action))
}
func (o *flagContext) args() []string           { return o.argList }
func (o *flagContext) set() *set                { return nil }
func (o *flagContext) target() target           { return o.option }
func (o *flagContext) setDidSubcommandExecute() {}
func (o *flagContext) lookupValue(name string) (interface{}, bool) {
	if name == "" {
		return o.option.value(), true
	}
	return nil, false
}
func (o *flagContext) Name() string {
	if len(o.option.Name) == 1 {
		return fmt.Sprintf("-%s", o.option.Name)
	}
	return fmt.Sprintf("--%s", o.option.Name)
}

func (o *wrapLookupContext) lookupValue(name string) (interface{}, bool) {
	if name == "" {
		return o.actual.value(), true
	}
	return nil, false
}

func (o *wrapLookupContext) Name() string {
	return o.actual.Name
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
	switch m := v.(type) {
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
	case *map[string]string:
		return "NAME=VALUE"
	case *File:
		return "FILE"
	case **url.URL:
		return "URL"
	case *net.IP:
		return "IP"
	case **regexp.Regexp:
		return "PATTERN"
	case interface{ Synopsis() string }:
		return m.Synopsis()
	default:
		return "VALUE"
	}
}

// Seen returns true if the flag was used on the command line at least once
func (f *Flag) Seen() bool {
	return f.option.Seen()
}

// Occurrences returns the number of times the flag was specified on the command line
func (f *Flag) Occurrences() int {
	return f.option.Occurrences()
}

// Names obtains the name of the flag and its aliases
func (f *Flag) Names() []string {
	return append([]string{f.Name}, f.Aliases...)
}

// Set will update the value of the flag
func (f *Flag) Set(arg string) error {
	return f.option.Set(arg)
}

func canonicalNames(name string, aliases []string) (long []string, short []rune) {
	long = make([]string, 0, len(aliases))
	short = make([]rune, 0, len(aliases))
	names := append([]string{name}, aliases...)

	for _, nom := range names {
		if len(nom) == 1 {
			short = append(short, ([]rune(nom))[0])
		} else {
			long = append(long, nom)
		}
	}
	return
}

// SetHidden causes the flag to be hidden
func (f *Flag) SetHidden() {
	f.setInternalFlags(internalFlagHidden)
}

// SetRequired causes the flag to be required
func (f *Flag) SetRequired() {
	f.setInternalFlags(internalFlagRequired)
}

func (f *Flag) internalFlags() internalFlags {
	return f.option.flags
}

func (f *Flag) setInternalFlags(i internalFlags) {
	f.option.flags |= i
}

func (f *Flag) wrapAction(fn func(Action) ActionFunc) {
	f.Action = fn(ActionOf(f.Action))
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

func (f *Flag) longNamePreferred() string {
	if len(f.Name) > 1 {
		return f.Name
	}
	for _, n := range f.Aliases {
		if len(n) > 1 {
			return n
		}
	}
	return f.Name
}

func (f FlagsByName) Len() int {
	return len(f)
}

func (f FlagsByName) Less(i, j int) bool {
	return f[i].longNamePreferred() < f[j].longNamePreferred()
}

func (f FlagsByName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// VisibleFlags filters all flags in the flag category by whether they are not hidden
func (f *FlagCategory) VisibleFlags() []*Flag {
	res := make([]*Flag, 0, len(f.Flags))
	for _, o := range f.Flags {
		if o.internalFlags().hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

// Undocumented determines whether the category is undocumented (i.e. has no HelpText set
// on any of its flags)
func (f *FlagCategory) Undocumented() bool {
	for _, x := range f.Flags {
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

func longName(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func shortName(s []rune) string {
	if len(s) == 0 {
		return ""
	}
	return string(s[0])
}

func isFlagType(p interface{}) bool {
	switch p.(type) {
	case *bool:
		return true
	}
	return false
}
