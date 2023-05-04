package cli

import (
	"fmt"
	"net"
	"net/url"
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
//	&Flag{
//	  Name: "age",
//	  Value: &age, // var age int -- defined somewhere in scope
//	}
//
//	&Flag{
//	  Name: "age",
//	  Value: cli.Int(), // also sets int
//	}
//
// The corresponding, typed method to access the value of the flag by name is available from the Context.
// In this case, you can  obtain value of the --age=21 flag using Context.Int("flag"), which may be
// necessary when you don't use your own variable.
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
	// text of placeholders.  The placeholder is used in the synopsis for the flag as well
	// as error messages.
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the flag.  If left blank, a succinct synopsis
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

	// DefaultText provides a description of the default value for the flag.  This is displayed
	// on help screens but is otherwise unused
	DefaultText string

	// Options sets various options about how to treat the flag.  For example, options can
	// hide the flag or make its value optional.
	Options Option

	// Category specifies the flag category.  When categories are used, flags are grouped
	// together on the help screen
	Category string

	// Description provides a long description for the flag.  The long description is
	// not used in any templates by default.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description interface{}

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

	// Completion specifies a callback function that determines the auto-complete results
	Completion Completion

	// Transform defines how to interpret the text passed to the *Flag.  This is generally used
	// when specialized syntax preprocesses the text, such as file references.  Refer to the
	// overview in cli.Transform for information.
	Transform TransformFunc

	internalOption
}

// FlagsByName is a sortable slice for flags
type FlagsByName []*Flag

type flagsByCategory []*flagCategory

type flagCategory struct {
	Category string
	Flags    []*Flag
}

type option interface {
	BindingState
	target
	Occurrences() int
	Seen() bool
	Set(any) error
	SetRequired(bool)

	reset()
	actualArgCounter() ArgCounter
	transformFunc() TransformFunc
	ensureInternalOpt()
	contextName() string
	value() interface{}
	name() string
	envVars() []string
	filePath() string
	helpText() string
	manualText() string
	usageText() string
	category() string
	setTransform(fn TransformFunc)
}

type wrapLookupContext struct {
	*optionContext
	actual *Flag
}

type wrapOccurrenceContext struct {
	*optionContext
	index int
	val   interface{}
}

type flagSynopsis struct {
	Short          string
	Shorts         []rune
	Long           string
	Primary        string // Long if present, otherwise Short
	Separator      string
	Names          []string
	AlternateNames []string
	Value          *valueSynopsis
}

type valueSynopsis struct {
	Placeholder string
	helpText    string
	usage       *usage
}

