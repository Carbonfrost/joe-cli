package cli

import (
	"fmt"
	"strings"
)

// Arg provides the representation of a positional argument.
type Arg struct {
	pipelinesSupport

	// Name provides the name of the argument. This value must be set, and it is used to access
	// the argument's value via the context
	Name string

	// EnvVars specifies the name of environment variables that are read to provide the
	// default value of the argument.
	EnvVars []string

	// FilePath specifies a file that is loaded to provide the default value of the argument.
	FilePath string

	// HelpText contains text which briefly describes the usage of the argument.
	// For style, generally the usage text should be limited to about 40 characters.
	// Sentence case is recommended for the usage text.    Uppercase is recommended for the
	// text of placeholders.  The placeholder is used in the synopsis for the argument as well
	// as error messages.
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// Description provides a long description for the flag.  The long description is
	// not used in any templates by default.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description interface{}

	// Category specifies the arg category.  Categories are not used by the help screen.
	Category string

	// UsageText provides the usage for the argument.  If left blank, a succinct synopsis
	// is generated from the type of the argument's value
	UsageText string

	// Value provides the value of the argument.  Any of the following types are valid for the
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
	// If unspecified, the value will depend upon NArg if it is a number, in which case either
	// a pointer to a string or a string slice will be used depending upon the semantics of the
	// ArgCount function.  For more information about Values, see the Value type
	Value interface{}

	// DefaultText provides a description of the default value for the argument.  This is displayed
	// on help screens but is otherwise unused
	DefaultText string

	// NArg describes how many values are passed to the argument.  For a description, see
	// ArgCount function. By convention, if the flag Value provides the method NewCounter() ArgCounter,
	// this method is consulted to obtain the arg counter.
	NArg interface{}

	// Options sets various options about how to treat the argument.
	Options Option

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// Action executes if the argument was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action interface{}

	// After executes after the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	After interface{}

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses interface{}

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	// Completion specifies a callback function that determines the auto-complete results
	Completion Completion

	option internalOption
}

//counterfeiter:generate . ArgCounter

// ArgCounter provides the behavior of counting
type ArgCounter interface {
	// Take considers the argument and and returns whether it can be used.
	// If the error EndOfArguments is returned, then the arg counter is done with
	// taking arguments.  All other errors are treated as fatal.
	Take(arg string, possibleFlag bool) error

	// Done is invoked to signal the end of arguments
	Done() error
}

type optionContext struct {
	option       option
	parentLookup internalCommandContext
}

type discreteCounter struct {
	total int
	count int
}

type defaultCounter struct {
	seen        bool
	requireSeen bool
}

type varArgsCounter struct {
	stopOnFlags bool
	intersperse bool
}

type matchesArgsCounter struct {
	fn    func(string) bool
	count int
	max   int
}

type argSynopsis struct {
	Value    string
	Multi    bool
	Optional bool
}

const (
	// TakeRemaining is the value to use for Arg.NArg to indicate that an argument takes
	// all the remaining tokens from the command line
	TakeRemaining = -1

	// TakeUntilNextFlag is the value to use for Arg.NArg to indicate that an argument takes
	// tokens from the command line until one looks like a flag.
	TakeUntilNextFlag = -2

	// TakeExceptForFlags is the value to use for Arg.NArg to indicate that an argument can
	// be interspersed with values that look like flags.  When the flag syntax is encountered,
	// a flag will be parsed and parsing the argument will resume.
	TakeExceptForFlags = -3
)

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

// ArgCount gets the arg counter for the specified value.  If the value is an int, it is interpreted
// as the discrete number of values in the argument if it is 1 or greater, but if it is < 0
// it implies taking all arguments, or 0 means take it if it exists.
//
//	>= 1   take exactly n number of arguments, though if they look like flags treat as an error
//	   0   take argument if it does not look like a flag
//	  -1   take all remaining arguments (even when they look like flags) (TakeRemaining)
//	  -2   take all remaining arguments but stop before taking one that looks like a flag (TakeUntilNextFlag)
//
// Any other negative value uses the behavior of -1.
func ArgCount(v interface{}) ArgCounter {
	switch count := v.(type) {
	case ArgCounter:
		return count
	case int:
		if count > 0 {
			return &discreteCounter{count, count}
		}
		if count == 0 {
			return &defaultCounter{}
		}
		return &varArgsCounter{
			stopOnFlags: count == TakeUntilNextFlag || count == TakeExceptForFlags,
			intersperse: count == TakeExceptForFlags,
		}
	case nil:
		return ArgCount(0)
	default:
		panic(fmt.Sprintf("unexpected type: %T", v))
	}
}

