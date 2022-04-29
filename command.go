package cli

import (
	"sort"
)

// Command represents a command with arguments, flags, and expressions
type Command struct {
	hooksSupport
	pipelinesSupport
	customizableSupport

	// Name of the command
	Name string

	// Subcommands provides sub-commands that compose the command.
	Subcommands []*Command

	// Flags that the command supports
	Flags []*Flag

	// Args that the command supports
	Args []*Arg

	// Aliases indicates alternate names that can be used
	Aliases []string

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
	// Actions within the Uses and Before pipelines can modify the app Commands and Flags lists.  Any
	// commands or flags added to the list will be initialized
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

	// HelpText describes the help text displayed for commands
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the command.  If left blank, a succint synopsis
	// is generated that lists each visible flag and arg
	UsageText string

	flags internalFlags
}

// CommandsByName provides a slice that can sort on name
type CommandsByName []*Command

// CommandCategory names a category and the commands it contains
type CommandCategory struct {
	// Category is the name of the category
	Category string
	// Commands in the category
	Commands []*Command
}

// CommandsByCategory provides a slice that can sort on category names and the commands
// themselves
type CommandsByCategory []*CommandCategory

type commandSynopsis struct {
	Name         string
	Flags        map[optionGroup][]*flagSynopsis
	Args         []*argSynopsis
	RequiredArgs []*argSynopsis
	OptionalArgs []*argSynopsis
	RTL          bool
}

type optionGroup int

type commandContext struct {
	cmd                  *Command
	flagSet              *set
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
		newCtx := c.Parent().commandContext(cmd).setTiming(ActionTiming)
		return cmd.parseAndExecuteSelf(newCtx, invoke)
	}
}

// GroupedByCategory will group the commands by category and sort the commands
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

// Category gets a category by name
func (c CommandsByCategory) Category(name string) *CommandCategory {
	for _, cc := range c {
		if cc.Category == name {
			return cc
		}
	}
	return nil
}

// Undocumented determines whether the category is undocumented (i.e. has no HelpText set
// on any of its commands)
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

// Synopsis returns the UsageTexzt for the command or produces a succint representation
// that names each flag and arg
func (c *Command) Synopsis() string {
	return sprintSynopsis("CommandSynopsis", c.newSynopsis())
}

// Command tries to obtain a sub-command by name or alias
func (c *Command) Command(name string) (*Command, bool) {
	return findCommandByName(c.Subcommands, name)
}

// Flag tries to obtain a flag by name or alias
func (c *Command) Flag(name string) (*Flag, bool) {
	return findFlagByName(c.Flags, name)
}

// Arg tries to obtain a arg by name or alias
func (c *Command) Arg(name string) (*Arg, bool) {
	return findArgByName(c.Args, name)
}