func groupFlagsByCategory(flags []*Flag) flagsByCategory {
	res := flagsByCategory{}
	all := map[string]*flagCategory{}
	category := func(name string) *flagCategory {
		if c, ok := all[name]; ok {
			return c
		}
		c := &flagCategory{Category: name, Flags: []*Flag{}}
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

// Use appends actions to Uses pipeline
func (f *Flag) Use(actions ...Action) *Flag {
	f.Uses = Pipeline(f.Uses).Append(actions...)
	return f
}

// Synopsis contains the name of the flag, its aliases, and the value placeholder.  The text of synopsis
// is inferred from the HelpText.  Up to one short and one long name will be used.
func (f *Flag) Synopsis() string {
	return sprintSynopsis("FlagSynopsis", f.synopsis())
}

func (f *Flag) cacheSynopsis(syn *flagSynopsis) *flagSynopsis {
	f.SetData(synopsisKey, syn)
	return syn
}

func (f *Flag) synopsis() *flagSynopsis {
	if f.Data != nil {
		if a, ok := f.Data[synopsisKey]; ok {
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
	value := getValueSynopsis(f)
	if len(value.Placeholder) == 0 {
		sep = ""
	}

	return (&flagSynopsis{
		Separator: sep,
		Value:     value,
	}).withLongAndShort(long, short)
}

func (f *flagSynopsis) withLongAndShort(long []string, short []rune) *flagSynopsis {
	var primary string

	if len(long) == 0 {
		primary = optionName(string(short[0]))
	} else {
		primary = optionName(long[0])
	}

	var (
		shorts = func() []string {
			res := make([]string, len(short))
			for i := range short {
				res[i] = "-" + string(short[i])
			}
			return res
		}()
		longs = func() []string {
			res := make([]string, len(long))
			for i := range long {
				res[i] = optionName(long[i])
			}
			return res
		}()

		names, alternateNames = func() ([]string, []string) {
			if len(longs) == 0 {
				return []string{shorts[0]}, shorts[1:]
			}
			if len(short) == 0 {
				return []string{longs[0]}, longs[1:]
			}
			return []string{shorts[0], longs[0]}, append(shorts[1:], longs[1:]...)
		}()
	)
	f.Short = shortName(short)
	f.Shorts = short
	f.Long = longName(long)
	f.Primary = primary
	f.Names = names
	f.AlternateNames = alternateNames
	return f
}

// SetData sets the specified metadata on the flag.  When v is nil, the corresponding
// metadata is deleted
func (f *Flag) SetData(name string, v interface{}) {
	f.Data = setData(f.Data, name, v)
}

// LookupData obtains the data if it exists
func (f *Flag) LookupData(name string) (interface{}, bool) {
	v, ok := f.Data[name]
	return v, ok
}

func (f *Flag) setDescription(value interface{}) {
	f.Description = value
}

func (f *Flag) setHelpText(name string) {
	f.HelpText = name
}

func (f *Flag) setManualText(name string) {
	f.ManualText = name
}

func (f *Flag) setCategory(name string) {
	f.Category = name
}

func (f *Flag) setCompletion(c Completion) {
	f.Completion = c
}

func (f *Flag) data() map[string]any {
	return f.Data
}

func (f *Flag) setOptional() {
	f.setOptionalValue(f.internalOption.smartOptionalDefault())
}

func (f *Flag) setOptionalValue(v interface{}) {
	f.setInternalFlags(internalFlagOptional, true)
	f.internalOption.optionalValue = v
}

func (f *Flag) ensureInternalOpt() {
	flags := internalFlagIsFlag
	if f.Value == nil {
		flags |= internalFlagDestinationImplicitlyCreated
	}

	p := f.value()
	f.internalOption = internalOption{
		flags: flags | isFlagType(p),
	}
	f.internalOption.setValue(f.value())
}

func (f *Flag) pipeline(t Timing) interface{} {
	switch t {
	case AfterTiming:
		return f.After
	case BeforeTiming:
		return f.Before
	case InitialTiming:
		return f.Uses
	case ActionTiming:
		fallthrough
	default:
		return f.Action
	}
}

func (f *Flag) options() *Option {
	return &f.Options
}

func (f *Flag) contextName() string {
	if len(f.Name) == 1 {
		return fmt.Sprintf("-%s", f.Name)
	}
	return fmt.Sprintf("--%s", f.Name)
}

func (f *Flag) setTransform(fn TransformFunc) {
	f.Transform = fn
}

func (f *Flag) transformFunc() TransformFunc {
	return f.Transform
}

func (f *Flag) completion() Completion {
	if f.Completion != nil {
		return f.Completion
	}
	if v, ok := f.Value.(valueCompleter); ok {
		return v.Completion()
	}

	return nil
}

func (c *wrapOccurrenceContext) rawOccurrences() [][]string {
	return c.parentLookup.set().Bindings(c.option.name())
}

func (c *wrapOccurrenceContext) current() []string {
	return c.lookupBinding("", true)
}

func (c *wrapOccurrenceContext) numOccurs() int {
	return len(c.rawOccurrences())
}

func (c *wrapOccurrenceContext) lookupBinding(name string, occurs bool) []string {
	if name == "" {
		var index int
		if occurs {
			index = 1
		}
		v := c.rawOccurrences()[c.index]
		return v[index:]
	}

	return c.optionContext.lookupBinding(name, occurs)
}

func (c *wrapOccurrenceContext) lookupValue(name string) (interface{}, bool) {
	if name == "" {
		return c.val, true
	}
	return c.optionContext.lookupValue(name)
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

	if o.usageText() != "" {
		return &valueSynopsis{
			Placeholder: o.usageText(),
			helpText:    usage.WithoutPlaceholders(),
			usage:       usage,
		}
	}

	if len(placeholders) > 0 {
		return &valueSynopsis{
			Placeholder: placeholders,
			usage:       usage,
			helpText:    usage.WithoutPlaceholders(),
		}
	}
	return &valueSynopsis{
		Placeholder: placeholder(o.value()),
		helpText:    usage.WithoutPlaceholders(),
		usage:       usage,
	}
}

func placeholder(v interface{}) string {
	switch m := v.(type) {
	case *bool:
		return ""
	case interface{ IsBoolFlag() bool }:
		if m.IsBoolFlag() {
			return ""
		}

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
	case **url.URL:
		return "URL"
	case *net.IP:
		return "IP"
	case **regexp.Regexp:
		return "PATTERN"
	case valueProvidesSynopsis:
		return m.Synopsis()
	default:
	}
	return "VALUE"
}

// Seen returns true if the flag was used on the command line at least once
func (f *Flag) Seen() bool {
	return f.internalOption.Seen()
}

// Occurrences returns the number of times the flag was specified on the command line
func (f *Flag) Occurrences() int {
	return f.internalOption.Occurrences()
}

// Short gets the short name for the flag including the leading dash.  This is
// the Name of the flag if it contains exactly one character, or this is the
// first alias which contains exactly one character.  This is the empty string
// if the name and all aliases are long names.  The result starts with a dash.
func (f *Flag) Short() string {
	if len(f.Name) == 1 {
		return "-" + f.Name
	}
	for _, nom := range f.Aliases {
		if len(nom) == 1 {
			return "-" + nom
		}
	}
	return ""
}

// Long gets the long name for the flag including the leading dashes.
// This is the Name of the flag if it contains more than one character, or this
// is the first alias which contains more than one character.  If the Name and
// all Aliases have exactly one character, then the value of Name is returned even
// if it has exactly one character.  Note that even in this case, the result starts
// with two leading dashes.
func (f *Flag) Long() string {
	if len(f.Name) > 1 {
		return "--" + f.Name
	}
	for _, nom := range f.Aliases {
		if len(nom) > 1 {
			return "--" + nom
		}
	}
	return "--" + f.Name
}

// Names obtains the name of the flag and its aliases, including their leading
// dashes.
func (f *Flag) Names() []string {
	res := []string{
		optionName(f.Name),
	}
	for _, a := range f.Aliases {
		res = append(res, optionName(a))
	}
	return res
}

// Set will update the value of the flag
func (f *Flag) Set(arg any) error {
	return f.internalOption.Set(arg)
}

// SetOccurrence will update the value of the flag
func (f *Flag) SetOccurrence(values ...string) error {
	return f.internalOption.SetOccurrence(values...)
}

// SetOccurrenceData will update the value of the flag
func (f *Flag) SetOccurrenceData(v any) error {
	return f.internalOption.SetOccurrenceData(v)
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
func (f *Flag) SetHidden(v bool) {
	f.setInternalFlags(internalFlagHidden, true)
}

// SetRequired causes the flag to be required
func (f *Flag) SetRequired(v bool) {
	f.setInternalFlags(internalFlagRequired, v)
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
	f.Value = ensureDestination(f, f.Value, false)
	return f.Value
}

func (f *Flag) category() string {
	return f.Category
}

func (f *Flag) helpText() string {
	return f.HelpText
}

func (f *Flag) manualText() string {
	return f.ManualText
}

func (f *Flag) usageText() string {
	return f.UsageText
}

func (f FlagsByName) Len() int {
	return len(f)
}

func (f FlagsByName) Less(i, j int) bool {
	return f[i].Long() < f[j].Long()
}

func (f FlagsByName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// VisibleFlags filters all flags in the flag category by whether they are not hidden
func (f *flagCategory) VisibleFlags() []*Flag {
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
func (f *flagCategory) Undocumented() bool {
	for _, x := range f.Flags {
		if x.HelpText != "" {
			return false
		}
	}
	return true
}

func (f flagsByCategory) Less(i, j int) bool {
	return f[i].Category < f[j].Category
}

func (f flagsByCategory) Len() int {
	return len(f)
}

func (f flagsByCategory) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func hasOnlyShortName(f *Flag) bool {
	return len(f.Name) == 1
}

func impliesValueFlagOnly(p interface{}) bool {
	switch val := p.(type) {
	case *bool:
		return true
	case interface{ IsBoolFlag() bool }:
		return val.IsBoolFlag()
	}
	return false
}

func ensureDestination(o option, dest interface{}, multi bool) interface{} {
	if dest == nil {
		if !multi {
			return String()
		}
		return List()
	}
	return dest
}

func findFlagByName(items []*Flag, v any) (*Flag, int, bool) {
	if f, ok := v.(*Flag); ok {
		for index, match := range items {
			if f == match {
				return f, index, true
			}
		}
		return nil, -1, false
	}
	name := strings.TrimLeft(v.(string), "-")
	for index, sub := range items {
		if sub.Name == name {
			return sub, index, true
		}
		for _, alias := range sub.Aliases {
			if alias == name {
				return sub, index, true
			}
		}
	}
	return nil, -1, false
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

func isFlagType(p interface{}) internalFlags {
	if impliesValueFlagOnly(p) {
		return internalFlagFlagOnly
	}
	return 0
}

var _ target = (*Flag)(nil)
