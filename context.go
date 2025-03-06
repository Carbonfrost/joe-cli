package cli

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"
)

// Context provides the context in which the app, command, or flag is executing or initializing.
// Context is used to faciliate interactions with the Joe-cli application context
// that is currently being initialized or executed.  It wraps context.Context and
// can be obtained from FromContext
type Context struct {

	// Stdout is the output writer to Stdout
	Stdout Writer

	// Stderr is the output writer to Stderr
	Stderr Writer

	// Stdin is the input reader
	Stdin io.Reader

	// FS is the file system used by the context.
	// If the FS implements func OpenContext(context.Context, string)(fs.File, error), note that
	// this will be called instead of Open in places where the Context is available.
	// For os.File this means that if the context has a Deadline, SetReadDeadline
	// and/or SetWriteDeadline will be set.  Clients can implement similar functions in their
	// own fs.File implementations provided from an FS implementation.
	FS fs.FS

	lookupCore
	internal internalContext
	timing   Timing
	request  *CompletionRequest

	parent    *Context
	pathCache ContextPath
	ref       context.Context
}

// WalkFunc provides the callback for the Walk function
type WalkFunc func(cmd *Context) error

type internalContext interface {
	lookupCore
	lookupBinding(name string, occurs bool) []string
	initialize(context.Context) error
	initializeDescendent(context.Context) error
	executeBeforeDescendent(context.Context) error
	executeBefore(context.Context) error
	executeAfter(context.Context) error
	executeAfterDescendent(context.Context) error
	execute(context.Context) error
	target() target // *Command, *Arg, *Flag, or *Expr
	Name() string
}

type internalCommandContext interface {
	internalContext
	set() BindingLookup
}

type parentLookup struct {
	lookupCore // delegates to the internal context
	parent     lookupCore
}

type valueTarget struct {
	pipelinesSupport

	v    any
	name string
}

// ContextPath provides a list of strings that name each one of the parent components
// in the context.  Each string follows the form:
//
//	command  a command matching the name "command"
//	-flag    a flag matching the flag name
//	<arg>    an argument matching the arg name
type ContextPath []string

type contextPathPattern struct {
	parts []string
}

// contextKeyType is the type of the key referencing *Context
type contextKeyType struct{}

const (
	synopsisKey = "_Synopsis"
)

var (
	// SkipCommand is used as a return value from WalkFunc to indicate that the command in the call is to be skipped.
	// This is also used to by ExecuteSubcommand (or HandleCommandNotFound) to indicate that no command should
	// be executed.
	SkipCommand = errors.New("skip this command")

	// ErrTimingTooLate occurs when attempting to run an action in a pipeline
	// when the pipeline is later than requested by the action.
	ErrTimingTooLate = errors.New("too late for requested action timing")
)

// FromContext obtains the Context, which faciliates interactions with the application
// that is initializing or running. If the argument is nil, the return value will be.
// Otherwise, if it can't be found, it panics
func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, ok := fromContext(ctx)
	if !ok {
		panic("ctx does not provide *cli.Context")
	}
	return c
}

func fromContext(ctx context.Context) (*Context, bool) {
	if ctx == nil {
		return nil, false
	}
	var key contextKeyType
	c, ok := ctx.Value(key).(*Context)
	return c, ok
}

func newContextPathPattern(pat string) contextPathPattern {
	return contextPathPattern{strings.Fields(pat)}
}

func newValueTarget(v any, name string, action Action) *valueTarget {
	return &valueTarget{
		v:    v,
		name: name,
		pipelinesSupport: pipelinesSupport{
			actionPipelines{
				Initializers: action,
			},
		},
	}
}

// Execute executes the context with the given arguments.
func (c *Context) Execute(args []string) error {
	if cmd, ok := c.target().(*Command); ok {
		if cmd.fromApp != nil {
			defer provideCurrentApp(cmd.fromApp)()
		}
	}
	res := c.parse(args)
	if res.err != nil {
		return res.err
	}

	if err := c.executeBefore(); err != nil {
		return err
	}

	if err := c.executeSelf(); err != nil {
		return err
	}

	return c.executeAfter()
}

func (c *Context) parse(args []string) *robustParseResult {
	root := c.Command()
	set := root.buildSet(c)
	if root.internalFlags().skipFlagParsing() {
		args = append([]string{args[0], "--"}, args[1:]...)
	}

	flags := root.internalFlags().toRaw() | RawSkipProgramName
	err := set.parse(args, flags)
	return &robustParseResult{bindings: set.bindings, err: err}
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Context().Deadline()
}

func (c *Context) Done() <-chan struct{} {
	return c.Context().Done()
}

func (c *Context) Err() error {
	return c.Context().Err()
}

func (c *Context) Context() context.Context {
	return c.ref
}

// SetContext sets the context
func (c *Context) SetContext(ctx context.Context) error {
	c.ref = ctx
	return nil
}

// SetContextValue updates the context with a value.
func (c *Context) SetContextValue(key, value any) error {
	return c.SetContext(context.WithValue(c.Context(), key, value))
}

// Matches detects if the given context filter matches the current
// context
func (c *Context) Matches(f ContextFilter) bool {
	if f == nil {
		return true
	}
	return f.Matches(c)
}

// Parent obtains the parent context or nil if the root context
func (c *Context) Parent() *Context {
	if c == nil {
		return nil
	}
	return c.parent
}

// App obtains the app
func (c *Context) App() *App {
	if app, ok := c.app(); ok {
		return app
	}
	return c.Parent().App()
}

func (c *Context) app() (*App, bool) {
	if root, ok := c.internal.(*commandContext); ok {
		a := root.cmd.fromApp
		return a, a != nil
	}
	return nil, false
}

// Root obtains the root command.
func (c *Context) Root() *Context {
	if c.Parent() == nil {
		return c
	}

	return c.Parent().Root()
}

func (c *Context) root() *rootCommandData {
	return c.Root().target().(*Command).rootData()
}

// Command obtains the command.  The command could be a synthetic command that was
// created to represent the root command of the app.
func (c *Context) Command() *Command {
	if c == nil {
		return nil
	}
	if cmd, ok := c.target().(*Command); ok {
		return cmd
	}
	return c.Parent().Command()
}

// Arg retrieves the argument in scope if any
func (c *Context) Arg() *Arg {
	if c == nil {
		return nil
	}
	if a, ok := c.target().(*Arg); ok {
		return a
	}
	return c.Parent().Arg()
}

// Flag retrieves the flag in scope if any
func (c *Context) Flag() *Flag {
	if c == nil {
		return nil
	}
	if f, ok := c.target().(*Flag); ok {
		return f
	}
	return c.Parent().Flag()
}