// VisibleArgs filters all arguments in the command by whether they are not hidden
func (c *Command) VisibleArgs() []*Arg {
	res := make([]*Arg, 0, len(c.actualArgs()))
	for _, o := range c.actualArgs() {
		if o.internalFlags().hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

// VisibleFlags filters all flags in the command by whether they are not hidden
func (c *Command) VisibleFlags() []*Flag {
	res := make([]*Flag, 0, len(c.actualFlags()))
	for _, o := range c.actualFlags() {
		if o.internalFlags().hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

// Names obtains the name of the command and its aliases
func (c *Command) Names() []string {
	return append([]string{c.Name}, c.Aliases...)
}

func (c *Command) appendArg(arg *Arg) *Command {
	c.Args = append(c.Args, arg)
	return c
}

func (c *Command) parseAndExecuteSelf(ctx *Context, args []string) error {
	ctx.argList = args
	set := c.buildSet(ctx)
	if c.internalFlags().skipFlagParsing() {
		args = append([]string{args[0], "--"}, args[1:]...)
	}

	err := set.parse(args, c.internalFlags().disallowFlagsAfterArgs())
	if err != nil {
		return err
	}

	return ctx.executeCommand()
}

func (c *Command) buildSet(ctx *Context) *set {
	set := newSet(c.internalFlags().rightToLeft())
	switch internal := ctx.internal.(type) {
	case *commandContext:
		internal.flagSet = set
	case *appContext:
		internal.commandContext.flagSet = set
	}

	for _, f := range c.actualFlags() {
		f.applyToSet(set)
	}
	if ctx.Parent() != nil {
		for _, f := range ctx.Parent().flags(true) {
			if f.internalFlags().nonPersistent() {
				continue
			}
			f.applyToSet(set)
			f.(*Flag).option.flags |= internalFlagPersistent
		}
	}
	for _, a := range c.actualArgs() {
		a.applyToSet(set)
	}
	return set
}

func ensureSubcommands(c *Context) error {
	c.target().(*Command).ensureSubcommands()
	return nil
}

func (c *Command) ensureSubcommands() {
	if len(c.Subcommands) > 0 {
		c.appendArg(&Arg{
			Name:      "command",
			UsageText: "<command> [<args>]",
			Value:     List(),
			NArg:      -1,
			Action:    ExecuteSubcommand(nil),
			Options:   DisableSplitting,
		})
		if c.Action == nil {
			c.Action = DisplayHelpScreen()
		}
	}
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

func (c *Command) newSynopsis() *commandSynopsis {
	groups := map[optionGroup][]*flagSynopsis{
		onlyShortNoValue:         {},
		onlyShortNoValueOptional: {},
		onlyBoolLong:             {},
		hidden:                   {},
		otherOptional:            {},
		actionGroup:              {},
		other:                    {},
	}
	args := make([]*argSynopsis, 0)
	for _, f := range c.actualFlags() {
		group := getGroup(f)
		groups[group] = append(groups[group], f.synopsis())
	}
	for _, a := range c.actualArgs() {
		args = append(args, a.newSynopsis())
	}

	sortedByName(groups[onlyShortNoValueOptional])
	sortedByName(groups[onlyShortNoValue])

	var required []*argSynopsis
	var optional []*argSynopsis

	rtl := c.internalFlags().rightToLeft()
	if rtl {
		var start int
		for i, p := range args {
			if p.Optional {
				start = i
				break
			}
		}
		required = args[0:start]
		optional = args[start:]
	} else {
		for _, p := range args {
			if p.Optional {
				optional = append(optional, p)
			} else {
				required = append(required, p)
			}
		}
	}

	return &commandSynopsis{
		Name:         c.Name,
		Flags:        groups,
		Args:         args,
		RequiredArgs: required,
		OptionalArgs: optional,
		RTL:          rtl,
	}
}

// SetData sets the specified metadata on the command
func (c *Command) SetData(name string, v interface{}) {
	c.ensureData()[name] = v
}

// LookupData obtains the data if it exists
func (c *Command) LookupData(name string) (interface{}, bool) {
	v, ok := c.ensureData()[name]
	return v, ok
}

func (c *Command) setCategory(name string) {
	c.Category = name
}

func (c *Command) setManualText(name string) {
	c.ManualText = name
}

func (c *Command) setHelpText(name string) {
	c.HelpText = name
}

func (c *Command) setDescription(name string) {
	c.Description = name
}

func (c *Command) ensureData() map[string]interface{} {
	if c.Data == nil {
		c.Data = map[string]interface{}{}
	}
	return c.Data
}

func (c *Command) setInternalFlags(f internalFlags) {
	c.flags |= f
}

func (c *Command) internalFlags() internalFlags {
	return c.flags
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
	rest := newPipelines(ActionOf(c.cmd.Uses), &c.cmd.Options)
	c.cmd.setPipelines(rest)
	return c.initializeCore(ctx)
}

func (c *commandContext) initializeCore(ctx *Context) error {
	return executeAll(ctx, ActionOf(c.cmd.uses().Initializers), defaultCommand.Initializers)
}

func initializeFlagsArgs(ctx *Context) error {
	var (
		flagStart   int
		argStart    int
		_, anyFlags = ctx.target().(hasFlags)
		_, anyArgs  = ctx.target().(hasArguments)
	)

	// New flags and/or args may have been introduced, so allow these to also initialize.
	// They can ONLY be appended to the slice, not inserted elsewhere
	for anyFlags || anyArgs {
		if cmd, ok := ctx.target().(hasFlags); ok {
			flags := cmd.actualFlags()[flagStart:]
			flagStart = len(cmd.actualFlags())

			for _, sub := range flags {
				err := ctx.flagContext(sub, nil).initialize()
				if err != nil {
					return err
				}
			}
			anyFlags = len(flags) > 0
		} else {
			anyFlags = false
		}

		if cmd, ok := ctx.target().(hasArguments); ok {
			args := cmd.actualArgs()[argStart:]
			argStart = len(cmd.actualArgs())

			for _, sub := range args {
				err := ctx.argContext(sub, nil).initialize()
				if err != nil {
					return err
				}
			}
			anyArgs = len(args) > 0
		} else {
			anyArgs = false
		}
	}

	return nil
}

func initializeSubcommands(ctx *Context) error {
	cmd := ctx.target().(*Command)
	for _, sub := range cmd.Subcommands {
		err := ctx.commandContext(sub).initialize()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *commandContext) executeBeforeDescendent(ctx *Context) error {
	return c.cmd.executeBeforeHooks(ctx)
}

func (c *commandContext) executeBefore(ctx *Context) error {
	if err := c.cmd.executeBeforeHooks(ctx); err != nil {
		return err
	}

	if err := execute(c.cmd.uses().Before, ctx); err != nil {
		return err
	}

	return executeAll(ctx, ActionOf(c.cmd.Before), defaultCommand.Before)
}

func (c *commandContext) executeAfter(ctx *Context) error {
	if err := c.cmd.executeAfterHooks(ctx); err != nil {
		return err
	}
	return executeAll(ctx, c.cmd.uses().After, ActionOf(c.cmd.After), defaultCommand.After)
}

func (c *commandContext) executeAfterDescendent(ctx *Context) error {
	return c.cmd.executeAfterHooks(ctx)
}

func (c *commandContext) execute(ctx *Context) error {
	if !c.didSubcommandExecute {
		return executeAll(ctx, c.cmd.uses().Action, ActionOf(c.cmd.Action))
	}
	return nil
}

func (c *commandContext) lookupBinding(name string) []string {
	return c.flagSet.bindings[name]
}
func (c *commandContext) target() target { return c.cmd }
func (c *commandContext) lookupValue(name string) (interface{}, bool) {
	return c.flagSet.lookupValue(name)
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
		return flags[i].Short < flags[j].Short
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

var _ targetConventions = (*Command)(nil)
var _ hookable = (*Command)(nil)
