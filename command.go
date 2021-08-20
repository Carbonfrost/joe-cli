package cli

import (
	"bytes"
	"sort"
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

	// Category places the command into a category.  Categories are displayed on the default
	// help screen.
	Category string

	// Description provides a long description for the command.  The long description is
	// displayed on the help screen
	Description string

	HelpText  string
	UsageText string
}

// CommandsByName provides a slice that can sort on name
type CommandsByName []*Command

type CommandCategory struct {
	Category string
	Commands []*Command
}

type CommandsByCategory []*CommandCategory

type commandSynopsis struct {
	name  string
	flags map[optionGroup][]*flagSynopsis
	args  []*argSynopsis
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

func GroupedByCategory(cmds []*Command) CommandsByCategory {
	res := CommandsByCategory{}
	for _, command := range cmds {
		cc := res.Category(command.Category)
		if cc == nil {
			cc = &CommandCategory{
				Category: command.Category,
				Commands: []*Command{},
			}
			res = append(res, cc)
		}
		cc.Commands = append(cc.Commands, command)
	}
	sort.Sort(res)
	return res
}

func (c CommandsByCategory) Category(name string) *CommandCategory {
	for _, cc := range c {
		if cc.Category == name {
			return cc
		}
	}
	return nil
}

func (c CommandsByCategory) Less(i, j int) bool {
	return c[i].Category < c[j].Category
}

func (c CommandsByCategory) Len() int {
	return len(c)
}

func (c CommandsByCategory) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

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

func (c *Command) createValues() map[string]interface{} {
	values := map[string]interface{}{}
	for _, f := range c.actualFlags() {
		values[f.Name] = f.value()
	}
	for _, f := range c.actualArgs() {
		values[f.Name] = f.value()
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

	for _, a := range c.args {
		res.WriteString(" ")
		res.WriteString(a.formatString())
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
	args := make([]*argSynopsis, 0)
	for _, f := range c.actualFlags() {
		group := getGroup(f)
		groups[group] = append(groups[group], f.newSynopsis())
	}
	for _, a := range c.actualArgs() {
		args = append(args, a.newSynopsis())
	}

	sortedByName(groups[onlyShortNoValueOptional])
	sortedByName(groups[onlyShortNoValue])

	return &commandSynopsis{
		name:  c.Name,
		flags: groups,
		args:  args,
	}
}

func (c CommandsByName) Len() int {
	return len(c)
}

func (c CommandsByName) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c CommandsByName) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
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
