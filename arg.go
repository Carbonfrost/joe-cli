// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cli

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

// Arg provides the representation of a positional argument.
type Arg struct {
	pipelinesSupport
	hooksSupport
	internalFlagsSupport

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
	Description any

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
	Value any

	// DefaultText provides a description of the default value for the argument.  This is displayed
	// on help screens but is otherwise unused
	DefaultText string

	// NArg describes how many values are passed to the argument.  For a description, see
	// ArgCount function. By convention, if the flag Value provides the method NewCounter() ArgCounter,
	// this method is consulted to obtain the arg counter.
	NArg any

	// Options sets various options about how to treat the argument.
	Options Option

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before any

	// Action executes if the argument was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action any

	// After executes after the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	After any

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses any

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]any

	// Completion specifies a callback function that determines the auto-complete results
	Completion Completion

	// Transform defines how to interpret the text passed to the *Arg.  This is generally used
	// when specialized syntax preprocesses the text, such as file references.  Refer to the
	// overview in cli.Transform for information.
	Transform TransformFunc

	count int
}

//counterfeiter:generate . ArgCounter

// ArgCounter provides the behavior of counting the values that
// are specified to an Arg.  The method Take is called repeatedly
// to process each occurrence of a value until the special error
// EndOfArguments is returned.  The method Done is used
// to finalize the counting operation.
type ArgCounter interface {
	// Take considers the argument and determines whether it can be used.
	// If the special error value EndOfArguments is returned, then the
	// value is not consumed and the counting operation stops.
	// Other errors are treated as parse errors.  When possibleFlag is
	// set, it signals that the argument could be a flag in such cases
	// that flags and args are interspersed on the command line.  When this is
	// set to true, you may prefer to stop parsing and let the next argument be
	// parsed as a flag.
	//
	// The interface can optionally provide a method that describes
	// the synopsis for args used with the counter:
	//
	//   - Usage() (optional, multi bool)  called to query the usage of the
	//                       arg counter, whether the arg is possibly optional
	//                       and whether the arg may be multiple values.
	Take(arg string, possibleFlag bool) error

	// Done is invoked to signal the end of arguments
	Done() error
}

type argCounterUsage interface {
	Usage() (bool, bool)
}

