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

	parent *Context
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
	args() []string
	set() *set
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
	Stdout io.Writer
	Stderr io.Writer
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
	if cmd, ok := c.target().(*Command); ok {
		return cmd
	}
	return c.Parent().Command()
}

// Arg retrieves the argument in scope if any
func (c *Context) Arg() *Arg {
	if a, ok := c.target().(*Arg); ok {
		return a
	}
	return c.Parent().Arg()
}

// Expr retrieves the expression operator in scope if any
func (c *Context) Expr() *Expr {
	return c.target().(*Expr)
}

// Flag retrieves the flag in scope if any
func (c *Context) Flag() *Flag {
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
	return c.internal.args()
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
func (c *Context) Expression() Expression {
	return c.Value("expression").(Expression)
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
	if c == nil {
		return nil
	}
	return c.internal.target()
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
			if err := c.commandContext(sub, nil).walkCore(fn); err != nil {
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
	return c.internal.Name()
}

// Path retrieves all of the names on the context and its ancetors to the root
func (c *Context) Path() ContextPath {
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

// Lineage retrieves all of the ancestor contexts up to the root
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

func rootContext(cctx context.Context, app *App, args []string) *Context {
	internal := &appContext{
		commandContext: &commandContext{
			cmd:     nil, // This will be set after initialization
			argList: args,
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

func (c *Context) commandContext(cmd *Command, args []string) *Context {
	return c.copy(&commandContext{
		cmd:     cmd,
		argList: args,
	}, true)
}

func (c *Context) flagContext(opt *Flag, args []string) *Context {
	return c.copy(&flagContext{
		option:  opt,
		argList: args,
	}, true)
}

func (c *Context) argContext(opt *Arg, args []string) *Context {
	return c.copy(&argContext{
		option:  opt,
		argList: args,
	}, true)
}

func (c *Context) optionContext(opt option, args []string) *Context {
	if f, ok := opt.(*Flag); ok {
		return c.flagContext(f, args)
	}
	return c.argContext(opt.(*Arg), args)
}

func (c *Context) exprContext(expr *Expr, args []string, data *set) *Context {
	return c.copy(&exprContext{
		expr:    expr,
		argList: args,
		flagSet: data,
	}, true)
}

func (c *Context) setTiming(t Timing) *Context {
	c.timing = t
	return c
}

func (c *Context) applySet() {
	set := c.internal.set()
	for _, f := range c.target().(*Command).actualFlags() {
		f.applyToSet(set)
	}
	if c.Parent() != nil {
		for _, f := range c.Parent().flags(true) {
			if f.internalFlags().nonPersistent() {
				continue
			}
			f.applyToSet(set)
			f.(*Flag).option.persistent = true
		}
	}
	for _, a := range c.target().(*Command).actualArgs() {
		a.applyToSet(set)
	}
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

func (c *Context) applyFlagsAndArgs() (err error) {
	args := make([]string, 0)
	args = append(args, c.internal.args()...)

	if c.Command().internalFlags().skipFlagParsing() {
		args = append([]string{args[0], "--"}, args[1:]...)
	}
	return c.internal.set().parse(args, c.Command().internalFlags().disallowFlagsAfterArgs())
}

func (c *Context) initialize() error {
	if c == nil {
		return nil
	}
	c.timing = InitialTiming
	return c.internal.initialize(c)
}

func (c *Context) copy(t internalContext, reparent bool) *Context {
	p := c.parent
	if reparent {
		p = c
	}
	return &Context{
		Context:       c.Context,
		contextData:   c.contextData,
		internal:      t,
		parent:        p,
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

func (c *Context) executeBeforeWithoutBubbling() error {
	c.setTiming(BeforeTiming)
	return c.internal.executeBefore(c)
}

func (c *Context) executeAfter() error {
	c.setTiming(AfterTiming)
	return tunnel(
		c,
		func(c1 *Context) error { return c1.internal.executeAfter(c) },
		func(c1 *Context) error { return c1.internal.executeAfterDescendent(c) },
	)
}

func (c *Context) executeAfterWithoutTunneling() error {
	c.setTiming(AfterTiming)
	return c.internal.executeAfter(c)
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
	if f, ok := c.LookupFlag(name); ok {
		return f, true
	}
	return c.LookupArg(name)
}

func (c *Context) option() option {
	return c.target().(option)
}

func triggerBeforeFlags(ctx *Context) error {
	return triggerBeforeOptions(ctx, ctx.flags(true))
}

func triggerBeforeArgs(ctx *Context) error {
	return triggerBeforeOptions(ctx, ctx.args())
}

func triggerBeforeOptions(ctx *Context, opts []option) error {
	bindings := ctx.internal.set().bindings
	for _, f := range opts {
		if flag, ok := f.(*Flag); ok {
			if flag.option.persistent {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}
		}

		err := ctx.optionContext(f, bindings[f.name()]).setTiming(BeforeTiming).executeBefore()
		if err != nil {
			return err
		}
	}

	// Invoke the Before action on all flags and args, but only the actual
	// Action when the flag or arg was set
	for _, f := range opts {
		if f.Seen() {
			err := ctx.optionContext(f, bindings[f.name()]).setTiming(ActionTiming).executeSelf()
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
	bindings := ctx.internal.set().bindings
	for _, f := range opts {
		if flag, ok := f.(*Flag); ok {
			if flag.option.persistent {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}
		}

		err := ctx.optionContext(f, bindings[f.name()]).setTiming(AfterTiming).executeAfter()
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
			o.option.flag = isFlagType(p)
		}
	}
	return nil
}

func setupOptionFromEnv(ctx *Context) error {
	o := ctx.option()
	if v, ok := loadFlagValueFromEnvironment(o); ok {
		return o.Set(v)
	}
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
	_ internalContext = (*exprContext)(nil)
	_ internalContext = (*appContext)(nil)
)
