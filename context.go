package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/term"
)

// Context provides the context in which the app, command, or flag is executing
type Context struct {
	context.Context
	*contextData
	internal internalContext
	timing   timing

	parent *Context
}

type internalContext interface {
	initialize(*Context) error
	executeBeforeDescendent(*Context) error
	executeBefore(*Context) error
	executeAfter(*Context) error
	executeAfterDescendent(*Context) error
	execute(*Context) error
	app() (*App, bool)
	args() []string
	set() *set
	target() target // *Command, *Arg, *Flag, or *Expr
	lookupValue(string) (interface{}, bool)
	setDidSubcommandExecute()
	hooks() *hooks
	Name() string
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

type hooks struct {
	before []*hook
	after  []*hook
}

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

func newContextPathPattern(pat string) contextPathPattern {
	return contextPathPattern{strings.Fields(pat)}
}

func (c *Context) Parent() *Context {
	if c == nil {
		return nil
	}
	return c.parent
}

func (c *Context) App() *App {
	if cmd, ok := c.internal.app(); ok {
		return cmd
	}
	return c.Parent().App()
}

func (c *Context) Command() *Command {
	if cmd, ok := c.target().(*Command); ok {
		return cmd
	}
	return c.Parent().Command()
}

func (c *Context) Arg() *Arg {
	if a, ok := c.target().(*Arg); ok {
		return a
	}
	return c.Parent().Arg()
}

func (c *Context) Expr() *Expr {
	return c.target().(*Expr)
}

func (c *Context) Flag() *Flag {
	if f, ok := c.target().(*Flag); ok {
		return f
	}
	return c.Parent().Flag()
}

func (c *Context) IsApp() bool {
	_, ok := c.internal.app()
	return ok
}

func (c *Context) IsCommand() bool {
	_, ok := c.target().(*Command)
	return ok
}

func (c *Context) IsExpr() bool {
	_, ok := c.target().(*Expr)
	return ok
}

func (c *Context) IsArg() bool {
	_, ok := c.target().(*Arg)
	return ok
}

func (c *Context) IsFlag() bool {
	_, ok := c.target().(*Flag)
	return ok
}

func (c *Context) IsInitializing() bool { return c.timing == initialTiming }
func (c *Context) IsBefore() bool       { return c.timing == beforeTiming }
func (c *Context) IsAfter() bool        { return c.timing == afterTiming }

func (c *Context) Timing() int { return int(c.timing) }

func (c *Context) isOption() bool {
	_, ok := c.target().(option)
	return ok
}

func (c *Context) Args() []string {
	return c.internal.args()
}

func (c *Context) LookupFlag(name interface{}) *Flag {
	if c == nil {
		return nil
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
				return f
			}
		}
	case *Flag:
		return v
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}

	return c.Parent().LookupFlag(name)
}

func (c *Context) LookupArg(name interface{}) *Arg {
	if c == nil {
		return nil
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
				return a
			}
		}
	case *Arg:
		return v
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}
	return c.Parent().LookupArg(name)
}

func (c *Context) Seen(name string) bool {
	f := c.lookupOption(name)
	return f != nil && f.Seen()
}

func (c *Context) Occurrences(name string) int {
	f := c.lookupOption(name)
	if f != nil {
		return f.Occurrences()
	}
	return -1
}

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
	res := make([]interface{}, 0, c.NValue())
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
	case rune:
		return c.valueCore(string(v))
	case int:
		return c.valueCore(c.logicalArg(v).Name)
	case string:
		return c.valueCore(v)
	case nil:
		return c.valueCore("")
	case *Arg:
		return c.valueCore(v.Name)
	case *Flag:
		return c.valueCore(v.Name)
	default:
		return c.Context.Value(name)
	}
}