// NoArgs provides an argument counter that takes no args.
func NoArgs() ArgCounter {
	return &discreteCounter{0, 0}
}

// OptionalArg provides an argument counter which matches zero or one
// argument using the specified function.
func OptionalArg(fn func(string) bool) ArgCounter {
	if fn == nil {
		fn = func(string) bool { return true }
	}
	return &matchesArgsCounter{
		fn:    fn,
		count: 0,
		max:   1,
	}
}

// Occurrences counts the number of times that the argument has occurred on the command line
func (a *Arg) Occurrences() int {
	return a.option.Occurrences()
}

// Seen reports true if the argument is used at least once.
func (a *Arg) Seen() bool {
	return a.option.Seen()
}

// Set will set the value of the argument
func (a *Arg) Set(arg string) error {
	return a.option.Set(arg)
}

// SetHidden causes the argument to be hidden from the help screen
func (a *Arg) SetHidden(v bool) {
	a.setInternalFlags(internalFlagHidden, v)
}

// SetRequired will indicate that the argument is required.
func (a *Arg) SetRequired(v bool) {
	a.setInternalFlags(internalFlagRequired, v)
}

// Use appends actions to Uses pipeline
func (a *Arg) Use(actions ...Action) *Arg {
	a.Uses = Pipeline(a.Uses).Append(actions...)
	return a
}

// Synopsis contains the value placeholder
func (a *Arg) Synopsis() string {
	return sprintSynopsis("ArgSynopsis", a.newSynopsis())
}

func (a *Arg) newSynopsis() *argSynopsis {
	return a.newSynopsisCore(fmt.Sprintf("<%s>", a.Name))
}

func (a *Arg) newSynopsisCore(defaultUsage string) *argSynopsis {
	usage := a.UsageText
	if usage == "" {
		usage = defaultUsage
	}

	opt, mul := aboutArgCounter(a.NArg)
	return &argSynopsis{
		Value:    usage,
		Multi:    mul,
		Optional: opt,
	}
}

func (a *Arg) internalFlags() internalFlags {
	return a.option.flags
}

func (a *Arg) setInternalFlags(i internalFlags, v bool) {
	if v {
		a.option.flags |= i
	} else {
		a.option.flags &= ^i
	}
}

