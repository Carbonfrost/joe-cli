package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"golang.org/x/term"
)

// Context provides the context in which the app, command, or flag is executing or initializing
type Context struct {
	context.Context

	*contextData
	*lookupSupport
	internal internalContext
	timing   Timing
	argList  []string

	parent    *Context
	pathCache ContextPath
}

// WalkFunc provides the callback for the Walk function
type WalkFunc func(cmd *Context) error

type internalContext interface {
	lookupCore
	initialize(*Context) error
	executeBeforeDescendent(*Context) error
	executeBefore(*Context) error
	executeAfter(*Context) error
	executeAfterDescendent(*Context) error
	execute(*Context) error
	lookupBinding(string) []string
	target() target // *Command, *Arg, *Flag, or *Expr
	setDidSubcommandExecute()
	Name() string
}

type parentLookup struct {
	lookupCore // delegates to the internal context
	parent     lookupCore
}

type hasArguments interface {
	actualArgs() []*Arg
}

type hasFlags interface {
	actualFlags() []*Flag
}

type hook struct {
	pat    contextPathPattern
	action Action
}

type valueTarget struct {
	pipelinesSupport

	v interface{}
}

// ContextPath provides a list of strings that name each one of the parent components
// in the context.  Each string follows the form:
//
//   command  a command matching the name "command"
//   -flag    a flag matching the flag name
//   <arg>    an argument matching the arg name
//
type ContextPath []string

type contextPathPattern struct {
	parts []string
}

// contextData provides data that is copied into child contexts
type contextData struct {
	Stdout Writer
	Stderr Writer
	Stdin  io.Reader
}

var (
	// SkipCommand is used as a return value from WalkFunc to indicate that the command in the call is to be skipped.
	SkipCommand = errors.New("skip this command")

	errModifyAfterInit = errors.New("modification has no effect at this time")
)

func newContextPathPattern(pat string) contextPathPattern {
	return contextPathPattern{strings.Fields(pat)}
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
	if root, ok := c.internal.(*appContext); ok {
		return root.app, true
	}
	return nil, false
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

// Timing retrieves the timing
func (c *Context) Timing() Timing {
	return c.timing
}

func (c *Context) isOption() bool {
	_, ok := c.target().(option)
	return ok
}

// Args retrieves the arguments.  IF the context corresponds to a command, these
// represent the name of the command plus the arguments passed to it.  For flags and arguments,
// this is the value passed to them
func (c *Context) Args() []string {
	return c.argList
}

// LookupCommand finds the command by name.  The name can be a string or *Command
func (c *Context) LookupCommand(name interface{}) (*Command, bool) {
	if c == nil {
		return nil, false
	}
	switch v := name.(type) {
	case string:
		if v == "" {
			return nil, false
		}
		if aa, ok := c.target().(*Command); ok {
			if r, found := findCommandByName(aa.Subcommands, v); found {
				return r, true
			}
		}
	case *Command:
		return v, true
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
		if aa, ok := c.target().(hasFlags); ok {
			if f, found := findFlagByName(aa.actualFlags(), v); found {
				return f, true
			}
		}
	case *Flag:
		return v, true
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
		if aa, ok := c.target().(hasArguments); ok {
			if a, found := findArgByName(aa.actualArgs(), v); found {
				return a, true
			}
		}
	case *Arg:
		return v, true
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
			return c.flagContext(f, nil), true
		}
	case matchArg(name):
		if a, ok := c.LookupArg(name); ok {
			return c.argContext(a, nil), true
		}
	default:
		if m, ok := c.LookupCommand(name); ok {
			return c.commandContext(m), true
		}
	}
	return nil, false
}

// Seen returns true if the specified flag or argument has been used at least once
func (c *Context) Seen(name string) bool {
	f, ok := c.lookupOption(name)
	return ok && f.Seen()
}

// Occurrences returns the number of times the specified flag or argument has been used
func (c *Context) Occurrences(name string) int {
	if f, ok := c.lookupOption(name); ok {
		return f.Occurrences()
	}
	return -1
}

// Expression obtains the expression from the context
func (c *Context) Expression(name string) *Expression {
	return c.Value(name).(*Expression)
}

