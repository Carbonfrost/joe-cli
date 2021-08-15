package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pborman/getopt/v2"
)

// Context provides the context in which the app, command, or flag is executing
type Context struct {
	context.Context
	*contextData

	parent *Context

	target interface{} // *Command, *Flag, or *Arg

	// When the context is being used for a command
	args   []string
	set    *getopt.Set
	values map[string]interface{}
}

type ContextPath []string

// contextData provides data that is copied into child contexts
type contextData struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

func (c *Context) Parent() *Context {
	if c == nil {
		return nil
	}
	return c.parent
}

func (c *Context) App() *App {
	if cmd, ok := c.target.(*App); ok {
		return cmd
	}
	return c.Parent().App()
}

func (c *Context) Command() *Command {
	if cmd, ok := c.target.(*Command); ok {
		return cmd
	}
	return c.Parent().Command()
}

func (c *Context) Arg() *Arg {
	if a, ok := c.target.(*Arg); ok {
		return a
	}
	return c.Parent().Arg()
}

func (c *Context) Flag() *Flag {
	if f, ok := c.target.(*Flag); ok {
		return f
	}
	return c.Parent().Flag()
}

func (c *Context) IsApp() bool {
	_, ok := c.target.(*App)
	return ok
}

func (c *Context) IsCommand() bool {
	_, ok := c.target.(*Command)
	return ok
}

func (c *Context) IsArg() bool {
	_, ok := c.target.(*Arg)
	return ok
}

func (c *Context) IsFlag() bool {
	_, ok := c.target.(*Flag)
	return ok
}

func (c *Context) Args() []string {
	return c.args
}

func (c *Context) Value(name string) interface{} {
	if c == nil {
		return nil
	}
	if v, ok := c.values[name]; ok {
		return dereference(v)
	}
	return c.Parent().Value(name)
}

func (c *Context) Bool(name string) bool {
	return *c.Value(name).(*bool)
}

func (c *Context) String(name string) string {
	return *c.Value(name).(*string)
}

func (c *Context) List(name string) []string {
	return *c.Value(name).(*[]string)
}

func (c *Context) Int(name string) int {
	return *c.Value(name).(*int)
}

func (c *Context) Int8(name string) int8 {
	return *c.Value(name).(*int8)
}

func (c *Context) Int16(name string) int16 {
	return *c.Value(name).(*int16)
}

func (c *Context) Int32(name string) int32 {
	return *c.Value(name).(*int32)
}

func (c *Context) Int64(name string) int64 {
	return *c.Value(name).(*int64)
}

func (c *Context) UInt(name string) uint {
	return *c.Value(name).(*uint)
}

func (c *Context) UInt8(name string) uint8 {
	return *c.Value(name).(*uint8)
}

func (c *Context) UInt16(name string) uint16 {
	return *c.Value(name).(*uint16)
}

func (c *Context) UInt32(name string) uint32 {
	return *c.Value(name).(*uint32)
}

func (c *Context) UInt64(name string) uint64 {
	return *c.Value(name).(*uint64)
}

func (c *Context) Float32(name string) float32 {
	return *c.Value(name).(*float32)
}

func (c *Context) Float64(name string) float64 {
	return *c.Value(name).(*float64)
}

