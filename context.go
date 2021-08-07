package cli

import (
	"context"

	"github.com/pborman/getopt"
)

// Context provides the context in which the app, command, or flag is executing
type Context struct {
	context.Context
	parent *Context

	// When the context is being used for a command
	command *Command
	args    []string
	set     *getopt.Set
}

func (c *Context) Parent() *Context {
	return c.parent
}

func (c *Context) Command() *Command {
	if c.command != nil {
		return c.command
	}
	if c.Parent() != nil {
		return c.Parent().Command()
	}
	return nil
}

func (*Context) Value(name string) interface{} {
	panic("not implemented: context value")
}

func rootContext(cctx context.Context) *Context {
	ctx := &Context{
		Context: cctx,
		set:     getopt.New(),
	}
	return ctx
}

func (c *Context) commandContext(cmd *Command, args []string) *Context {
	return &Context{
		Context: c.Context,
		command: cmd,
		args:    args,
		parent:  c,
		set:     cmd.createAndApplySet(),
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
			if sub, ok := ctx.command.Command(args[0]); ok {
				ctx = ctx.commandContext(sub, args)
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
		current      Arg

		// ctx.args contains the name of the command and its arguments
		args []string = ctx.args

		enumerator = func() bool {
			actual := ctx.command.actualArgs()
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

func (ctx *Context) executeCommand() error {
	var (
		defaultBefore = func(*Context) error {
			return nil
		}
		defaultAfter = emptyActionImpl
	)

	cmd := ctx.command
	if err := hookExecute(cmd.Before, defaultBefore, ctx); err != nil {
		return err
	}

	return hookExecute(cmd.Action, defaultAfter, ctx)
}
