package cli

import (
	"sort"
	"strings"
)

type Command struct {
	Name        string
	Subcommands []*Command
	Flags       []*Flag
	Args        []*Arg
	Exprs       []*Expr
	Aliases     []string

	// Action specifies the action to run for the command, assuming no other more specific command
	// has been selected.  Refer to cli.Action about the correct function signature to use.
	Action interface{}

	// Before executes before the app action or any sub-command action runs.  Refer to
	// cli.Action about the correct function signature to use.
	Before interface{}

	// After executes after the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	After interface{}

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses interface{}

	// Category places the command into a category.  Categories are displayed on the default
	// help screen.
	Category string

	// Description provides a long description for the command.  The long description is
	// displayed on the help screen
	Description string

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	// Options sets common options for use with the command
	Options Option

	HelpText  string
	UsageText string

	cmdHooks            hooks
	didSetupDefaultArgs bool
	flags               internalFlags
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
	hasArguments
	hasFlags
	Command(string) (*Command, bool)
	Flag(string) (*Flag, bool)
	Arg(string) (*Arg, bool)
}

type commandContext struct {
	cmd                  *Command
	args_                []string
	set_                 *set
	didSubcommandExecute bool
}

const (
	onlyShortNoValue         = optionGroup(iota) // -v
	onlyShortNoValueOptional                     // [-v]
	onlyBoolLong                                 // [--[no-]support]
	otherOptional                                // [--long=value]
	other                                        // --long=value
	actionGroup                                  // { --help|--version}
	hidden
)

// ExecuteSubcommand finds and executes a sub-command.  This action is intended to be used
// as the action on an argument.  The argument should be a list of strings, which represent
// the subcommand to locate and execute and the arguments to use.  If no sub-command matches, an error
// is generated, which you can intercept with custom handling using interceptErr.  It is uncommon
// to use this action because this action is implicitly bound to a synthetic argument when a
// command defines any sub-commands.
func ExecuteSubcommand(interceptErr func(*Context, error) (*Command, error)) ActionFunc {
	return func(c *Context) error {
		invoke := c.List("")
		cmd, err := tryFindCommandOrIntercept(c, c.Command(), invoke[0], interceptErr)
		if err != nil {
			return err
		}
		c.Parent().internal.setDidSubcommandExecute()
		newCtx := c.Parent().commandContext(cmd, invoke)
		return cmd.parseAndExecuteSelf(newCtx)
	}
}

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