// Action either stores or executes the action. When called from the initialization or before pipelines, this
// appends the action to the pipeline for the current flag, arg, or command/app.
// When called from the action or after pipelines, this simply causes the action to be invoked immediately.
func (c *Context) Action(v interface{}) error {
	return c.act(v, actionTiming)
}

// Before either stores or executes the action.  When called from the initialization pipeline, this appends
// the action to the Before pipeline for the current flag, arg, expression, or command/app.  If called
// from the Before pipeline, this causes the action to be invoked immeidately.  If called
// at any other time, this causes the action to be ignored and an error to be returned.
func (c *Context) Before(v interface{}) error {
	return c.act(v, beforeTiming)
}

// After either stores or executes the action.  When called from the initialization, before, or action pipelines,
// this appends the action to the After pipeline for the current flag, arg, expression, or command/app.  If called
// from the After pipeline itself, the action is invoked immediately
func (c *Context) After(v interface{}) error {
	return c.act(v, afterTiming)
}

func (c *Context) act(v interface{}, desired timing) error {
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

func (c *Context) target() target {
	if c == nil {
		return nil
	}
	return c.internal.target()
}

func (c *Context) app() *App {
	a, _ := c.internal.app()
	return a
}

func (c *Context) valueCore(name string) interface{} {
	if name == "" {
		if c.isOption() {
			return dereference(c.option().value())
		}
		name = c.Name()
	}

	// Strip possible decorators --flag, <arg>
	name = strings.Trim(name, "-<>")
	if v, ok := c.internal.lookupValue(name); ok {
		return dereference(v)
	}
	return c.Parent().Value(name)
}

func (c *Context) Bool(name interface{}) bool {
	return lookupBool(c, name)
}

func (c *Context) String(name interface{}) string {
	return lookupString(c, name)
}

func (c *Context) List(name interface{}) []string {
	return lookupList(c, name)
}

func (c *Context) Int(name interface{}) int {
	return lookupInt(c, name)
}

func (c *Context) Int8(name interface{}) int8 {
	return lookupInt8(c, name)
}

func (c *Context) Int16(name interface{}) int16 {
	return lookupInt16(c, name)
}

func (c *Context) Int32(name interface{}) int32 {
	return lookupInt32(c, name)
}

func (c *Context) Int64(name interface{}) int64 {
	return lookupInt64(c, name)
}

func (c *Context) UInt(name interface{}) uint {
	return lookupUInt(c, name)
}

func (c *Context) UInt8(name interface{}) uint8 {
	return lookupUInt8(c, name)
}

func (c *Context) UInt16(name interface{}) uint16 {
	return lookupUInt16(c, name)
}

func (c *Context) UInt32(name interface{}) uint32 {
	return lookupUInt32(c, name)
}

func (c *Context) UInt64(name interface{}) uint64 {
	return lookupUInt64(c, name)
}

func (c *Context) Float32(name interface{}) float32 {
	return lookupFloat32(c, name)
}

func (c *Context) Float64(name interface{}) float64 {
	return lookupFloat64(c, name)
}

// File obtains the file for the specified flag or argument.
func (c *Context) File(name interface{}) *File {
	return lookupFile(c, name)
}

func (c *Context) Map(name interface{}) map[string]string {
	return lookupMap(c, name)
}

func (c *Context) URL(name interface{}) *url.URL {
	return lookupURL(c, name)
}

func (c *Context) Regexp(name interface{}) *regexp.Regexp {
	return lookupRegexp(c, name)
}

func (c *Context) IP(name interface{}) *net.IP {
	return lookupIP(c, name)
}

func (c *Context) Do(actions ...Action) error {
	for _, a := range actions {
		err := a.Execute(c)
		if err != nil {
			return err
		}
	}
	return nil
}

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

func (c *Context) Name() string {
	switch t := c.target().(type) {
	case *Arg:
		return fmt.Sprintf("<%s>", t.Name)
	case *Flag:
		if len(t.Name) == 1 {
			return fmt.Sprintf("-%s", t.Name)
		}
		return fmt.Sprintf("--%s", t.Name)
	case *Command:
		return t.Name
	}
	return ""
}

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

func (c *Context) demandInit() *hooks {
	// TODO Might not always be available
	return c.internal.hooks()
}

func (c ContextPath) Last() string {
	return c[len(c)-1]
}

func (c ContextPath) IsCommand() bool {
	return !(c.IsFlag() || c.IsArg())
}

func (c ContextPath) IsFlag() bool {
	return []rune(c.Last())[0] == '-'
}

func (c ContextPath) IsArg() bool {
	return []rune(c.Last())[0] == '<'
}

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
//
func (c ContextPath) Match(pattern string) bool {
	return newContextPathPattern(pattern).Match(c)
}

func (cp contextPathPattern) Match(c ContextPath) bool {
	if len(cp.parts) == 0 {
		return true
	}
	if len(cp.parts) == 1 && cp.parts[0] == "*" {
		return true
	}
	for i, j := len(cp.parts)-1, len(c)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if matchField(cp.parts[i], c[j]) {
			return true
		}
	}
	return false
}

