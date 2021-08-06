package cli

import (
	"context"

	"github.com/pborman/getopt"
)

type Command struct {
	Name        string
	Subcommands []*Command
	Flags       []Flag
	Args        []Arg
	Action      ActionFunc
	Before      ActionFunc
}

func (c *Command) Command(name string) (*Command, bool) {
	for _, sub := range c.Subcommands {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}

func (c *Command) applyArgs(ctx *Context, args []string) error {
	for _, arg := range c.actualArgs() {
		err := arg.Getopt(args)
		if err != nil {
			// Failed to set the option to the corresponding flag
			return err
		}
	}
	if len(args) > 0 {
		panic("not implemented: apply rest of args to command")
	}
	return nil
}

func (c *Command) newContext(cctx context.Context, parent *Context) *Context {
	ctx := &Context{
		Context: cctx,
		set:     getopt.New(),
	}
	for _, f := range c.actualFlags() {
		f.applyToSet(ctx.set)
	}
	return ctx
}

func (c *Command) actualArgs() []Arg {
	if c.Args == nil {
		return make([]Arg, 0)
	}
	return c.Args
}

func (c *Command) actualFlags() []Flag {
	if c.Flags == nil {
		return make([]Flag, 0)
	}
	return c.Flags
}
