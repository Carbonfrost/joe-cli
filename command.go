package cli

import (
	"bytes"
	"sort"

	"github.com/pborman/getopt/v2"
)

type Command struct {
	Name        string
	Subcommands []*Command
	Flags       []*Flag
	Args        []*Arg

	// Action specifies the action to run for the command, assuming no other more specific command
	// has been selected.  Refer to cli.Action about the correct function signature to use.
	Action interface{}

	// Before executes before the app action or any sub-command action runs.  Refer to
	// cli.Action about the correct function signature to use.
	Before interface{}

	HelpText  string
	UsageText string
}

type commandSynopsis struct {
	name  string
	flags map[optionGroup][]*flagSynopsis
}

type optionGroup int

type command interface {
	Command(string) (*Command, bool)
	Flag(string) (*Flag, bool)
	Arg(string) (*Arg, bool)
}

const (
	onlyShortNoValue         = optionGroup(iota) // -v
	onlyShortNoValueOptional                     // [-v]
	onlyBoolLong                                 // [--[no-]support]
	otherOptional                                // [--long=value]
	other                                        // --long=value
	hidden
)

func (c *Command) Synopsis() string {
	return c.newSynopsis().formatString()
}

func (c *Command) Command(name string) (*Command, bool) {
	return findCommandByName(c.Subcommands, name)
}

func (c *Command) Flag(name string) (*Flag, bool) {
	return findFlagByName(c.Flags, name)
}

func (c *Command) Arg(name string) (*Arg, bool) {
	return findArgByName(c.Args, name)
}

func (c *Command) createAndApplySet() *getopt.Set {
	set := getopt.New()
	for _, f := range c.actualFlags() {
		f.applyToSet(set)
	}
	return set
}

func (c *Command) createValues() map[string]interface{} {
	values := map[string]interface{}{}
	for _, f := range c.actualFlags() {
		values[f.Name] = f.Value
	}
	for _, f := range c.actualArgs() {
		values[f.Name] = f.Value
	}
	return values
}

func (c *Command) parseAndExecute(ctx *Context, args []string) error {
	ctx = ctx.commandContext(c, args)
	err := ctx.executeBefore()
	if err != nil {
		return err
	}

	ctx, err = ctx.applySubcommands()
	if err != nil {
		return err
	}

	if err := ctx.applyFlagsAndArgs(); err != nil {
		return err
	}

	return ctx.executeCommand()
}

func (c *Command) actualArgs() []*Arg {
	if c.Args == nil {
		return make([]*Arg, 0)
	}
	return c.Args
}

func (c *Command) actualFlags() []*Flag {
	if c.Flags == nil {
		return make([]*Flag, 0)
	}
	return c.Flags
}

func (c *Command) flagsAndArgs() []option {
	res := make([]option, 0, len(c.Flags)+len(c.Args))
	for _, f := range c.Flags {
		res = append(res, f)
	}
	for _, a := range c.Args {
		res = append(res, a)
	}
	return res
}

func (c *commandSynopsis) formatString() string {
	var res bytes.Buffer
	res.WriteString(c.name)

	groups := c.flags
	// short option list -abc
	if len(groups[onlyShortNoValue]) > 0 {
		res.WriteString(" -")
		for _, f := range groups[onlyShortNoValue] {
			res.WriteString(f.short)
		}
	}

	if len(groups[onlyShortNoValueOptional]) > 0 {
		res.WriteString(" [-")
		for _, f := range groups[onlyShortNoValueOptional] {
			res.WriteString(f.short)
		}
		res.WriteString("]")
	}

	for _, f := range groups[otherOptional] {
		res.WriteString(" [")
		res.WriteString(f.formatString(true))
		res.WriteString("]")
	}

	for _, f := range groups[other] {
		res.WriteString(" ")
		res.WriteString(f.formatString(true))
	}

	return res.String()
}

func (c *Command) newSynopsis() *commandSynopsis {
	groups := map[optionGroup][]*flagSynopsis{
		onlyShortNoValue:         []*flagSynopsis{},
		onlyShortNoValueOptional: []*flagSynopsis{},
		onlyBoolLong:             []*flagSynopsis{},
		hidden:                   []*flagSynopsis{},
		otherOptional:            []*flagSynopsis{},
		other:                    []*flagSynopsis{},
	}
	for _, f := range c.actualFlags() {
		group := getGroup(f)
		groups[group] = append(groups[group], f.newSynopsis())
	}

	sortedByName(groups[onlyShortNoValueOptional])
	sortedByName(groups[onlyShortNoValue])

	return &commandSynopsis{
		name:  c.Name,
		flags: groups,
	}
}

func getGroup(f *Flag) optionGroup {
	if f.Hidden {
		return hidden
	}
	if hasOnlyShortName(f) && hasNoValue(f) {
		if f.Required {
			return onlyShortNoValue
		}
		return onlyShortNoValueOptional
	}
	if f.Required {
		return other
	}
	return otherOptional
}

func sortedByName(flags []*flagSynopsis) {
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].short < flags[j].short
	})
}

func findCommand(current *Command, commands []string) (*Command, error) {
	for _, c := range commands {
		var ok bool
		current, ok = current.Command(c)
		if !ok {
			return nil, commandMissing(c)
		}
	}
	return current, nil
}

func findCommandByName(cmds []*Command, name string) (*Command, bool) {
	for _, sub := range cmds {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}

var _ command = &Command{}