func (i *hooks) hookBefore(pat string, a Action) {
	i.before = append(i.before, &hook{newContextPathPattern(pat), a})
}

func (i *hooks) execBeforeHooks(target *Context) error {
	if i == nil {
		return nil
	}
	for _, b := range i.before {
		if b.pat.Match(target.Path()) {
			b.action.Execute(target)
		}
	}
	return nil
}

func (i *hooks) hookAfter(pat string, a Action) {
	i.after = append(i.after, &hook{newContextPathPattern(pat), a})
}

func (i *hooks) execAfterHooks(target *Context) error {
	if i == nil {
		return nil
	}
	for _, b := range i.after {
		if b.pat.Match(target.Path()) {
			b.action.Execute(target)
		}
	}
	return nil
}

func matchField(pattern, field string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == "-" || pattern == "--" {
		return strings.HasPrefix(field, "-")
	}
	if pattern == "<>" {
		return strings.HasPrefix(field, "<")
	}
	if pattern == field {
		return true
	}
	return false
}

func rootContext(cctx context.Context, app *App, args []string) *Context {
	return &Context{
		Context:     cctx,
		contextData: &contextData{},
		internal: &appContext{
			commandContext: &commandContext{
				cmd:   nil, // This will be set after initialization
				args_: args,
			},
			app_: app,
		},
	}
}

func (c *Context) commandContext(cmd *Command, args []string) *Context {
	return &Context{
		Context:     c.Context,
		contextData: c.contextData,
		internal: &commandContext{
			cmd:   cmd,
			args_: args,
		},
		parent: c,
	}
}

func (c *Context) flagContext(opt *Flag, args []string) *Context {
	return &Context{
		Context:     c.Context,
		contextData: c.contextData,
		internal: &flagContext{
			option: opt,
			args_:  args,
		},
		parent: c,
	}
}

func (c *Context) argContext(opt *Arg, args []string) *Context {
	return &Context{
		Context:     c.Context,
		contextData: c.contextData,
		internal: &argContext{
			option: opt,
			args_:  args,
		},
		parent: c,
	}
}

func (c *Context) optionContext(opt option, args []string) *Context {
	if f, ok := opt.(*Flag); ok {
		return c.flagContext(f, args)
	}
	return c.argContext(opt.(*Arg), args)
}

func (c *Context) exprContext(expr *Expr, args []string, data *set) *Context {
	return &Context{
		Context:     c.Context,
		contextData: c.contextData,
		internal: &exprContext{
			expr:  expr,
			args_: args,
			set_:  data,
		},
		parent: c,
	}
}

func (c *Context) setTiming(t timing) *Context {
	c.timing = t
	return c
}