func (c *Context) Do(actions ...ActionHandler) error {
	for _, a := range actions {
		err := a.Execute(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) Name() string {
	switch t := c.target.(type) {
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

func (c ContextPath) Match(pattern string) bool {
	parts := strings.Fields(pattern)
	if len(parts) == 0 {
		return true
	}
	if len(parts) == 1 && parts[0] == "*" {
		return true
	}
	for i, j := len(parts)-1, len(c)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if matchField(parts[i], c[j]) {
			return true
		}
	}
	return false
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

func rootContext(cctx context.Context, app *App) *Context {
	return &Context{
		Context:     cctx,
		contextData: &contextData{},
		target:      app,
	}
}

func (c *Context) commandContext(cmd *Command, args []string) *Context {
	return &Context{
		Context:     c.Context,
		contextData: c.contextData,
		target:      cmd,
		args:        args,
		parent:      c,
		set:         cmd.createAndApplySet(),
		values:      cmd.createValues(),
	}
}

func (c *Context) optionContext(opt option) *Context {
	return &Context{
		Context:     c.Context,
		contextData: c.contextData,
		target:      opt,
		parent:      c,
	}
}

func (c *Context) applySubcommands() (*Context, error) {
	ctx := c
	args := c.args
	for len(args) > 0 {
		err := ctx.set.Getopt(args, nil)
		if err != nil {
			// Failed to set the option to the corresponding flag
			return nil, err
		}
		args = ctx.set.Args()

		// Args were modified by Getopt to apply any flags and stopped
		// at the first argument.  If the argument matches a sub-command, then
		// we push the command onto the stack
		if len(args) > 0 {
			cmd := ctx.target.(*Command)
			if sub, ok := cmd.Command(args[0]); ok {
				ctx = ctx.commandContext(sub, args)
			} else if len(cmd.Subcommands) > 0 {
				return c, commandMissing(args[0])
			} else {
				// Stop looking for commands; this is it
				break
			}
		}
	}
	return ctx, nil
}

func (ctx *Context) applyFlagsAndArgs() (err error) {
	var (
		currentIndex = -1
		current      *Arg

		// ctx.args contains the name of the command and its arguments
		args []string = ctx.args

		enumerator = func() bool {
			cmd := ctx.target.(*Command)
			actual := cmd.actualArgs()
			currentIndex = currentIndex + 1
			if currentIndex < len(actual) {
				current = actual[currentIndex]
				return true
			}
			if len(args) > 0 {
				err = unexpectedArgument(args[0])
			}
			return false
		}
	)

	for enumerator() {
		if err != nil {
			return
		}

		err = ctx.set.Getopt(args, nil)
		if err != nil {
			return
		}

		args = ctx.set.Args()
		if len(args) == 0 {
			break
		}

		args, err = applyArgument(args, current)
	}

	// Done with parsing.  Returning here is necessary because trying empty args
	// with Getopt will corrupt its internal state
	if len(args) == 0 {
		return
	}

	// Any remaining parsing must be flags only
	err = ctx.set.Getopt(args, nil)
	if err != nil {
		return
	}
	args = ctx.set.Args()

	if len(args) > 0 {
		err = unexpectedArgument(args[0])
	}
	return
}

func applyArgument(args []string, current *Arg) ([]string, error) {
	var (
		// takeArgs updates args by taking the values that will be passed to
		// the *Arg.Set call.  narg < 0 implies taking all arguments,
		// 0 means take it if it exists and doesn't look like a flag,
		// other values are a discrete number to take
		takeArgs = func(narg int) func() bool {
			if narg < 0 {
				return func() bool {
					args = args[1:]
					return true
				}
			}
			if narg == 0 {
				narg = 1
			}
			return func() bool {
				narg = narg - 1
				if narg > 0 {
					args = args[1:]
				}
				return narg >= 0
			}
		}
	)

	taker := takeArgs(current.NArg)
	for {
		if len(args) == 0 {
			break
		}

		value := args[0]
		if !taker() {
			break
		}
		err := current.Set(value)
		if err != nil {
			return nil, err
		}
	}
	return args, nil
}

func (ctx *Context) executeBefores() error {
	if ctx == nil {
		return nil
	}

	err := ctx.Parent().executeBefores()
	if err != nil {
		return err
	}

	switch c := ctx.target.(type) {
	case *App:
		return hookExecute(Action(c.Before), defaultBeforeApp(c), ctx)
	case *Command:
		return hookExecute(Action(c.Before), defaultBeforeCommand(c), ctx)
	case option:
		return hookExecute(c.before(), defaultBeforeOption(c), ctx)
	}

	return nil
}

func (ctx *Context) executeCommand() error {
	cmd := ctx.target.(*Command)

	var (
		defaultAfter = emptyAction
	)

	if err := ctx.executeBefores(); err != nil {
		return err
	}

	return hookExecute(Action(cmd.Action), defaultAfter, ctx)
}

func (ctx *Context) executeOption() error {
	f := ctx.target.(option)

	var (
		defaultAfter = emptyAction
	)

	return hookExecute(f.action(), defaultAfter, ctx)
}

func defaultBeforeOption(o option) ActionFunc {
	return func(ctx *Context) error {
		if v, ok := loadFlagValueFromEnvironment(o); ok {
			return o.Set(v)
		}
		return nil
	}
}

func defaultBeforeCommand(c *Command) ActionFunc {
	return func(ctx *Context) error {
		for _, f := range c.flagsAndArgs() {
			err := hookExecute(f.before(), defaultBeforeOption(f), ctx)
			if err != nil {
				return err
			}
		}

		// Invoke the Before action on all flags and args, but only the actual
		// Action when the flag or arg was set
		for _, f := range c.flagsAndArgs() {
			if f.Seen() {
				err := ctx.optionContext(f).executeOption()
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func defaultBeforeApp(a *App) ActionFunc {
	return Pipeline(
		ActionFunc(setupDefaultIO),
		ActionFunc(setupDefaultData),
	)
}

func reverse(arr []string) []string {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return arr
}