func (a *Arg) applyToSet(s *set) {
	a.Name = s.defineArg(&a.option, a.Name)
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

func (a *Arg) category() string {
	return a.Category
}

func (a *Arg) helpText() string {
	return a.HelpText
}

func (a *Arg) manualText() string {
	return a.ManualText
}

func (a *Arg) usageText() string {
	return a.UsageText
}

func (a *Arg) value() interface{} {
	_, multi := aboutArgCounter(a.NArg)
	a.Value = ensureDestination(a, a.Value, multi)
	return a.Value
}

// SetData sets the specified metadata on the arg.  When v is nil, the corresponding
// metadata is deleted
func (a *Arg) SetData(name string, v interface{}) {
	a.Data = setData(a.Data, name, v)
}

// LookupData obtains the data if it exists
func (a *Arg) LookupData(name string) (interface{}, bool) {
	v, ok := a.Data[name]
	return v, ok
}

func (a *Arg) setCategory(name string) {
	a.Category = name
}

func (a *Arg) setDescription(value interface{}) {
	a.Description = value
}

func (a *Arg) setManualText(value string) {
	a.ManualText = value
}

func (a *Arg) setHelpText(value string) {
	a.HelpText = value
}

func (a *Arg) setCompletion(c Completion) {
	a.Completion = c
}

func (a *Arg) pipeline(t Timing) interface{} {
	switch t {
	case AfterTiming:
		return a.After
	case BeforeTiming:
		return a.Before
	case InitialTiming:
		return a.Uses
	case ActionTiming:
		return a.Action
	default:
		panic("unreachable")
	}
}

func (a *Arg) options() *Option {
	return &a.Options
}

func (a *Arg) contextName() string {
	return fmt.Sprintf("<%s>", a.Name)
}

func (a *Arg) ensureInternalOpt() {
	var flags internalFlags
	if a.Value == nil {
		flags = internalFlagDestinationImplicitlyCreated
	}
	a.option = internalOption{
		value: wrapGeneric(a.value()),
		narg:  a.NArg,
		flags: flags,
	}
}

func (a *Arg) setTransform(fn TransformFunc) {
	a.option.transform = fn
}

func (a *Arg) completion() Completion {
	if a.Completion != nil {
		return a.Completion
	}
	if v, ok := a.Value.(valueCompleter); ok {
		return v.Completion()
	}

	return nil
}

func (a *argSynopsis) String() string {
	if a.Multi {
		return a.Value + "..."
	}
	return a.Value
}

func (o *optionContext) initialize(c *Context) error {
	return execute(c, defaultOption.Initializers)
}

func (o *optionContext) executeBefore(ctx *Context) error {
	return execute(ctx, defaultOption.Before)
}

func (o *optionContext) initializeDescendent(ctx *Context) error    { return nil }
func (o *optionContext) executeBeforeDescendent(ctx *Context) error { return nil }
func (o *optionContext) executeAfterDescendent(ctx *Context) error  { return nil }
func (o *optionContext) executeAfter(ctx *Context) error {
	return execute(ctx, defaultOption.After)
}

func (o *optionContext) execute(ctx *Context) error {
	return execute(ctx, defaultOption.Action)
}

func (o *optionContext) lookupBinding(name string, occurs bool) []string {
	if _, isArg := o.option.(*Arg); isArg {
		// Don't specify the argument name when obtaining current binding
		return o.parentLookup.set().RawOccurrences(o.option.name())
	}
	return o.parentLookup.lookupBinding(o.option.name(), occurs)
}

func (o *optionContext) target() target { return o.option }
func (o *optionContext) lookupValue(name string) (interface{}, bool) {
	if name == "" {
		return o.option.value(), true
	}
	return nil, false
}
func (o *optionContext) Name() string {
	return o.option.contextName()
}

func (d *discreteCounter) Take(arg string, possibleFlag bool) error {
	if d.count == 0 {
		return EndOfArguments
	}
	d.count--
	return nil
}

func (d *discreteCounter) Done() error {
	if d.count > 0 {
		return expectedArgument(d.total)
	}
	return nil
}

func (d *defaultCounter) Take(arg string, possibleFlag bool) error {
	if d.seen {
		return EndOfArguments
	}
	d.seen = true
	return nil
}

func (d *defaultCounter) Done() error {
	if d.requireSeen && !d.seen {
		return expectedArgument(1)
	}
	return nil
}

func (v *varArgsCounter) Take(arg string, possibleFlag bool) error {
	if v.stopOnFlags && allowFlag(arg, possibleFlag) {
		if v.intersperse {
			return argCannotUseFlag
		}
		return EndOfArguments
	}
	return nil
}

func (*varArgsCounter) Done() error {
	return nil
}

func (o *matchesArgsCounter) Take(arg string, possibleFlag bool) error {
	if o.fn(arg) && o.count < o.max {
		o.count++
		return nil
	}
	o.count++
	return EndOfArguments
}

func (*matchesArgsCounter) Done() error {
	return nil
}

func findArgByName(items []*Arg, v interface{}) (*Arg, int, bool) {
	switch name := v.(type) {
	case int:
		if name < 0 {
			name = len(items) + name
		}
		if name >= 0 && name < len(items) {
			return items[name], name, true
		}
	case string:
		name = strings.TrimPrefix(name, "<")
		name = strings.TrimSuffix(name, ">")
		for i, sub := range items {
			if sub.Name == name {
				return sub, i, true
			}
		}
	case *Arg:
		for i := range items {
			if items[i] == v {
				return name, i, true
			}
		}
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}

	return nil, -1, false
}

func aboutArgCounter(narg interface{}) (optional, multi bool) {
	switch c := narg.(type) {
	case int:
		return c == 0, c < 0 || c > 1
	case *varArgsCounter:
		return true, true
	case nil:
		return true, false
	case *defaultCounter:
		return !c.requireSeen, false
	case *discreteCounter:
		return false, c.count > 1
	}
	return false, false
}

var _ target = (*Arg)(nil)
