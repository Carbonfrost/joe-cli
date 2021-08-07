package cli

import (
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

func (c *Command) createAndApplySet() *getopt.Set {
	set := getopt.New()
	for _, f := range c.actualFlags() {
		f.applyToSet(set)
	}
	return set
}

func (c *Command) parseAndExecute(ctx *Context, args []string) error {
	ctx, err := ctx.commandContext(c, args).applySubcommands()
	if err != nil {
		return err
	}

	if err := ctx.applyFlagsAndArgs(); err != nil {
		return err
	}

	return ctx.executeCommand()
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