// IsInitializing returns true if the context represents initialization
func (c *Context) IsInitializing() bool { return c.timing == InitialTiming }

// IsBefore returns true if the context represents actions running before executing the command
func (c *Context) IsBefore() bool { return c.timing == BeforeTiming }

// IsAfter returns true if the context represents actions running after executing the command
func (c *Context) IsAfter() bool { return c.timing == AfterTiming }

// IsAction returns true if the context represents actions running for the actual execution of the command
func (c *Context) IsAction() bool { return c.timing == ActionTiming }

// Timing retrieves the timing
func (c *Context) Timing() Timing {
	return c.timing
}

// HasValue tests whether the target has a value
func (c *Context) HasValue() bool {
	_, ok := c.target().(*valueTarget)
	return c.isOption() || ok
}

// IsCommand tests whether the last segment of the path represents a command
func (c *Context) IsCommand() bool {
	_, ok := c.target().(*Command)
	return ok
}

// IsFlag tests whether the last segment of the path represents a flag
func (c *Context) IsFlag() bool {
	_, ok := c.target().(*Flag)
	return ok
}

// IsArg tests whether the last segment of the path represents an argument
func (c *Context) IsArg() bool {
	_, ok := c.target().(*Arg)
	return ok
}

func (c *Context) isOption() bool {
	_, ok := c.target().(option)
	return ok
}

func (c *Context) subcommandDidNotExecute() bool {
	return !c.target().internalFlags().didSubcommandExecute()
}

// Args retrieves the arguments.  IF the context corresponds to a command, these
// represent the name of the command plus the arguments passed to it.  For flags and arguments,
// this is the value passed to them
func (c *Context) Args() []string {
	return c.internal.lookupBinding("", false)
}

// Flags obtains the flags from the command, including
// persistent flags which were defined by ancestor commands.
// If the current context is not a command, this is nil.
// Compare Flags, PersistentFlags, and LocalFlags.
func (c *Context) Flags() []*Flag {
	if !c.IsCommand() {
		return nil
	}
	return append(c.PersistentFlags(), c.LocalFlags()...)
}

// LocalFlags obtains the flags from the command.  If the current
// context is not a command, this is nil.
// Compare Flags, PersistentFlags, and LocalFlags.
func (c *Context) LocalFlags() []*Flag {
	if !c.IsCommand() {
		return nil
	}
	return c.Target().(*Command).Flags
}

// LocalArgs obtains the args from the command or value target.  If the current context
// is not a command or value target, this is nil.
func (c *Context) LocalArgs() []*Arg {
	if c.IsCommand() {
		return c.Command().Args
	}
	if aa, ok := c.Target().(interface{ LocalArgs() []*Arg }); ok {
		return aa.LocalArgs()
	}
	return nil
}

// PersistentFlags locates the nearest command and obtains flags
// from its parent and ancestor commands.  If the current
// context is not a command, this is nil.
// Compare Flags, PersistentFlags,  and LocalFlags.
func (c *Context) PersistentFlags() []*Flag {
	if !c.IsCommand() {
		return nil
	}
	if c.Parent() == nil {
		return nil
	}
	return c.Parent().Flags()
}

