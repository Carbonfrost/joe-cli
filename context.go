package cli

import (
	"context"

	"github.com/pborman/getopt/v2"
)

// Context provides the context in which the app, command, or flag is executing
type Context struct {
	context.Context
	parent *Context

	target interface{} // *Command, *Flag, or *Arg

	// When the context is being used for a command
	args []string
	set  *getopt.Set
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

func (*Context) Value(name string) interface{} {
	panic("not implemented: context value")
}

func rootContext(cctx context.Context, app *App) *Context {
	return &Context{
		Context: cctx,
		target:  app,
	}
}

func (c *Context) commandContext(cmd *Command, args []string) *Context {
	return &Context{
		Context: c.Context,
		target:  cmd,
		args:    args,
		parent:  c,
		set:     cmd.createAndApplySet(),
	}
}

func (c *Context) optionContext(opt option) *Context {
	return &Context{
		Context: c.Context,
		target:  opt,
		parent:  c,
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
			err = unexpectedArgument(args[0])
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

		if len(args) > 0 {
			err = current.Set(args[0])
			if err != nil {
				return
			}
		} else {
			break
		}
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
	return nil
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
	return nil
}