func (e *CommandCategory) Undocumented() bool {
	for _, x := range e.Commands {
		if x.HelpText != "" {
			return false
		}
	}
	return true
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
	return strings.Join(textUsage.command(c.newSynopsis()), " ")
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

func (c *Command) VisibleArgs() []*Arg {
	res := make([]*Arg, 0, len(c.actualArgs()))
	for _, o := range c.actualArgs() {
		if o.flags.hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (c *Command) VisibleFlags() []*Flag {
	res := make([]*Flag, 0, len(c.actualFlags()))
	for _, o := range c.actualFlags() {
		if o.flags.hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (c *Command) VisibleExprs() []*Expr {
	res := make([]*Expr, 0, len(c.actualExprs()))
	for _, o := range c.actualExprs() {
		if o.flags.hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (c *Command) Names() []string {
	return append([]string{c.Name}, c.Aliases...)
}

func (c *Command) appendArg(arg *Arg) *Command {
	c.Args = append(c.Args, arg)
	return c
}

func (c *Command) parseAndExecuteSelf(ctx *Context) error {
	c.setupDefaultArgs()
	ctx.applySet()

	if err := ctx.applyFlagsAndArgs(); err != nil {
		return err
	}

	return ctx.executeCommand()
}

func (c *Command) setupDefaultArgs() {
	if c.didSetupDefaultArgs {
		return
	}
	c.didSetupDefaultArgs = true
	c.ensureSubcommands()
	c.ensureExprs()
}

func (c *Command) ensureSubcommands() {
	if len(c.Subcommands) > 0 {
		if len(c.Args) > 0 {
			panic("cannot specify subcommands and arguments")
		}
		c.appendArg(&Arg{
			Name:      "command",
			UsageText: "<command> [<args>]",
			Value:     List(),
			NArg:      -1,
			Action:    ExecuteSubcommand(nil),
		})
		if c.Action == nil {
			c.Action = DisplayHelpScreen()
		}
	}
}

func (c *Command) ensureExprs() {
	if len(c.Exprs) > 0 {
		c.appendArg(&Arg{
			Name:      "expression",
			UsageText: "<expression>",
			Value:     new(exprPipeline),
			NArg:      -1,
			Action: BindExpression(func(c *Context) ([]*Expr, error) {
				return c.Command().Exprs, nil
			}),
		})
	}
}

func (c *Command) actualArgs() []*Arg {
	if c.Args == nil {
		return make([]*Arg, 0)
	}
	return c.Args
}

func (c *Command) actualExprs() []*Expr {
	if c.Exprs == nil {
		return make([]*Expr, 0)
	}
	return c.Exprs
}

func (c *Command) actualFlags() []*Flag {
	if c.Flags == nil {
		return make([]*Flag, 0)
	}
	return c.Flags
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

func (c *Command) setData(name string, v interface{}) {
	c.ensureData()[name] = v
}

func (c *Command) setCategory(name string) {
	c.Category = name
}

func (c *Command) ensureData() map[string]interface{} {
	if c.Data == nil {
		c.Data = map[string]interface{}{}
	}
	return c.Data
}

func (c *Command) hooks() *hooks {
	return &c.cmdHooks
}

func (c *Command) setInternalFlags(f internalFlags) {
	c.flags |= f
}

func (c *Command) internalFlags() internalFlags {
	return c.flags
}

func (c *Command) options() Option {
	return c.Options
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

func (c *commandContext) initialize(ctx *Context) error {
	// FIXME takeInitializers(c.cmd.Uses)
	if err := hookExecute(Action(c.cmd.Uses), defaultCommand.Uses, ctx); err != nil {
		return err
	}

	for _, sub := range c.cmd.Flags {
		err := ctx.optionContext(sub).initialize()
		if err != nil {
			return err
		}
	}
	for _, sub := range c.cmd.Args {
		err := ctx.optionContext(sub).initialize()
		if err != nil {
			return err
		}
	}
	for _, sub := range c.cmd.Exprs {
		err := ctx.exprContext(sub, nil, nil).initialize()
		if err != nil {
			return err
		}
	}
	c.cmd.setupDefaultArgs()
	return nil
}

func (c *commandContext) executeBeforeDescendent(ctx *Context) error {
	return ctx.executeBeforeHooks(c.cmd)
}

func (c *commandContext) executeBefore(ctx *Context) error {
	if err := ctx.executeBeforeHooks(c.cmd); err != nil {
		return err
	}
	return hookExecute(Action(c.cmd.Before), defaultCommand.Before, ctx)
}

func (c *commandContext) executeAfter(ctx *Context) error {
	if err := ctx.executeAfterHooks(c.cmd); err != nil {
		return err
	}
	return hookExecute(Action(c.cmd.After), defaultCommand.After, ctx)
}

func (c *commandContext) executeAfterDescendent(ctx *Context) error {
	return ctx.executeAfterHooks(c.cmd)
}

func (c *commandContext) execute(ctx *Context) error {
	if !c.didSubcommandExecute {
		return actionOrEmpty(c.cmd.Action).Execute(ctx)
	}
	return nil
}

func (c *commandContext) hooks() *hooks {
	return &c.cmd.cmdHooks
}

func (c *commandContext) app() (*App, bool) { return nil, false }
func (c *commandContext) set() *set {
	if c.set_ == nil {
		c.set_ = newSet()
	}
	return c.set_
}
func (c *commandContext) args() []string { return c.args_ }
func (c *commandContext) target() target { return c.cmd }
func (c *commandContext) lookupValue(name string) (interface{}, bool) {
	return c.set().lookupValue(name)
}

func (c *commandContext) setDidSubcommandExecute() {
	c.didSubcommandExecute = true
}

func (c *commandContext) Name() string {
	return c.cmd.Name
}

func getGroup(f *Flag) optionGroup {
	if f.internalFlags().hidden() {
		return hidden
	}
	if f.internalFlags().exits() {
		return actionGroup
	}
	if hasOnlyShortName(f) && hasNoValue(f) {
		if f.internalFlags().required() {
			return onlyShortNoValue
		}
		return onlyShortNoValueOptional
	}
	if f.internalFlags().required() {
		return other
	}
	return otherOptional
}

func sortedByName(flags []*flagSynopsis) {
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].short < flags[j].short
	})
}

func findCommandByName(cmds []*Command, name string) (*Command, bool) {
	for _, sub := range cmds {
		if sub.Name == name {
			return sub, true
		}
		for _, alias := range sub.Aliases {
			if alias == name {
				return sub, true
			}
		}
	}
	return nil, false
}

func tryFindCommandOrIntercept(c *Context, cmd *Command, sub string, interceptErr func(*Context, error) (*Command, error)) (*Command, error) {
	if res, ok := cmd.Command(sub); ok {
		return res, nil
	}
	if interceptErr != nil {
		res, err := interceptErr(c, commandMissing(sub))
		if res != nil || err != nil {
			return res, err
		}
	}
	return nil, commandMissing(sub)
}

var _ command = &Command{}