// ValueContextOf creates a context for the use with values that
// may have initialization steps.
func (c *Context) ValueContextOf(name string, v any) *Context {
	switch v.(type) {
	case *Arg, *Flag, *Command, *App:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
	return c.valueContext(newValueTarget(v, name, nil))
}

// ContextOf creates a context for use with the given target, which must
// be a flag, arg, or sub-command, or it must be the name of a flag or arg
// (using the typing and rules of LookupFlag and LookupArg). This method
// does not check whether the target is actually in the scope of the
// current target. The result could be nil if the name or index does not
// exist.
func (c *Context) ContextOf(target any) *Context {
	if target == "" {
		return c
	}
	if !c.IsCommand() {
		return c.Parent().ContextOf(target)
	}
	switch name := target.(type) {
	case *Arg, *Flag:
		return c.optionContext(target.(option))
	case *Command:
		return c.commandContext(target.(*Command))
	case int:
		return c.tryOptionContext(c.LookupArg(name))
	case rune:
		return c.tryOptionContext(c.LookupFlag(name))
	case string:
		if a, ok := c.LookupArg(name); ok {
			return c.optionContext(a)
		}
		return c.tryOptionContext(c.LookupFlag(name))
	default:
		panic(fmt.Sprintf("unexpected target type %T", target))
	}
}

func (c *Context) tryOptionContext(target any, ok bool) *Context {
	if !ok {
		return nil
	}
	return c.optionContext(target.(option))
}

// LookupCommand finds the command by name.  The name can be a string or *Command
func (c *Context) LookupCommand(name interface{}) (*Command, bool) {
	if c == nil {
		return nil, false
	}
	switch v := name.(type) {
	case string:
		if v == "" {
			return c.Command(), true
		}
		if aa, ok := c.target().(*Command); ok {
			if r, _, found := findCommandByName(aa.Subcommands, v); found {
				return r, true
			}
		}
	case *Command:
		return v, v != nil
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}

	return nil, false
}

// LookupFlag finds the flag by name.  The name can be a string, rune, or *Flag
func (c *Context) LookupFlag(name interface{}) (*Flag, bool) {
	if c == nil {
		return nil, false
	}
	switch v := name.(type) {
	case rune:
		return c.LookupFlag(string(v))
	case string:
		if v == "" {
			if c.isOption() {
				return c.LookupFlag(c.option())
			}
			name = c.Name()
		}
		if aa, ok := c.target().(*Command); ok {
			if f, _, found := findFlagByName(aa.Flags, v); found {
				return f, true
			}
		}
	case *Flag:
		return v, v != nil
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}

	return c.Parent().LookupFlag(name)
}

// LookupArg finds the arg by name.  The name can be a string, rune, or *Arg
func (c *Context) LookupArg(name interface{}) (*Arg, bool) {
	if c == nil {
		return nil, false
	}
	switch v := name.(type) {
	case int:
		return c.LookupArg(c.logicalArg(v))
	case string:
		if v == "" {
			if c.isOption() {
				return c.LookupArg(c.option())
			}
			name = c.Name()
		}
		if a, _, found := findArgByName(c.LocalArgs(), v); found {
			return a, true
		}
	case *Arg:
		return v, v != nil
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}
	return c.Parent().LookupArg(name)
}

// FindTarget finds the given target corresponding to the context path.
func (c *Context) FindTarget(path ContextPath) (res *Context, ok bool) {
	res = c
	ok = true
	if len(path) == 0 || path[0] == "" {
		return res, true
	}
	if !matchField(path[0], res.Name()) {
		return res, false
	}
	for _, p := range path[1:] {
		res, ok = res.findTarget(p)
		if !ok {
			break
		}
	}
	return
}

func (c *Context) findTarget(name string) (*Context, bool) {
	switch {
	case name == "":
		return c, false
	case matchFlag(name):
		if f, ok := c.LookupFlag(name); ok {
			return c.optionContext(f), true
		}
	case matchArg(name):
		if a, ok := c.LookupArg(name); ok {
			return c.optionContext(a), true
		}
	default:
		if m, ok := c.LookupCommand(name); ok {
			return c.commandContext(m), true
		}
	}
	return nil, false
}

// Seen returns true if the specified flag or argument has been used at least once.
// This is false in the context of commands.
func (c *Context) Seen(name interface{}) bool {
	f, ok := c.lookupOption(name)
	return ok && f.Seen()
}

// Occurrences returns the number of times the specified flag or argument has been used
// This is -1 in the context of commands.
func (c *Context) Occurrences(name interface{}) int {
	if f, ok := c.lookupOption(name); ok {
		return f.Occurrences()
	}
	return -1
}

// ImplicitlySet returns true if the flag or arg was implicitly
// set.
func (c *Context) ImplicitlySet() bool {
	return c.target().internalFlags().seenImplied()
}

// Expression obtains the expression from the context
func (c *Context) Expression(name string) *Expression {
	return c.Value(name).(*Expression)
}

// NValue gets the maximum number available, exclusive, as an argument Value.
func (c *Context) NValue() int {
	return len(c.LocalArgs())
}

// Values gets all the values from the context .
func (c *Context) Values() []interface{} {
	res := make([]interface{}, c.NValue())
	for i := 0; i < len(res); i++ {
		res[i] = c.Value(i)
	}
	return res
}

// Value obtains the value of the flag or argument with the specified name.  If name
// is the empty string or nil, this is interpreted as using the name of whatever is the
// current context flag or argument.  The name can also be one of several other types:
//
//   - rune - corresponds to the short name of a flag
//   - int - obtain the argument by index
//   - *Arg - get value of the arg
//   - *Flag - get value of the flag
//
// All other types are delegated to the underlying Context.  This implies that you can only
// use your own (usually unexported) hashable type when setting up keys in the
// context.Context.  (This is the recommended practice in any case, but it is made explicit
// by how this method works.)
func (c *Context) Value(name interface{}) interface{} {
	if c == nil {
		return nil
	}

	switch v := name.(type) {
	case rune, string, nil, *Arg, *Flag, int:
		it, _ := tryLookup(c.lookupCore, c.nameToString(v), true)
		return it
	case contextKeyType:
		return c
	default:
		return c.Context().Value(name)
	}
}

// Raw gets the exact value which was passed to the arg, flag including
// the name that was used.  Value can be the empty string if no value
// was passed
func (c *Context) Raw(name interface{}) []string {
	return c.rawCore(name, false)
}

// RawOccurrences gets the exact value which was passed to the arg, flag
// excluding the name that was used.  Value can be the empty string if no value
// was passed
func (c *Context) RawOccurrences(name interface{}) []string {
	return c.rawCore(name, true)
}

func (c *Context) rawCore(name interface{}, occurs bool) []string {
	if c == nil {
		return []string{}
	}

	switch f := name.(type) {
	case rune, string, nil, int, *Arg, *Flag:
		d := c.internal.lookupBinding(c.nameToString(f), occurs)
		if f != "" && len(d) == 0 {
			return c.Parent().rawCore(c.nameToString(f), occurs)
		}

		return d

	default:
		return []string{}
	}
}

// BindingLookup returns the parse data for the context.  Note that this lookup
// only applies to the current context and does not traverse inherited contexts.
// Compare Raw and RawOccurrences which perform this for you.
func (c *Context) BindingLookup() BindingLookup {
	if c == nil {
		return nil
	}
	if cc, ok := c.internal.(internalCommandContext); ok {
		return cc.set()
	}
	return c.Parent().BindingLookup()
}

// LookupData gets the data matching the key, including recursive traversal
// up the lineage contexts
func (c *Context) LookupData(name string) (interface{}, bool) {
	if c == nil || c.target() == nil {
		return nil, false
	}
	if res, ok := c.target().LookupData(name); ok {
		return res, true
	}
	return c.Parent().LookupData(name)
}

// Aliases obtains the aliases for the current command or flag; otherwise,
// for args it is nill
func (c *Context) Aliases() []string {
	switch t := c.Target().(type) {
	case *Command:
		return t.Aliases
	case *Flag:
		return t.Aliases
	}
	return nil
}

// AddAlias adds one or more aliases to the current command or flag.
// For other targets, this operation is ignored.  An error is returned
// if this is called after initialization.
func (c *Context) AddAlias(aliases ...string) error {
	return c.updateAliases(func(aa []string) []string {
		return append(aa, aliases...)
	})
}

// RemoveAlias removes the given alias from the current command or
// flag. For other targets, this operation is ignored. An error is
// returned if this is caleld after initialization.
func (c *Context) RemoveAlias(a string) error {
	return c.updateAliases(func(aliases []string) []string {
		return slices.DeleteFunc(aliases, func(s string) bool {
			return s == a
		})
	})
}

func (c *Context) updateAliases(fn func([]string) []string) error {
	err := c.requireInit()
	if err != nil {
		return err
	}
	switch t := c.Target().(type) {
	case *Command:
		t.Aliases = fn(t.Aliases)
	case *Flag:
		t.Aliases = fn(t.Aliases)
	}
	return nil
}

// SetTransform sets the transform for the current flag or arg.
// For other targets, this operation is ignored.  An error is returned
// if this is called after initialization.
func (c *Context) SetTransform(fn TransformFunc) error {
	err := c.requireInit()
	if err != nil {
		return err
	}
	if option, ok := c.target().(option); ok {
		option.setTransform(fn)
	}

	return nil
}

// Data obtains the data for the current target.  This could be a nil map.
func (c *Context) Data() map[string]any {
	return c.target().data()
}

// Category obtains the category for the current target.
func (c *Context) Category() string {
	return c.target().category()
}

// Description obtains the description for the current target.
func (c *Context) Description() any {
	return c.target().description()
}

// HelpText obtains the helpText for the current target.
func (c *Context) HelpText() string {
	return c.target().helpText()
}

// UsageText obtains the usageText for the current target.
func (c *Context) UsageText() string {
	return c.target().usageText()
}

// ManualText obtains the manualText for the current target.
func (c *Context) ManualText() string {
	return c.target().manualText()
}

// Completion obtains the completion for the current target.
func (c *Context) Completion() Completion {
	return c.target().completion()
}

// SetData sets data on the current target.  Despite the return value,
// this method never returns an error.
func (c *Context) SetData(name string, v interface{}) error {
	c.target().SetData(name, v)
	return nil
}

func (c *Context) nameToString(name interface{}) string {
	switch v := name.(type) {
	case rune, string, nil, *Arg, *Flag:
		return nameToString(name)
	case int:
		return c.logicalArg(v).Name
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}
}

// SetOptionalValue sets the optional value for the flag
func (c *Context) SetOptionalValue(v any) error {
	err := c.requireInit()
	if err != nil {
		return err
	}
	c.Flag().setOptionalValue(v)
	return nil
}

// SetName sets the name on the current target
func (c *Context) SetName(name string) error {
	switch o := c.target().(type) {
	case *Arg:
		o.Name = name
	case *Flag:
		o.Name = name
	case *Command:
		o.Name = name
	case interface{ SetName(string) }:
		o.SetName(name)
	}
	return nil
}

// SetCategory sets the category on the current target
func (c *Context) SetCategory(name string) error {
	c.target().setCategory(name)
	return nil
}

// SetDescription sets the description on the current target
func (c *Context) SetDescription(v interface{}) error {
	c.target().setDescription(v)
	return nil
}

// SetHelpText sets the help text on the current target
func (c *Context) SetHelpText(s string) error {
	c.target().setHelpText(s)
	return nil
}

// SetUsageText sets the usage text on the current target
func (c *Context) SetUsageText(s string) error {
	c.target().setUsageText(s)
	return nil
}

// SetManualText sets the manualText on the current target
func (c *Context) SetManualText(v string) error {
	c.target().setManualText(v)
	return nil
}

// SetValue checks the timing and sets the value of the current
// flag or arg.  This method can be called at any time to set
// the value; however, if the current timing is the ImplicitValueTiming,
// calling this method will return ErrImplicitValueAlreadySet for
// the second and subsequent invocations of this method.  Clients
// typically ignore this error and don't bubble it up as a usage error,
// or they can check ImplicitlySet() to preempt it.  Note that you can
// always set the value on the Arg or Flag directly with no checks
// for these timing semantics.
func (c *Context) SetValue(arg any) error {
	if c.implicitTimingActive() {
		if c.target().internalFlags().seenImplied() {
			return ErrImplicitValueAlreadySet
		}

		c.target().setInternalFlags(internalFlagSeenImplied, true)
	}
	return c.target().(option).Set(arg)
}

// At either stores or executes the action at the given timing.
func (c *Context) At(t Timing, v Action) error {
	return Do(c, At(t, v))
}

// Action either stores or executes the action. When called from the initialization or before pipelines, this
// appends the action to the pipeline for the current flag, arg, or command/app.
// When called from the action pipeline, this simply causes the action to be invoked immediately.
func (c *Context) Action(action Action) error {
	return c.At(ActionTiming, action)
}

// Before either stores or executes the action.  When called from the initialization pipeline, this appends
// the action to the Before pipeline for the current flag, arg, expression, or command/app.  If called
// from the Before pipeline, this causes the action to be invoked immediately.  If called
// at any other time, this causes the action to be ignored and an error to be returned.
func (c *Context) Before(action Action) error {
	return c.At(BeforeTiming, action)
}

// After either stores or executes the action.  When called from the initialization, before, or action pipelines,
// this appends the action to the After pipeline for the current flag, arg, expression, or command/app.  If called
// from the After pipeline itself, the action is invoked immediately
func (c *Context) After(action Action) error {
	return c.At(AfterTiming, action)
}

// Use can only be used during initialization timing, in which case the action is just invoked.  In other timings,
// this is an error
func (c *Context) Use(action Action) error {
	return c.At(InitialTiming, action)
}

func (c *Context) act(v interface{}, desired Timing, optional bool) error {
	// For the purposes of determining whether we can run this action,
	// remove synthetic timing
	actual := desired
	if desired > syntheticTiming {
		actual = BeforeTiming
	}
	if c.timing < actual {
		c.target().appendAction(desired, ActionOf(v))
		return nil
	}
	if c.timing == actual {
		return ActionOf(v).Execute(c)
	}
	if optional {
		return nil
	}
	if c.timing > actual {
		return c.internalError(ErrTimingTooLate)
	}
	return nil
}

// Target retrieves the target of the context, which is *App, *Command, *Flag, *Arg,
// or *Expr
func (c *Context) Target() any {
	if val, ok := c.target().(*valueTarget); ok {
		return val.v
	}
	return c.target()
}

func (c *Context) target() target {
	if c == nil {
		return nil
	}
	return c.internal.target()
}

// Hook registers a hook that runs for any context in the given timing.
func (c *Context) Hook(timing Timing, handler Action) error {
	if h, ok := c.hookable(); ok {
		if c.Timing() <= timing {
			return h.hook(timing, handler)
		}

		return c.internalError(fmt.Errorf("%s: %w", errCantHook, ErrTimingTooLate))
	}
	return errCantHook
}

// HookBefore registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func (c *Context) HookBefore(pattern string, handler Action) error {
	return c.hookAt(BeforeTiming, pattern, handler)
}

