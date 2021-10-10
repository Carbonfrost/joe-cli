package cli

import (
	"fmt"
)

type Arg struct {
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
	// text of placeholders.  The placeholder is used in the synoposis for the argument as well
	// as error messages.
	HelpText string

	// UsageText provides the usage for the argument.  If left blank, a succint synopsis
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
	// If unspecified, the value will be a string pointer.
	Value interface{}

	// DefaultText provides a description of the detault value for the argument.  This is displayed
	// on help screens but is otherwise unused
	DefaultText string

	// NArg describes how many values are passed to the argument.  For a description, see
	// ArgCount function
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

	option *internalOption
	flags  internalFlags
	uses   *actionPipelines
}

//counterfeiter:generate . ArgCounter

// ArgCounter provides the behavior of counting
type ArgCounter interface {
	// Take considers the argument and and returns whether it can be used.
	// If the error EndOfArguments is returned, then the arg counter is done with
	// taking argumens.  All other errors are treated as fatal.
	Take(arg string, possibleFlag bool) error

	// Done is invoked to signal the end of arguments
	Done() error
}

type argContext struct {
	option *Arg
	args_  []string
}

type discreteCounter struct {
	total int
	count int
}

type optionalCounter struct {
	seen bool
}

type varArgsCounter struct {
	stopOnFlags bool
}

type argSynopsis struct {
	value string
	multi bool
}

const (
	// TakeRemaining is the value to use for Arg.NArg to indicate that an argument takes
	// all the remaining tokens from the command line
	TakeRemaining = -1

	// TakeRemaining is the value to use for Arg.NArg to indicate that an argument takes
	// tokens from the command line until one looks like a flag.
	TakeUntilNextFlag = -2
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
//  >= 1   take exactly n number of arguments, though if they look like flags treat as an error
//     0   take argument if it does not look like a flag
//    -1   take all remaining arguments (even when they look like flags) (TakeRemaining)
//    -2   take all remaining arguments but stop before taking one that looks like a flag (TakeUntilNextFlag)
//
// Any other negative value uses the behavior of -1.
//
func ArgCount(v interface{}) ArgCounter {
	switch count := v.(type) {
	case ArgCounter:
		return count
	case int:
		if count > 0 {
			return &discreteCounter{count, count}
		}
		if count == 0 {
			return &optionalCounter{}
		}
		return &varArgsCounter{
			stopOnFlags: count == -2,
		}
	case nil:
		return ArgCount(0)
	default:
		panic(fmt.Sprintf("unexpected type: %T", v))
	}
}

// Occurrences counts the number of times that the argument has occurred on the command line
func (a *Arg) Occurrences() int {
	if a == nil || a.option == nil {
		return 0
	}
	return a.option.Count()
}

// Seen reports true if the argument is used at least once.
func (a *Arg) Seen() bool {
	if a == nil || a.option == nil {
		return false
	}
	return a.option.Count() > 0
}

// Set will set the value of the argument
func (a *Arg) Set(arg string) error {
	return a.option.Value().Set(arg, a.option)
}

// SetHidden causes the argument to be hidden from the help screen
func (a *Arg) SetHidden() {
	a.flags |= internalFlagHidden
}

// SetRequired will indicate that the argument is required.
func (a *Arg) SetRequired() {
	a.flags |= internalFlagRequired
}

// Synopsis contains the value placeholder
func (a *Arg) Synopsis() string {
	return textUsage.arg(a.newSynopsis())
}

func (a *Arg) newSynopsis() *argSynopsis {
	return a.newSynopsisCore(fmt.Sprintf("<%s>", a.Name))
}

func (a *Arg) newSynopsisCore(defaultUsage string) *argSynopsis {
	usage := a.UsageText
	if usage == "" {
		usage = defaultUsage
	}
	return &argSynopsis{
		value: usage,
		multi: isMulti(a.NArg),
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

func (a *Arg) wrapAction(fn func(Action) ActionFunc) {
	a.Action = fn(ActionOf(a.Action))
}

func (a *Arg) applyToSet(s *set) {
	a.option = s.defineArg(a.Name, a.value(), a.NArg)
}

func (a *Arg) action() Action {
	return ActionOf(a.Action)
}

func (a *Arg) before() Action {
	return ActionOf(a.Before)
}

func (a *Arg) after() Action {
	return ActionOf(a.After)
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
	a.Value = ensureDestination(a.Value, isMulti(a.NArg))
	return a.Value
}

// SetData sets the specified metadata on the argument
func (a *Arg) SetData(name string, v interface{}) {
	a.ensureData()[name] = v
}

func (a *Arg) setCategory(name string) {}

func (a *Arg) ensureData() map[string]interface{} {
	if a.Data == nil {
		a.Data = map[string]interface{}{}
	}
	return a.Data
}

func (a *Arg) hooks() *hooks {
	return nil
}

func (a *Arg) appendAction(t timing, ah Action) {
	a.uses.add(t, ah)
}

func (o *argContext) hooks() *hooks {
	return nil
}

func (o *argContext) initialize(c *Context) error {
	rest := takeInitializers(ActionOf(o.option.Uses), o.option.Options, c)
	o.option.uses = rest
	return executeAll(c, rest.Initializers, defaultOption.Initializers)
}

func (o *argContext) executeBefore(ctx *Context) error {
	tt := o.option
	return executeAll(ctx, tt.uses.Before, tt.before(), defaultOption.Before)
}

func (o *argContext) executeBeforeDescendent(ctx *Context) error { return nil }
func (o *argContext) executeAfterDescendent(ctx *Context) error  { return nil }
func (o *argContext) executeAfter(ctx *Context) error {
	tt := o.option
	return executeAll(ctx, tt.uses.After, tt.after(), defaultOption.After)
}
func (o *argContext) execute(ctx *Context) error { return nil }
func (o *argContext) app() (*App, bool)          { return nil, false }
func (o *argContext) args() []string             { return o.args_ }
func (o *argContext) set() *set                  { return nil }
func (o *argContext) target() target             { return o.option }
func (o *argContext) setDidSubcommandExecute()   {}
func (o *argContext) lookupValue(name string) (interface{}, bool) {
	if name == "" {
		return o.option.value(), true
	}
	return nil, false
}
func (o *argContext) Name() string {
	return o.option.name()
}

func (d *discreteCounter) Take(arg string, possibleFlag bool) error {
	if d.count == 0 {
		return EndOfArguments
	}
	d.count -= 1
	return nil
}

func (d *discreteCounter) Done() error {
	if d.count > 0 {
		return expectedArgument(d.total)
	}
	return nil
}

func (d *optionalCounter) Take(arg string, possibleFlag bool) error {
	if d.seen {
		return EndOfArguments
	}
	d.seen = true
	return nil
}

func (d *optionalCounter) Done() error {
	return nil
}

func (v *varArgsCounter) Take(arg string, possibleFlag bool) error {
	if v.stopOnFlags && allowFlag(arg, possibleFlag) {
		return EndOfArguments
	}
	return nil
}

func (*varArgsCounter) Done() error {
	return nil
}

func findArgByName(items []*Arg, name string) (*Arg, bool) {
	for _, sub := range items {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}

func isMulti(narg interface{}) bool {
	if narg, ok := narg.(int); ok {
		return narg < 0 || narg > 1
	}
	return false
}