// NValue gets the maximum number available, exclusive, as an argument Value.
func (c *Context) NValue() int {
	if t, ok := c.target().(hasArguments); ok {
		return len(t.actualArgs())
	}
	return 0
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
//  * rune - corresponds to the short name of a flag
//  * int - obtain the argument by index
//  * *Arg - get value of the arg
//  * *Flag - get value of the flag
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
		return c.lookupSupport.Value(c.nameToString(v))
	default:
		return c.Context.Value(name)
	}
}

// LookupData gets the data matching the key, including recursive traversal
// up the lineage contexts
func (c *Context) LookupData(name string) (interface{}, bool) {
	if c == nil {
		return nil, false
	}
	if res, ok := c.target().LookupData(name); ok {
		return res, true
	}
	return c.Parent().LookupData(name)
}

// SetData sets data on the current target
func (c *Context) SetData(name string, v interface{}) {
	c.target().SetData(name, v)
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

// SetValue sets the value of the current flag or arg
func (c *Context) SetValue(arg string) error {
	return c.target().(option).Set(arg)
}

// Action either stores or executes the action. When called from the initialization or before pipelines, this
// appends the action to the pipeline for the current flag, arg, or command/app.
// When called from the action or after pipelines, this simply causes the action to be invoked immediately.
func (c *Context) Action(v interface{}) error {
	return c.act(v, ActionTiming)
}

// Before either stores or executes the action.  When called from the initialization pipeline, this appends
// the action to the Before pipeline for the current flag, arg, expression, or command/app.  If called
// from the Before pipeline, this causes the action to be invoked immeidately.  If called
// at any other time, this causes the action to be ignored and an error to be returned.
func (c *Context) Before(v interface{}) error {
	return c.act(v, BeforeTiming)
}

// After either stores or executes the action.  When called from the initialization, before, or action pipelines,
// this appends the action to the After pipeline for the current flag, arg, expression, or command/app.  If called
// from the After pipeline itself, the action is invoked immediately
func (c *Context) After(v interface{}) error {
	return c.act(v, AfterTiming)
}

func (c *Context) act(v interface{}, desired Timing) error {
	if c.timing < desired {
		c.target().appendAction(desired, ActionOf(v))
		return nil
	}
	if c.timing == desired {
		return ActionOf(v).Execute(c)
	}
	if c.timing > desired {
		return errors.New("too late to exec action")
	}
	return nil
}

// Target retrieves the target of the context, which is *App, *Command, *Flag, *Arg,
// or *Expr
func (c *Context) Target() interface{} {
	return c.target()
}

func (c *Context) target() target {
	// This is to avoid complexity in test setup; internal should never actually be nil
	if c == nil || c.internal == nil {
		return nil
	}
	return c.internal.target()
}

func (c *Context) hookable() (hookable, bool) {
	h, ok := c.internal.target().(hookable)
	return h, ok
}

func (c *Context) customizable() (customizable, bool) {
	// if c == nil || c.internal == nil{
	// 	return nil, false
	// }
	h, ok := c.target().(customizable)
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
func (c *Context) Do(actions ...Action) error {
	currentTiming := c.Timing()
	for _, a := range actions {
		if a == nil {
			continue
		}
		err := c.act(a, timingOf(a, currentTiming))
		if err != nil {
			return err
		}
	}
	return nil
}

// Template retrieves a template by name
func (c *Context) Template(name string) *Template {
	str := c.App().ensureTemplates()[name]
	funcMap := c.App().ensureTemplateFuncs()
	return &Template{
		Template: template.Must(
			template.New(name).Funcs(funcMap).Parse(str),
		),
		Debug: os.Getenv("CLI_DEBUG_TEMPLATES") == "1",
	}
}

// Name gets the name of the context, which is the name of the command, arg, flag, or expression
// operator in use
func (c *Context) Name() string {
	// This is to avoid complexity in test setup; internal should never actually be nil
	if c.internal == nil {
		return ""
	}
	return c.internal.Name()
}

// Path retrieves all of the names on the context and its ancetors to the root
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
	return c.target().(hasArguments).actualArgs()[index]
}

// AddFlag provides a convenience method that adds a flag to the current command or app.  This
// is only valid during the initialization phase.  An error is returned for other timings.
func (c *Context) AddFlag(f *Flag) error {
	if !c.IsInitializing() {
		return errModifyAfterInit
	}
	if app, ok := c.app(); ok {
		app.Flags = append(app.Flags, f)
	} else {
		c.Command().Flags = append(c.Command().Flags, f)
	}
	return nil
}

// AddCommand provides a convenience method that adds a Command to the current command or app.  This
// is only valid during the initialization phase.  An error is returned for other timings.
func (c *Context) AddCommand(v *Command) error {
	if !c.IsInitializing() {
		return errModifyAfterInit
	}
	if app, ok := c.app(); ok {
		app.Commands = append(app.Commands, v)
	} else {
		c.Command().Subcommands = append(c.Command().Subcommands, v)
	}
	return nil
}

// AddArg provides a convenience method that adds an Arg to the current command or app.  This
// is only valid during the initialization phase.  An error is returned for other timings.
func (c *Context) AddArg(v *Arg) error {
	if !c.IsInitializing() {
		return errModifyAfterInit
	}
	if app, ok := c.app(); ok {
		app.Args = append(app.Args, v)
	} else {
		c.Command().Args = append(c.Command().Args, v)
	}
	return nil
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

// SkipImplicitSetup gets whether implicit setup steps should be skipped
func (c *Context) SkipImplicitSetup() bool {
	_, ok := c.LookupData("_taintSetup")
	return ok
}

// PreventSetup causes implicit setup options to be skipped.  The function
// returns an error if the timing is not initial timing.
func (c *Context) PreventSetup() error {
	if !c.IsInitializing() {
		return errModifyAfterInit
	}
	c.SetData("_taintSetup", true)
	return nil
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

// ProvideValueInitializer causes an additional child context to be created
// which is used to initialize an arbitrary value.  Typically, the value is
// the value of the flag or arg.
// The value can provides methods such as SetDescription(string),
// SetHelpText(string), etc. in order to operate with actions that set these values.
func (c *Context) ProvideValueInitializer(v interface{}, name string, action Action) error {
	adapter := &valueTarget{
		v: v,
		pipelinesSupport: pipelinesSupport{
			&actionPipelines{
				Initializers: action,
			},
		},
	}
	return c.Do(Setup{
		Uses: func(c1 *Context) error {
			return c1.valueContext(adapter, name).initialize()
		},
		Before: func(c1 *Context) error {
			return c1.valueContext(adapter, name).executeBefore()
		},
		After: func(c1 *Context) error {
			return c1.valueContext(adapter, name).executeAfter()
		},
		Action: func(c1 *Context) error {
			return c1.valueContext(adapter, name).executeSelf()
		},
	})
}

// Customize matches a flag, arg, or command and runs additional pipeline steps.  Customize
// is usually used to apply further customizations after an extension has done setup of
// the defaults.
func (c *Context) Customize(pattern string, a ...Action) error {
	if h, ok := c.customizable(); ok {
		h.customize(pattern, ActionPipeline(a))
		return nil
	}
	return cantHookError
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
//   *        any command
//   command  a command matching the name "command"
//   -        any flag
//   -flag    a flag matching the flag name
//   <>       any argument
//   <arg>    an argument matching the arg name
//   <->      any expression
//   <-expr>  an expression matching the arg name
//
func (c ContextPath) Match(pattern string) bool {
	return newContextPathPattern(pattern).Match(c)
}

func (cp contextPathPattern) Match(c ContextPath) bool {
	if len(cp.parts) == 0 {
		return true
	}
	for i, j := len(cp.parts)-1, len(c)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if matchField(cp.parts[i], c[j]) {
			return true
		}
	}
	return false
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
	internal := &appContext{
		commandContext: &commandContext{
			cmd: nil, // This will be set after initialization
		},
		app: app,
	}
	return &Context{
		Context:       cctx,
		contextData:   &contextData{},
		internal:      internal,
		lookupSupport: newLookupSupport(internal, nil),
	}
}

func newLookupSupport(t internalContext, parent lookupCore) *lookupSupport {
	if parent == nil {
		return &lookupSupport{t}
	}

	return &lookupSupport{
		&parentLookup{t, parent},
	}
}

func (c *Context) commandContext(cmd *Command) *Context {
	return c.copy(&commandContext{
		cmd: cmd,
	}, nil, true)
}

func (c *Context) flagContext(opt *Flag, args []string) *Context {
	return c.copy(&flagContext{
		option: opt,
	}, args, true)
}

func (c *Context) argContext(opt *Arg, args []string) *Context {
	return c.copy(&argContext{
		option:  opt,
		argList: args,
	}, args, true)
}

func (c *Context) optionContext(opt option, args []string) *Context {
	if f, ok := opt.(*Flag); ok {
		return c.flagContext(f, args)
	}
	return c.argContext(opt.(*Arg), args)
}

func (c *Context) valueContext(adapter *valueTarget, name string) *Context {
	return c.copy(&valueContext{
		v:    adapter,
		name: name,
	}, nil, true)
}

func (c *Context) exprContext(expr *Expr, args []string, data *set) *Context {
	adapter := &valueTarget{
		v: expr,
		pipelinesSupport: pipelinesSupport{
			&actionPipelines{},
		},
	}
	return c.copy(&valueContext{
		v:      adapter,
		name:   expr.Name,
		lookup: data,
	}, args, true)
}

func (c *Context) setTiming(t Timing) *Context {
	c.timing = t
	return c
}

func (c *Context) flags(persistent bool) []option {
	result := make([]option, 0)
	var (
		cmd hasFlags
		ok  bool
		all = map[string]bool{}
	)
	for {
		if cmd, ok = c.target().(hasFlags); !ok {
			break
		}
		for _, f := range cmd.actualFlags() {
			if all[f.Name] {
				continue
			}
			all[f.Name] = true
			result = append(result, f)
		}
		if !persistent {
			break
		}
		c = c.Parent()
	}
	return result
}

func (c *Context) args() []option {
	result := make([]option, 0)
	for _, a := range c.target().(hasArguments).actualArgs() {
		result = append(result, a)
	}
	return result
}

func (c *Context) initialize() error {
	if c == nil {
		return nil
	}
	c.timing = InitialTiming
	return c.internal.initialize(c)
}

func (c *Context) copy(t internalContext, args []string, reparent bool) *Context {
	p := c.parent
	if reparent {
		p = c
	}
	return &Context{
		Context:       c.Context,
		contextData:   c.contextData,
		internal:      t,
		parent:        p,
		argList:       args,
		lookupSupport: newLookupSupport(t, p),
	}
}

func bubble(start *Context, self func(*Context) error, anc func(*Context) error) error {
	current := start
	fn := self
	for current != nil {
		if err := fn(current); err != nil {
			return err
		}
		fn = anc
		current = current.Parent()
	}
	return nil
}

func tunnel(start *Context, self func(*Context) error, anc func(*Context) error) error {
	lineage := start.Lineage()
	fn := anc
	for i := len(lineage) - 1; i >= 0; i-- {
		current := lineage[i]
		if i == 0 {
			fn = self
		}
		if err := fn(current); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) executeBefore() error {
	c.setTiming(BeforeTiming)
	return bubble(
		c,
		func(c1 *Context) error { return c1.internal.executeBefore(c) },
		func(c1 *Context) error { return c1.internal.executeBeforeDescendent(c) },
	)
}

func (c *Context) executeAfter() error {
	c.setTiming(AfterTiming)
	return tunnel(
		c,
		func(c1 *Context) error { return c1.internal.executeAfter(c) },
		func(c1 *Context) error { return c1.internal.executeAfterDescendent(c) },
	)
}

func (c *Context) executeCommand() error {
	if err := c.executeBefore(); err != nil {
		return err
	}

	if err := c.executeSelf(); err != nil {
		return err
	}

	return c.executeAfter()
}

func (c *Context) executeSelf() error {
	c.setTiming(ActionTiming)
	return c.internal.execute(c)
}

func (c *Context) lookupOption(name string) (option, bool) {
	if name == "" {
		return c.option(), true
	}
	if f, ok := c.LookupFlag(name); ok {
		return f, true
	}
	return c.LookupArg(name)
}

func (c *Context) option() option {
	return c.target().(option)
}

func (v *valueTarget) setDescription(arg string) {
	switch val := v.v.(type) {
	case targetConventions:
		val.setDescription(arg)
	case interface{ SetDescription(string) }:
		val.SetDescription(arg)
	}
}

func (v *valueTarget) setHelpText(arg string) {
	switch val := v.v.(type) {
	case targetConventions:
		val.setHelpText(arg)
	case interface{ SetHelpText(string) }:
		val.SetHelpText(arg)
	}
}

func (v *valueTarget) setManualText(arg string) {
	switch val := v.v.(type) {
	case targetConventions:
		val.setManualText(arg)
	case interface{ SetManualText(string) }:
		val.SetManualText(arg)
	}
}

func (v *valueTarget) setCategory(arg string) {
	switch val := v.v.(type) {
	case targetConventions:
		val.setCategory(arg)
	case interface{ SetCategory(string) }:
		val.SetCategory(arg)
	}
}

func (*valueTarget) setInternalFlags(internalFlags) {}

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

func (*valueTarget) WriteSynopsis(Writer) {}

func (v *valueTarget) actualArgs() []*Arg {
	if a, ok := v.v.(hasArguments); ok {
		return a.actualArgs()
	}
	return nil
}

func triggerBeforeFlags(ctx *Context) error {
	return triggerBeforeOptions(ctx, ctx.flags(true))
}

func triggerBeforeArgs(ctx *Context) error {
	return triggerBeforeOptions(ctx, ctx.args())
}

func triggerBeforeOptions(ctx *Context, opts []option) error {
	bindings := ctx.internal.lookupBinding
	for _, f := range opts {
		if flag, ok := f.(*Flag); ok {
			if flag.option.flags.persistent() {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}
		}

		err := ctx.optionContext(f, bindings(f.name())).setTiming(BeforeTiming).executeBefore()
		if err != nil {
			return err
		}
	}

	// Invoke the Before action on all flags and args, but only the actual
	// Action when the flag or arg was set
	for _, f := range opts {
		if f.Seen() {
			err := ctx.optionContext(f, bindings(f.name())).setTiming(ActionTiming).executeSelf()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func triggerAfterFlags(ctx *Context) error {
	return triggerAfterOptions(ctx, ctx.flags(true))
}

func triggerAfterArgs(ctx *Context) error {
	return triggerAfterOptions(ctx, ctx.args())
}

func triggerAfterOptions(ctx *Context, opts []option) error {
	bindings := ctx.internal.lookupBinding
	for _, f := range opts {
		if flag, ok := f.(*Flag); ok {
			if flag.option.flags.persistent() {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}
		}

		err := ctx.optionContext(f, bindings(f.name())).setTiming(AfterTiming).executeAfter()
		if err != nil {
			return err
		}
	}
	return nil
}

func reverse(arr []string) []string {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}

func setupValueInitializer(c *Context) error {
	if v, ok := c.option().value().(valueInitializer); ok {
		return c.Do(v.Initializer())
	}
	return nil
}

func fixupOptionInternals(c *Context) error {
	// Because Uses pipeline could have changed Flag, copy flag internals again
	switch o := c.target().(type) {
	case *Flag:
		long, short := canonicalNames(o.Name, o.Aliases)
		o.option.short = short
		o.option.long = long
		o.option.uname = o.Name

		p := o.value()
		if o.Value != nil && p != o.option.value.p {
			o.option.value = wrapGeneric(p)
			o.option.flags |= isFlagType(p)
		}
	case *Arg:
		o.option.uname = o.Name
		if o.option.narg != o.NArg {
			if o.option.flags.destinationImplicitlyCreated() {
				o.Value = nil
			}
			o.option.narg = o.NArg
		}

		p := o.value()
		if o.Value != nil && p != o.option.value.p {
			o.option.value = wrapGeneric(p)
		}
		if _, ok := o.Value.(*string); ok {
			o.option.flags |= internalFlagMerge
		}
	}
	return nil
}

func setupOptionFromEnv(ctx *Context) error {
	o := ctx.option()
	return ctx.Do(ImplicitValue(func() (string, bool) {
		return loadFlagValueFromEnvironment(o)
	}))
}

func handleCustomizations(target *Context) error {
	path := target.Path()
	target.lineageFunc(func(c *Context) {
		if h, ok := c.customizable(); ok {
			for _, b := range h.customizations() {
				if b.pat.Match(path) {
					b.action.Execute(target)
				}
			}
		}
	})
	return nil
}

func guessWidth() int {
	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		width, _, err := term.GetSize(fd)
		if err == nil && width > 12 && width < 80 {
			return width
		}
	}
	return 80
}

var (
	_ hasArguments    = (*Expr)(nil)
	_ Lookup          = (*Context)(nil)
	_ internalContext = (*commandContext)(nil)
	_ internalContext = (*flagContext)(nil)
	_ internalContext = (*argContext)(nil)
	_ internalContext = (*valueContext)(nil)
	_ internalContext = (*appContext)(nil)
)