// HookAfter registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func (c *Context) HookAfter(pattern string, handler Action) error {
	return c.hookAt(AfterTiming, pattern, handler)
}

func (c *Context) hookAt(timing Timing, pattern string, handler Action) error {
	// Specifying the empty string for the pattern is the same as acting
	// on itself and therefore does not need a hook
	if pattern == "" {
		return c.At(timing, handler)
	}
	return c.Hook(timing, IfMatch(PatternFilter(pattern), handler))
}

func (c *Context) hookable() (hookable, bool) {
	h, ok := c.internal.target().(hookable)
	return h, ok
}

// Walk traverses the hierarchy of commands.  The provided function fn is called once for each command
// that is encountered.  If fn returns an error, the traversal stops and the error is returned; however, the
// specialized return value SkipCommand indicates that traversal skips the sub-commands of the current
// command and continues.
func (c *Context) Walk(fn WalkFunc) error {
	return c.walkCore(fn)
}

func (c *Context) walkCore(fn WalkFunc) error {
	current := c.Command()
	err := fn(c)
	switch err {
	case nil:
		for _, sub := range current.Subcommands {
			if err := c.commandContext(sub).walkCore(fn); err != nil {
				return err
			}
		}
		return nil
	case SkipCommand:
		return nil
	default:
		return err
	}
}

// Do executes the specified actions in succession.  If an action returns an error, that
// error is returned and the rest of the actions aren't run
func (c *Context) Do(action Action) error {
	return Do(c, action)
}

// Template retrieves a template by name
func (c *Context) Template(name string) *Template {
	t := c.root().ensureTemplates().Lookup(name)
	if t == nil {
		return nil
	}
	return &Template{
		Template: t,
		Debug:    debugTemplates(),
	}
}