type optionContext struct {
	option       option
	parentLookup internalContext
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

const (
	// TakeUnlessFlag is the value to use for Arg.NArg to indicate that an argument takes the
	// value unless it looks like a flag.  This is the default
	TakeUnlessFlag = 0

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
// A panic occurs when this function is not called properly: when names and values
// are not arranged in pairs or when an unsupported type of value is used.
func Args(namevalue ...any) []*Arg {
	if len(namevalue)%2 != 0 {
		panic("unexpected number of arguments")
	}
	res := make([]*Arg, 0, len(namevalue)/2)
	for i := 0; i < len(namevalue); i += 2 {
		value := namevalue[i+1]
		if err := checkSupportedFlagType(value); err != nil {
			panic(err)
		}
		res = append(res, &Arg{
			Name:  namevalue[i].(string),
			Value: value,
		})
	}
	return res
}

// ArgCount gets the arg counter for the specified value.  If the value is an int, it is interpreted
// as the discrete number of values in the argument if it is 1 or greater, but if it is < 0
// it implies taking all arguments, or 0 means take it if it exists.
//
//	>= 1   take exactly n number of arguments, though if they look like flags treat as an error
//	   0   take argument if it does not look like a flag ([TakeUnlessFlag])
//	  -1   take all remaining arguments (even when they look like flags) ([TakeRemaining])
//	  -2   take all remaining arguments but stop before taking one that looks like a flag ([TakeUntilNextFlag])
//	  -3   take all remaining arguments except ones that look like flags ([TakeExceptForFlags])
//
// Any other negative value uses the behavior of -1.
// As a special case, if v is an initialized *Arg or *Flag, it obtains the actual arg counter which will be
// used for it, or if v is nil, this is the same as [TakeUnlessFlag].  If the value
// is already ArgCounter, it is returned as-is.
func ArgCount(v any) ArgCounter {
	switch count := v.(type) {
	case ArgCounter:
		return count
	case int:
		if count > 0 {
			return &discreteCounter{count, count}
		}
		if count == TakeUnlessFlag {
			return &defaultCounter{}
		}
		return &varArgsCounter{
			stopOnFlags: count == TakeUntilNextFlag || count == TakeExceptForFlags,
			intersperse: count == TakeExceptForFlags,
		}
	case *Arg, *Flag:
		if !count.(option).internalFlags().initialized() {
			panic(fmt.Sprintf("value %T is not initialized", count))
		}
		return count.(option).actualArgCounter()
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
	return a.count
}

// Seen reports true if the argument is used at least once.
func (a *Arg) Seen() bool {
	return a.count > 0
}

// Set will set the value of the argument
func (a *Arg) Set(v any) error {
	if arg, ok := v.(string); ok {
		return setCore(a.Value, a.flags.disableSplitting(), arg)
	}

	return setDirect(a.Value, v)
}

// SetOccurrence will update the value of the arg
func (a *Arg) SetOccurrence(values ...string) error {
	return optionSetOccurrence(a, values...)
}

// SetOccurrenceData will update the value of the arg
func (a *Arg) SetOccurrenceData(v any) error {
	a.nextOccur()
	return SetData(a.Value, v)
}

func (a Arg) actualArgCounter() ArgCounter {
	if a.NArg != nil {
		return ArgCount(a.NArg)
	}
	return argCounterImpliedFromValue(a.Value, false)
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
func (a *Arg) Use(action Action) *Arg {
	a.Uses = Pipeline(a.Uses).Append(action)
	return a
}

// Synopsis contains the value placeholder
func (a *Arg) Synopsis() string {
	return sprintSynopsis(a.newSynopsis())
}

func (a *Arg) newSynopsis() *synopsis.Arg {
	defaultUsage := fmt.Sprintf("<%s>", a.Name)
	return synopsis.NewArg(cmp.Or(a.UsageText, defaultUsage), a.NArg)
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

func (a *Arg) defaultText() string {
	return a.DefaultText
}

func (a *Arg) value() any {
	_, multi := synopsis.ArgCounter(a.NArg)
	var created bool
	a.Value, created = ensureDestination(a.Value, multi)
	if created {
		a.setInternalFlags(internalFlagDestinationImplicitlyCreated, true)
	}

	if _, ok := a.Value.(*string); ok {
		a.setInternalFlags(internalFlagMerge, true)
	}
	return a.Value
}

// SetData sets the specified metadata on the arg.  When v is nil, the corresponding
// metadata is deleted
func (a *Arg) SetData(name string, v any) {
	a.Data = setData(a.Data, name, v)
}

// LookupData obtains the data if it exists
func (a *Arg) LookupData(name string) (any, bool) {
	v, ok := a.Data[name]
	return v, ok
}

func (a *Arg) setCategory(name string) {
	a.Category = name
}

func (a *Arg) setDefaultText(name string) {
	a.DefaultText = name
}

func (a *Arg) setDescription(value any) {
	a.Description = value
}

func (a *Arg) setManualText(value string) {
	a.ManualText = value
}

func (a *Arg) setHelpText(value string) {
	a.HelpText = value
}

func (a *Arg) setUsageText(s string) {
	a.UsageText = s
}

func (a *Arg) setCompletion(c Completion) {
	a.Completion = c
}

func (a *Arg) data() map[string]any {
	return a.Data
}

func (a *Arg) description() any {
	return a.Description
}

func (a *Arg) helpText() string {
	return a.HelpText
}

func (a *Arg) usageText() string {
	return a.UsageText
}

func (a *Arg) manualText() string {
	return a.ManualText
}

func (a *Arg) pipeline(t Timing) any {
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

func (a *Arg) setTransform(fn TransformFunc) {
	a.Transform = fn
}

func (a *Arg) transformFunc() TransformFunc {
	return a.Transform
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

func (a *Arg) cloneZero() {
	a.Value = valueCloneZero(a.Value)
}

func (a *Arg) nextOccur() {
	a.count++
	optionApplyValueConventions(a.Value, a.flags, a.count == 1)
}

func (a *Arg) reset() {
	optionReset(a.Value, a.internalFlags())
}

func (o *optionContext) argHandling() ([]string, bool) {
	if _, isArg := o.option.(*Arg); isArg {
		// Don't specify the argument name when obtaining current binding
		return o.parentLookup.RawOccurrences(o.option.name()), true
	}
	return nil, false
}

func (o *optionContext) Raw(name string) []string {
	if arg, ok := o.argHandling(); ok {
		return arg
	}
	return o.parentLookup.Raw(o.option.name())
}

func (o *optionContext) RawOccurrences(name string) []string {
	if arg, ok := o.argHandling(); ok {
		return arg
	}
	return o.parentLookup.RawOccurrences(o.option.name())
}

func (o *optionContext) Bindings(name string) [][]string {
	return o.parentLookup.Bindings(name)
}

func (o *optionContext) BindingNames() []string {
	return o.parentLookup.BindingNames()
}

func (o *optionContext) lookupValue(name string) (any, bool) {
	if name == "" {
		return o.option.value(), true
	}
	return nil, false
}

func (d *discreteCounter) Take(_ string, _ bool) error {
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

func (d *discreteCounter) Usage() (optional, multi bool) {
	return false, d.count > 1
}

func (d *defaultCounter) Take(_ string, _ bool) error {
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

func (d *defaultCounter) Usage() (optional, multi bool) {
	return !d.requireSeen, false
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

func (*varArgsCounter) Usage() (optional, multi bool) {
	return true, true
}

func (*matchesArgsCounter) Usage() (optional, multi bool) {
	return true, false
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

func argCounterImpliedFromValue(value any, requireSeen bool) ArgCounter {
	switch value := value.(type) {
	case *[]string:
		return ArgCount(TakeUntilNextFlag)
	case valueProvidesCounter:
		return value.NewCounter()
	}
	return &defaultCounter{requireSeen: requireSeen}
}

func findArgByName(items []*Arg, v any) (*Arg, int, bool) {
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

var _ target = (*Arg)(nil)

var (
	_ argCounterUsage = (*varArgsCounter)(nil)
	_ argCounterUsage = (*defaultCounter)(nil)
	_ argCounterUsage = (*discreteCounter)(nil)
)