func (c *Context) applySet() {
	set := c.internal.set()
	for _, f := range c.target().(*Command).actualFlags() {
		f.applyToSet(set)
	}
	if c.Parent() != nil {
		for _, f := range c.Parent().allFlagsInScope() {
			f.applyToSet(set)
			f.option.persistent = true
		}
	}
	for _, a := range c.target().(*Command).actualArgs() {
		a.applyToSet(set)
	}
}

func (c *Context) allFlagsInScope() []*Flag {
	result := make([]*Flag, 0)
	for {
		var (
			cmd *Command
			ok  bool
			all = map[string]bool{}
		)
		if cmd, ok = c.target().(*Command); !ok {
			break
		}
		for _, f := range cmd.actualFlags() {
			if all[f.Name] {
				continue
			}
			all[f.Name] = true
			result = append(result, f)
		}
		c = c.Parent()
	}
	return result
}

func (c *Context) flagsAndArgs(persistent bool) []option {
	cmd := c.Command()
	res := make([]option, 0, len(cmd.Flags)+len(cmd.Args))
	if persistent {
		for _, f := range c.allFlagsInScope() {
			res = append(res, f)
		}
	} else {
		for _, f := range cmd.Flags {
			res = append(res, f)
		}
	}
	for _, a := range cmd.Args {
		res = append(res, a)
	}
	return res
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
	c.timing = initialTiming
	return c.internal.initialize(c)
}

func (c *Context) executeBeforeHooks(which *Command) error {
	return c.target().hooks().execBeforeHooks(c)
}

func (c *Context) executeAfterHooks(which *Command) error {
	return c.target().hooks().execAfterHooks(c)
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
	c.setTiming(beforeTiming)
	return bubble(
		c,
		func(c1 *Context) error { return c1.internal.executeBefore(c) },
		func(c1 *Context) error { return c1.internal.executeBeforeDescendent(c) },
	)
}

func (c *Context) executeAfter() error {
	c.setTiming(afterTiming)
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

	c.setTiming(actionTiming)
	if err := c.internal.execute(c); err != nil {
		return err
	}

	return c.executeAfter()
}

func (c *Context) executeOption() error {
	return executeAll(c, c.option().action())
}

func (c *Context) lookupOption(name string) option {
	f := c.LookupFlag(name)
	if f != nil {
		return f
	}
	return c.LookupArg(name)
}

func (c *Context) option() option {
	return c.target().(option)
}

func triggerFlagsAndArgs(ctx *Context) error {
	opts := ctx.flagsAndArgs(true)
	bindings := ctx.internal.set().bindings
	for _, f := range opts {
		if flag, ok := f.(*Flag); ok {
			if flag.option.persistent {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}
		}

		err := ctx.optionContext(f, bindings[f.name()]).setTiming(beforeTiming).executeBefore()
		if err != nil {
			return err
		}
	}

	// Invoke the Before action on all flags and args, but only the actual
	// Action when the flag or arg was set
	for _, f := range opts {
		if f.Seen() {
			err := ctx.optionContext(f, bindings[f.name()]).setTiming(actionTiming).executeOption()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func triggerAfterFlagsAndArgs(ctx *Context) error {
	opts := ctx.flagsAndArgs(true)
	bindings := ctx.internal.set().bindings
	for _, f := range opts {
		if flag, ok := f.(*Flag); ok {
			if flag.option.persistent {
				// This is a persistent flag that was cloned into the flag set of the current
				// command; don't process it again
				continue
			}
		}

		err := ctx.optionContext(f, bindings[f.name()]).setTiming(afterTiming).executeAfter()
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
		if err == nil && width > 12 {
			return width
		}
	}
	return 80
}

var (
	_ hasArguments    = &Expr{}
	_ Lookup          = &Context{}
	_ internalContext = &commandContext{}
	_ internalContext = &flagContext{}
	_ internalContext = &argContext{}
	_ internalContext = &exprContext{}
	_ internalContext = &appContext{}
)