func withExecute(funcMap template.FuncMap, self *template.Template) template.FuncMap {
	// Execute function needs a closure containing the template itself, so is
	// added afterwards
	funcMap["Execute"] = func(name string, data interface{}) (string, error) {
		buf := bytes.NewBuffer(nil)
		if err := self.ExecuteTemplate(buf, name, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	return funcMap
}

func debugTemplates() bool {
	return os.Getenv("CLI_DEBUG_TEMPLATES") == "1"
}

// Name gets the name of the context, which is the name of the command, arg, flag, or expression
// operator in use
func (c *Context) Name() string {
	if c.internal == nil {
		// Due to tests this could be unset
		return ""
	}
	return c.internal.Name()
}

// Path retrieves all of the names on the context and its ancestors to the root.
// If the root command had no name, it is implied name from os.Args.
func (c *Context) Path() ContextPath {
	if len(c.pathCache) == 0 {
		c.pathCache = c.pathSlow()
	}
	return c.pathCache
}

func (c *Context) pathSlow() ContextPath {
	res := make([]string, 0)
	c.lineageFunc(func(ctx *Context) {
		res = append(res, ctx.Name())
	})

	// Reverse to get the proper order, and remove the root context if present
	res = reverse(res)
	if res[0] == "" {
		res = res[1:]
	}
	return ContextPath(res)
}

// Lineage retrieves all of the ancestor contexts up to the root.  The result
// contains the current context and all contexts up to the root.
func (c *Context) Lineage() []*Context {
	res := make([]*Context, 0)
	c.lineageFunc(func(ctx *Context) {
		res = append(res, ctx)
	})
	return res
}

func (c *Context) lineageFunc(f func(*Context)) {
	current := c
	for current != nil {
		f(current)
		current = current.Parent()
	}
}

func (c *Context) logicalArg(index int) *Arg {
	args := c.LocalArgs()
	_, idx, _ := findArgByName(args, index)
	if idx < 0 || idx >= len(args) {
		return nil
	}
	return args[idx]
}

// AddFlag provides a convenience method that adds a flag to the current command or app.  This
// is only valid during the initialization phase.  An error is returned for other timings.
func (c *Context) AddFlag(f *Flag) error {
	return c.updateFlags(func(flags []*Flag) []*Flag {
		return append(flags, f)
	})
}

// AddCommand provides a convenience method that adds a Command to the current command or app.  This
// is only valid during the initialization phase.  An error is returned for other timings.
func (c *Context) AddCommand(v *Command) error {
	return c.updateSubcommands(func(cmds []*Command) []*Command {
		return append(cmds, v)
	})
}

// AddArg provides a convenience method that adds an Arg to the current command or app.  This
// is only valid during the initialization phase.  An error is returned for other timings.
func (c *Context) AddArg(v *Arg) error {
	return c.updateArgs(func(args []*Arg) []*Arg {
		return append(args, v)
	})
}

// RemoveArg provides a convenience method that removes an Arg from the current command or app.
// The name specifies the name, index, or actual arg.  This
// is only valid during the initialization phase.  An error is returned for other timings.
// If the arg does not exist, if the name or index is out of bounds, the operation
// will still succeed.
func (c *Context) RemoveArg(name interface{}) error {
	return c.updateArgs(func(args []*Arg) []*Arg {
		if _, index, ok := findArgByName(args, name); ok {
			return append(args[0:index], args[index+1:]...)
		}
		return args
	})
}

// RemoveCommand provides a convenience method that removes a command from the current command or app.
// The name specifies the name or actual command.  This
// is only valid during the initialization phase.  An error is returned for other timings.
// If the Command does not exist, if the name or index is out of bounds, the operation
// will still succeed.
func (c *Context) RemoveCommand(name interface{}) error {
	return c.updateSubcommands(func(cmds []*Command) []*Command {
		if _, index, ok := findCommandByName(cmds, name); ok {
			return slices.Delete(cmds, index, index+1)
		}
		return cmds
	})
}

// RemoveFlag provides a convenience method that removes a Flag from the current command or app.
// The name specifies the name, index, or actual flag.  This
// is only valid during the initialization phase.  An error is returned for other timings.
// If the flag does not exist, if the name or index is out of bounds, the operation
// will still succeed.
func (c *Context) RemoveFlag(name any) error {
	return c.updateFlags(func(flags []*Flag) []*Flag {
		if _, index, ok := findFlagByName(flags, name); ok {
			return slices.Delete(flags, index, index+1)
		}
		return flags
	})
}

// AddFlags provides a convenience method for adding flags to the current command or app.
func (c *Context) AddFlags(flags ...*Flag) (err error) {
	for _, f := range flags {
		if err = c.AddFlag(f); err != nil {
			break
		}
	}
	return
}

// AddCommands provides a convenience method for adding commands to the current command or app.
func (c *Context) AddCommands(commands ...*Command) (err error) {
	for _, cmd := range commands {
		if err = c.AddCommand(cmd); err != nil {
			break
		}
	}
	return
}

// AddArgs provides a convenience method for adding args to the current command or app.
func (c *Context) AddArgs(args ...*Arg) (err error) {
	for _, a := range args {
		if err = c.AddArg(a); err != nil {
			break
		}
	}
	return
}

func (c *Context) updateFlags(fn func([]*Flag) []*Flag) error {
	err := c.requireInit()
	if err != nil {
		return err
	}
	cmd := c.Command()
	cmd.Flags = fn(cmd.Flags)
	return nil
}

func (c *Context) updateArgs(fn func([]*Arg) []*Arg) error {
	err := c.requireInit()
	if err != nil {
		return err
	}
	cmd := c.Command()
	cmd.Args = fn(cmd.Args)
	return nil
}

func (c *Context) updateSubcommands(fn func([]*Command) []*Command) error {
	err := c.requireInit()
	if err != nil {
		return err
	}
	cmd := c.Command()
	cmd.Subcommands = fn(cmd.Subcommands)
	return nil
}

// SkipImplicitSetup gets whether implicit setup steps should be skipped
func (c *Context) SkipImplicitSetup() bool {
	return c.flagSetOrAncestor((internalFlags).taintSetup)
}

// PreventSetup causes implicit setup options to be skipped.  The function
// returns an error if the timing is not initial timing.
func (c *Context) PreventSetup() error {
	return c.Use(setInternalFlag(internalFlagTaintSetup))
}

// SetColor sets whether terminal color and styles are enabled on stdout.
func (c *Context) SetColor(v bool) {
	c.Stderr.SetColorCapable(v)
	c.Stdout.SetColorCapable(v)
}

// AutodetectColor causes terminal color and styles to automatically
// detect support for stdout.  Auto-detection is the default behavior, but this
// is provided to reset if SetColor has modified.
func (c *Context) AutodetectColor() {
	c.Stderr.ResetColorCapable()
	c.Stdout.ResetColorCapable()
}

// Trigger marks a flag or arg action to be triggered. Generally,
// a flag or arg Action is only triggered if it is set or if it or its
// parent command is marked with ImpliedAction. To mark the Action to be triggered for some
// other reason, you can use this action.
func (c *Context) Trigger() error {
	return Do(c, Trigger)
}

// ProvideValueInitializer causes an additional child context to be created
// which is used to initialize an arbitrary value.  Typically, the value is
// the value of the flag or arg.  Indeed, a pattern is to expose this action as
// the return value of a method Initializer()Action (refer to the overview in [Value]
// about methods implemented by values by convention).
//
// The value can also provide methods such as SetDescription(string),
// SetHelpText(string), etc. in order to operate with actions that set these values.
//
// If the value has local args (a method LocalArgs() []*Arg), then their
// pipelines are triggered.
func (c *Context) ProvideValueInitializer(v any, name string, actionopt ...Action) error {
	h, ok := c.hookable()
	if !ok {
		return c.internalError(errCantHook)
	}

	adapter := newValueTarget(v, name, ActionPipeline(actionopt))
	h.addValueTarget(adapter)
	return nil
}

// Customize matches a flag, arg, or command and runs additional pipeline steps.  Customize
// is usually used to apply further customization after an extension has done setup of
// the defaults.  As a special case, if pattern is the empty string, this is the same as
// calling Use, because it pertains to the current target.
//
// During the initialization process, the customization action will be run for each matching
// flag, arg, or command.  As a special case, if the flag or arg is created or changes its
// name during its own initialization process, customizations will be re-run on it.
func (c *Context) Customize(pattern string, a Action) error {
	return c.hookAt(InitialTiming, pattern, a)
}

// ReadPasswordString securely gets a password, without the trailing '\n'.
// An error will be returned if the reader is not stdin connected to TTY.
func (c *Context) ReadPasswordString(prompt string) (string, error) {
	return c.displayPrompt(prompt, ReadPasswordString)
}

// ReadString securely gets a password, without the trailing '\n'.
// An error will be returned if the reader is not stdin connected to TTY.
func (c *Context) ReadString(prompt string) (string, error) {
	return c.displayPrompt(prompt, ReadString)
}

func (c *Context) displayPrompt(prompt string, fn func(io.Reader) (string, error)) (string, error) {
	fmt.Fprint(c.Stderr, prompt)
	return fn(c.Stdin)
}

// Print formats using the default formats for its operands and writes to
// standard output using the behavior of fmt.Print.
func (c *Context) Print(a ...any) (n int, err error) {
	return fmt.Fprint(c.Stdout, a...)
}

// Println formats using the default formats for its operands and writes to
// standard output using the behavior of fmt.Println
func (c *Context) Println(a ...any) (n int, err error) {
	return fmt.Fprintln(c.Stdout, a...)
}

// Printf formats according to a format specifier and writes to standard
// output using the behavior of fmt.Printf
func (c *Context) Printf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(c.Stdout, format, a...)
}

// Fprint formats using the default formats for its operands and writes to
// a file using the behavior of fmt.Fprint.
// As a special case, if writer is nil, [Context.Stderr] is used.
func (c *Context) Fprint(w io.Writer, a ...any) (n int, err error) {
	return fmt.Fprint(cmp.Or[io.Writer](w, c.Stderr), a...)
}

// Fprintln formats using the default formats for its operands and writes to
// a file using the behavior of fmt.Fprintln
// As a special case, if writer is nil, [Context.Stderr] is used.
func (c *Context) Fprintln(w io.Writer, a ...any) (n int, err error) {
	return fmt.Fprintln(cmp.Or[io.Writer](w, c.Stderr), a...)
}

// Fprintf formats according to a format specifier and writes to a file
// using the behavior of fmt.Fprintf
// As a special case, if writer is nil, [Context.Stderr] is used.
func (c *Context) Fprintf(w io.Writer, format string, a ...any) (n int, err error) {
	return fmt.Fprintf(cmp.Or[io.Writer](w, c.Stderr), format, a...)
}

func (c *Context) implicitTimingActive() bool {
	return c.flagSetOrAncestor((internalFlags).implicitTimingActive)
}

func (c *Context) flagSetOrAncestor(fn func(internalFlags) bool) bool {
	var result bool
	c.lineageFunc(func(c1 *Context) {
		result = result || fn(c1.target().internalFlags())
	})
	return result
}

// Last gets the last name in the path
func (c ContextPath) Last() string {
	return c[len(c)-1]
}

// IsCommand tests whether the last segment of the path represents a command
func (c ContextPath) IsCommand() bool {
	return matchCommand(c.Last())
}

// IsFlag tests whether the last segment of the path represents a flag
func (c ContextPath) IsFlag() bool {
	return matchFlag(c.Last())
}

// IsArg tests whether the last segment of the path represents an argument
func (c ContextPath) IsArg() bool {
	return matchArg(c.Last())
}

// IsExpr tests whether the last segment of the path represents an expression
func (c ContextPath) IsExpr() bool {
	return matchExpr(c.Last())
}

// String converts the context path to a string
func (c ContextPath) String() string {
	return strings.Join([]string(c), " ")
}

// Match determines if the path matches the given pattern.  Pattern elements:
//
// - *        any command
// - command  a command matching the name "command"
// - -        any flag
// - -flag    a flag matching the flag name
// - <>       any argument
// - <arg>    an argument matching the arg name
// - <->      any expression
// - <-expr>  an expression matching the arg name

func (c ContextPath) Match(pattern string) bool {
	return newContextPathPattern(pattern).Match(c)
}

func (cp contextPathPattern) Matches(c context.Context) bool {
	return cp.Match(FromContext(c).Path())
}

func (cp contextPathPattern) Match(c ContextPath) bool {
	if len(cp.parts) == 0 {
		return true
	}
	for i, j := len(cp.parts)-1, len(c)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if !matchField(cp.parts[i], c[j]) {
			return false
		}
	}
	return true
}

func matchField(pattern, field string) bool {
	if pattern == "" {
		return true
	}
	if pattern == "*" {
		return matchCommand(field)
	}
	if pattern == "-" || pattern == "--" {
		return matchFlag(field)
	}
	if pattern == "<>" {
		return matchArg(field)
	}
	if pattern == "<->" {
		return matchExpr(field)
	}
	if pattern == field {
		return true
	}

	// Special case: long flag names are allowed to use one dash
	if strings.HasPrefix(pattern, "-") && ("-"+pattern) == field {
		return true
	}
	return false
}

func matchCommand(field string) bool {
	return !matchFlag(field) && !matchArg(field) && !matchExpr(field)
}

func matchExpr(field string) bool {
	return strings.HasPrefix(field, "<-")
}

func matchArg(field string) bool {
	return strings.HasPrefix(field, "<") && !strings.HasPrefix(field, "<-")
}

func matchFlag(field string) bool {
	return strings.HasPrefix(field, "-")
}

func rootContext(cctx context.Context, app *App) *Context {
	internal := &commandContext{
		cmd:     app.createRoot(),
		flagSet: newSet(),
	}
	return &Context{
		ref:        cctx,
		internal:   internal,
		lookupCore: newLookupCore(internal, nil),
	}
}

func newLookupCore(t internalContext, parent lookupCore) lookupCore {
	if parent == nil {
		return t
	}

	return &parentLookup{t, parent}
}

func (c *Context) commandContext(cmd *Command) *Context {
	return c.copy(&commandContext{
		cmd:     cmd,
		flagSet: newSet(),
	})
}

func (c *Context) optionContext(opt option) *Context {
	return c.copy(&optionContext{
		option:       opt,
		parentLookup: c.internal.(internalCommandContext),
	})
}

func (c *Context) valueContext(adapter *valueTarget) *Context {
	return c.copy(&valueContext{
		v:      adapter,
		lookup: adapter.lookup(),
	})
}

func (c *Context) setTiming(t Timing) *Context {
	c.timing = t
	return c
}

func triggerOptionsHO(t Timing, on func(*Context) error) ActionFunc {
	return func(ctx *Context) error {
		for _, f := range ctx.Flags() {
			if f.internalFlags().persistent() {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}

			err := on(ctx.optionContext(f).setTiming(t))
			if err != nil {
				return err
			}
		}

		for _, f := range ctx.LocalArgs() {
			err := on(ctx.optionContext(f).setTiming(t))
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func triggerValueTargetsHO(t Timing, on func(*Context) error) ActionFunc {
	return func(ctx *Context) error {
		me, _ := ctx.hookable()
		for _, sub := range me.valueTargets() {
			err := on(ctx.valueContext(sub).setTiming(t))
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func (c *Context) initialize() error {
	if c == nil {
		return nil
	}
	c.setTiming(InitialTiming)
	return tunnel(
		c,
		(internalContext).initialize,
		(internalContext).initializeDescendent,
	)
}

// reinitialize is a special case where hooks need to be
// re-run because a flag or arg changed its name during its
// own initialization process
func (c *Context) reinitialize() error {
	if c == nil {
		return nil
	}
	c.setTiming(InitialTiming)
	return tunnel(
		c,
		nil,
		(internalContext).initializeDescendent,
	)
}

func (c *Context) requireInit() error {
	if !c.IsInitializing() {
		return c.internalError(ErrTimingTooLate)
	}
	return nil
}

func (c *Context) internalError(err error) error {
	return &InternalError{Path: c.Path(), Timing: c.Timing(), Err: err}
}

func (c *Context) copy(t internalContext) *Context {
	return &Context{
		ref:        c.ref,
		Stdin:      c.Stdin,
		Stdout:     c.Stdout,
		Stderr:     c.Stderr,
		FS:         c.FS,
		internal:   t,
		parent:     c,
		request:    c.request,
		lookupCore: newLookupCore(t, c),
	}
}

func bubble(start *Context, self, anc func(internalContext, context.Context) error) error {
	current := start
	fn := self
	for current != nil {
		if err := fn(current.internal, start); err != nil {
			return err
		}
		fn = anc
		current = current.Parent()
	}
	return nil
}

func tunnel(start *Context, self, anc func(internalContext, context.Context) error) error {
	lineage := start.Lineage()
	fn := anc
	for i := len(lineage) - 1; i >= 0; i-- {
		current := lineage[i]
		if i == 0 {
			fn = self
		}
		if fn == nil {
			continue
		}
		if err := fn(current.internal, start); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) executeBefore() error {
	c.setTiming(BeforeTiming)
	return bubble(
		c,
		(internalContext).executeBefore,
		(internalContext).executeBeforeDescendent,
	)
}

func (c *Context) executeAfter() error {
	c.setTiming(AfterTiming)
	return tunnel(
		c,
		(internalContext).executeAfter,
		(internalContext).executeAfterDescendent,
	)
}

func (c *Context) executeSelf() error {
	c.setTiming(ActionTiming)
	return c.internal.execute(c)
}

func (c *Context) lookupOption(name interface{}) (option, bool) {
	if name == "" {
		o, ok := c.target().(option)
		return o, ok
	}
	if f, ok := c.LookupFlag(name); ok {
		return f, true
	}
	return c.LookupArg(name)
}

func (c *Context) option() option {
	return c.target().(option)
}

func (c *Context) actualFS() fs.FS {
	if c.FS == nil {
		return newDefaultFS(c.Stdin, c.Stdout)
	}
	return c.FS
}

// Bool obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Bool(name any) bool {
	return lookupBool(c, name)
}

// String obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) String(name any) string {
	return lookupString(c, name)
}

// List obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) List(name any) []string {
	return lookupList(c, name)
}

// Int obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Int(name any) int {
	return lookupInt(c, name)
}

// Int8 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Int8(name any) int8 {
	return lookupInt8(c, name)
}

// Int16 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Int16(name any) int16 {
	return lookupInt16(c, name)
}

// Int32 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Int32(name any) int32 {
	return lookupInt32(c, name)
}

// Int64 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Int64(name any) int64 {
	return lookupInt64(c, name)
}

// Uint obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Uint(name any) uint {
	return lookupUint(c, name)
}

// Uint8 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Uint8(name any) uint8 {
	return lookupUint8(c, name)
}

// Uint16 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Uint16(name any) uint16 {
	return lookupUint16(c, name)
}

// Uint32 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Uint32(name any) uint32 {
	return lookupUint32(c, name)
}

// Uint64 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Uint64(name any) uint64 {
	return lookupUint64(c, name)
}

// Float32 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Float32(name any) float32 {
	return lookupFloat32(c, name)
}

// Float64 obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Float64(name any) float64 {
	return lookupFloat64(c, name)
}

// Duration obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Duration(name any) time.Duration {
	return lookupDuration(c, name)
}

// File obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) File(name any) *File {
	return lookupFile(c, name)
}

// FileSet obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) FileSet(name any) *FileSet {
	return lookupFileSet(c, name)
}

// Map obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Map(name any) map[string]string {
	return lookupMap(c, name)
}

// NameValue obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) NameValue(name any) *NameValue {
	return lookupNameValue(c, name)
}

// NameValues obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) NameValues(name any) []*NameValue {
	return lookupNameValues(c, name)
}

// URL obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) URL(name any) *url.URL {
	return lookupURL(c, name)
}

// Regexp obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Regexp(name any) *regexp.Regexp {
	return lookupRegexp(c, name)
}

// IP obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) IP(name any) net.IP {
	return lookupIP(c, name)
}

// BigInt obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) BigInt(name any) *big.Int {
	return lookupBigInt(c, name)
}

// BigFloat obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) BigFloat(name any) *big.Float {
	return lookupBigFloat(c, name)
}

// Bytes obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Bytes(name any) []byte {
	return lookupBytes(c, name)
}

// Interface obtains a value by the name of the flag, arg, or other value in scope
func (c *Context) Interface(name any) (any, bool) {
	return tryLookup(c, name, false)
}

func (v *valueTarget) setDescription(arg interface{}) {
	switch val := v.v.(type) {
	case interface{ SetDescription(string) }:
		val.SetDescription(fmt.Sprint(arg))
	case interface{ SetDescription(interface{}) }:
		val.SetDescription(arg)
	}
}

func (v *valueTarget) setHelpText(arg string) {
	if val, ok := v.v.(interface{ SetHelpText(string) }); ok {
		val.SetHelpText(arg)
	}
}

func (v *valueTarget) setManualText(arg string) {
	if val, ok := v.v.(interface{ SetManualText(string) }); ok {
		val.SetManualText(arg)
	}
}

func (v *valueTarget) setUsageText(arg string) {
	if val, ok := v.v.(interface{ SetUsageText(string) }); ok {
		val.SetUsageText(arg)
	}
}

func (v *valueTarget) setCategory(arg string) {
	if val, ok := v.v.(interface{ SetCategory(string) }); ok {
		val.SetCategory(arg)
	}
}

func (v *valueTarget) setCompletion(c Completion) {
	if val, ok := v.v.(interface{ SetCompletion(Completion) }); ok {
		val.SetCompletion(c)
	}
}

func (v *valueTarget) description() any {
	return nil
}

func (v *valueTarget) helpText() string {
	return ""
}

func (v *valueTarget) usageText() string {
	return ""
}

func (v *valueTarget) manualText() string {
	return ""
}

func (v *valueTarget) category() string {
	return ""
}

func (v *valueTarget) data() map[string]any {
	return nil
}

func (*valueTarget) SetHidden(bool) {
}

func (*valueTarget) setInternalFlags(internalFlags, bool) {}

func (*valueTarget) internalFlags() internalFlags {
	return 0
}

func (v *valueTarget) SetData(name string, val interface{}) {
	if t, ok := v.v.(interface{ SetData(string, interface{}) }); ok {
		t.SetData(name, val)
	}
}

func (v *valueTarget) LookupData(name string) (interface{}, bool) {
	if t, ok := v.v.(interface {
		LookupData(string) (interface{}, bool)
	}); ok {
		return t.LookupData(name)
	}
	return nil, false
}

func (v *valueTarget) LocalArgs() []*Arg {
	if a, ok := v.v.(interface{ LocalArgs() []*Arg }); ok {
		return a.LocalArgs()
	}
	return nil
}

func (*valueTarget) completion() Completion {
	return nil
}

func (*valueTarget) options() *Option {
	return nil
}

func (*valueTarget) pipeline(Timing) interface{} {
	return nil
}

func (v *valueTarget) lookup() BindingLookup {
	b, _ := v.v.(BindingLookup)
	return b
}

func applyImplicitVisibility(c *Context) error {
	if strings.HasPrefix(strings.TrimLeft(c.Name(), "-"), "_") {
		if c.target().internalFlags().visibleExplicitlyRequested() {
			return nil
		}
		if c.flagSetOrAncestor((internalFlags).disableAutoVisibility) {
			return nil
		}

		c.target().setInternalFlags(internalFlagHidden, true)
	}
	return nil
}

func preventSetupIfPresent(c context.Context) error {
	// PreventSetup if specified must be handled before all other options
	opts := FromContext(c).target().options()
	return execute(c, *opts&PreventSetup)
}

func applyUserOptions(c context.Context) error {
	opts := FromContext(c).target().options()
	return execute(c, opts)
}

func executeDeferredPipeline(at Timing) actionFunc {
	return func(c context.Context) error {
		return execute(c, FromContext(c).target().uses().pipeline(at))
	}
}

func executeUserPipeline(at Timing) actionFunc {
	return func(c context.Context) error {
		return execute(c, ActionOf(FromContext(c).target().pipeline(at)))
	}
}

func triggerOptions(ctx *Context) error {
	// Invoke the Before action on all flags and args, but only the actual
	// Action when the flag or arg was set
	for _, f := range ctx.Flags() {
		err := triggerOption(ctx, f)
		if err != nil {
			return err
		}
	}

	for _, f := range ctx.LocalArgs() {
		err := triggerOption(ctx, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func triggerOption(ctx *Context, f option) error {
	if f.Seen() || hasSeenImplied(f, ctx.target()) {
		return ctx.optionContext(f).executeSelf()
	}
	return nil
}

func hasSeenImplied(f option, parent target) bool {
	if f.internalFlags().triggerRequested() {
		return true
	}
	if f.internalFlags().seenImplied() {
		return f.internalFlags().impliedAction() || parent.internalFlags().impliedAction()
	}
	return false
}

func reverse(arr []string) []string {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func setupValueInitializer(c context.Context) error {
	if v, ok := FromContext(c).option().value().(valueInitializer); ok {
		return Do(c, v.Initializer())
	}
	return nil
}

func setupOptionFromEnv(c context.Context) error {
	opt := FromContext(c).option()
	return Do(c, Pipeline(
		FromEnv(opt.envVars()...),
		FromFilePath(nil, opt.filePath()),
	))
}

func checkForRequiredOption(c *Context) error {
	if c.option().internalFlags().required() {
		if !c.Seen("") {
			return expectedRequiredOption(c.Name())
		}
	}
	return nil
}

func checkForSupportedFlagType(c *Context) error {
	if err := checkSupportedFlagType(c.option().value()); err != nil {
		return c.internalError(err)
	}
	return nil
}

func executeOptionPipeline(ctx context.Context) error {
	target := FromContext(ctx).target()
	return Do(ctx, Pipeline(target.uses().pipeline(ActionTiming), target.pipeline(ActionTiming)))
}

var (
	_ Lookup          = (*Context)(nil)
	_ context.Context = (*Context)(nil)
	_ internalContext = (*commandContext)(nil)
	_ internalContext = (*optionContext)(nil)
	_ internalContext = (*valueContext)(nil)
)
