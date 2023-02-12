package cli

import (
	"sort"
	"strings"
	"text/template"
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
	// displayed on the help screen.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description interface{}

	// Comment provides a short descriptive comment.  This is
	// usually a few words to summarize the purpose of the command.
	Comment string

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

	// Completion provides the completion for use in the command.  By default, the
	// completion detects whether a flag or arg is being used and then delegates to
	// the completion present there
	Completion Completion

	flags   internalFlags
	fromApp *App
	ifRoot  *rootCommandData
}

type rootCommandData struct {
	templateFuncs map[string]interface{}
	templates     *template.Template
}

type robustParseResult struct {
	bindings BindingMap
	err      error
}

// CommandsByName provides a slice that can sort on name
type CommandsByName []*Command

type commandCategory struct {
	Category string
	Commands []*Command
}

type commandsByCategory []*commandCategory

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
	args                 []string
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

const (
	commandNotFoundKey           = "__CommandNotFound"
	searchingAlternateCommandKey = "__SearchingAlternateCommand"
)

// ExecuteSubcommand finds and executes a sub-command.  This action is intended to be used
// as the action on an argument.  The argument should be a list of strings, which represent
// the subcommand to locate and execute and the arguments to use.  If used within the
// Uses pipeline of an argument, a prototype applies these requirements for you and other
// good defaults to support completion and synopsis.  If no sub-command matches, an error
// is generated, which you can intercept with custom handling using interceptErr.  The interceptErr function
// should return a command to execute in lieu of returning the error.  If the interceptErr
// command is nil, it is interpreted as the command not existing and the app will exit with a generic "command
// not found error" message.  If it returns an error, then executing the sub-command fails with the error.
// However, if SkipCommand is returned, then no command is executed, and no error is generated.
// It is uncommon to use this action because this action is implicitly bound to a synthetic argument when a
// command defines any sub-commands.
func ExecuteSubcommand(interceptErr func(*Context, error) (*Command, error)) Action {
	return Pipeline(&Prototype{
		Name:       "command",
		UsageText:  "<command> [<args>]",
		Value:      List(),
		NArg:       -1,
		Options:    DisableSplitting,
		Completion: CompletionFunc(completeSubCommand),
	}, AtTiming(ActionFunc(func(c *Context) error {
		invoke := c.List("")
		return subcommandCore(c, invoke, interceptErr)
	}), ActionTiming))
}

func subcommandCore(c *Context, invoke []string, interceptErr func(*Context, error) (*Command, error)) error {
	cmd, err := tryFindCommandOrIntercept(c, c.Command(), invoke[0], interceptErr)
	if err == SkipCommand {
		return nil
	}
	if err != nil {
		return err
	}
	c.Parent().internal.setDidSubcommandExecute()
	newCtx := c.Parent().commandContext(cmd, invoke)
	return newCtx.Execute(invoke)
}

// HandleCommandNotFound assigns a default function to invoke when a command cannot be found.
// The specified function is invoked if a command cannot be found.  It contains the context of the
// parent attempting to invoke a command and the error previously encountered.  It returns the
// command if any that can substitute.  Composition occurs with functions registered to handle
// commands not found.  They each get called until one returns a command.
func HandleCommandNotFound(fn func(*Context, error) (*Command, error)) Action {
	return ActionFunc(func(c *Context) error {
		cmd := c.Command()
		if existing, ok := cmd.Data[commandNotFoundKey]; ok {
			// Compose functions
			newFn := fn
			fn = func(c *Context, err1 error) (*Command, error) {
				cmd, err := newFn(c, err1)
				if cmd != nil && err == nil {
					return cmd, nil
				}
				return existing.(func(*Context, error) (*Command, error))(c, err)
			}
		}
		c.SetData(commandNotFoundKey, fn)
		return nil
	})
}

// ImplicitCommand indicates the command which is implicit when no sub-command matches.
// The main use case for this is to allow a command to be invoked by default without being
// named.  For example, you might have a sub-command called "exec" which can be omitted, making
// the following invocations equivalent:
//
//   - cloud exec tail -f /var/output/log
//   - cloud tail -f /var/output/log
func ImplicitCommand(name string) Action {
	return HandleCommandNotFound(func(c *Context, _ error) (*Command, error) {
		invoke := append([]string{name}, c.Args()...)
		err := subcommandCore(c, invoke, nil)
		if err != nil {
			return nil, err
		}

		return nil, SkipCommand
	})
}

func groupedByCategory(cmds []*Command) commandsByCategory {
	res := commandsByCategory{}
	for _, command := range cmds {
		cc := res.Category(command.Category)
		if cc == nil {
			cc = &commandCategory{
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

func (c commandsByCategory) Category(name string) *commandCategory {
	for _, cc := range c {
		if cc.Category == name {
			return cc
		}
	}
	return nil
}

// Undocumented determines whether the category is undocumented (i.e. has no HelpText set
// on any of its commands)
func (e *commandCategory) Undocumented() bool {
	for _, x := range e.Commands {
		if x.HelpText != "" {
			return false
		}
	}
	return true
}

func (c commandsByCategory) Less(i, j int) bool {
	return c[i].Category < c[j].Category
}

func (c commandsByCategory) Len() int {
	return len(c)
}

func (c commandsByCategory) Swap(i, j int) {
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
func (c *Command) Arg(name interface{}) (*Arg, bool) {
	a, _, ok := findArgByName(c.Args, name)
	return a, ok
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

// VisibleSubcommands filters all sub-commands in the command by whether they are not hidden
func (c *Command) VisibleSubcommands() []*Command {
	res := make([]*Command, 0, len(c.Subcommands))
	for _, o := range c.Subcommands {
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

// Use appends actions to Uses pipeline
func (c *Command) Use(actions ...Action) *Command {
	c.Uses = Pipeline(c.Uses).Append(actions...)
	return c
}

func (c *Command) buildSet(ctx *Context) *set {
	set := ctx.internal.(*commandContext).flagSet
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
	cmd := c.target().(*Command)

	if len(cmd.Subcommands) > 0 {
		if cmd.Action == nil {
			cmd.Action = DisplayHelpScreen()
		}
		return c.Do(AddArg(&Arg{
			Name: "command",
			Uses: ExecuteSubcommand(nil),
		}))
	}
	return nil
}

func completeSubCommand(cc *CompletionContext) []CompletionItem {
	c := cc.Context
	invoke := c.List("")
	detect := func(s string) bool {
		return strings.HasPrefix(s, cc.Incomplete)
	}

	if len(invoke) == 0 {
		cmd := c.Command()
		res := make([]CompletionItem, 0, len(cmd.Subcommands))

		for _, s := range cmd.Subcommands {
			if detect(s.Name) {
				res = append(res, CompletionItem{Value: s.Name, HelpText: s.HelpText})
			}
		}
		for _, s := range cmd.Subcommands {
			for _, alias := range s.Aliases {
				if detect(alias) {
					res = append(res, CompletionItem{Value: alias, HelpText: s.HelpText})
				}
			}
		}
		return res
	}

	cmd, err := tryFindCommandOrIntercept(c, c.Command(), invoke[0], nil)
	if err != nil {
		return nil
	}

	newCtx := c.Parent().commandContext(cmd, invoke)
	return newCtx.Complete(invoke, cc.Incomplete)
}

func (c *Command) completion() Completion {
	if c.Completion != nil {
		return c.Completion
	}
	return CompletionFunc(defaultCommandCompletion)
}

func defaultCommandCompletion(cc *CompletionContext) []CompletionItem {
	cmd := cc.Context.Target().(*Command)
	var items []CompletionItem

	if strings.HasPrefix(cc.Incomplete, "-") {
		// If a search only finds one match, then complete the flag
		items = findSolitaryMatch(cc)
		if items != nil {
			return items
		}

		for _, f := range cmd.VisibleFlags() {
			for _, n := range f.synopsis().Names {
				if strings.HasPrefix(n, cc.Incomplete) {
					items = append(items, CompletionItem{Value: n, HelpText: f.HelpText})
				}
			}
		}
		return items
	}

	if cc.Err != nil {
		name := cc.Err.(*ParseError).Name

		if strings.HasPrefix(name, "-") {
			flag, ok := cmd.Flag(name)
			if ok {
				return actualCompletion(flag.completion()).Complete(cc)
			}
		}

		arg, ok := cmd.Arg(name)
		if !ok && len(cmd.Args) > 0 {
			arg = cmd.Args[0]
		}
		if arg == nil {
			return nil
		}
		return actualCompletion(arg.completion()).Complete(cc)
	}

	// Request completion of the last argument that was seen
	if len(cmd.Args) > 0 {
		last := cmd.Args[0]
		for _, a := range cmd.Args {
			last = a
			if len(cc.Bindings[a.Name]) == 0 {
				break
			}
		}
		return actualCompletion(last.completion()).Complete(cc.optionContext(last))
	}

	return items
}

func findSolitaryMatch(cc *CompletionContext) []CompletionItem {
	cmd := cc.Context.Target().(*Command)
	flagName, _, hasArg := strings.Cut(cc.Incomplete, "=")
	var match *Flag
	var matchName string

	for _, f := range cmd.VisibleFlags() {
		for _, n := range f.synopsis().Names {
			if n == cc.Incomplete || (hasArg && strings.HasPrefix(n, flagName)) {
				return actualCompletion(f.completion()).Complete(cc.optionContext(f))
			}
			if strings.HasPrefix(n, cc.Incomplete) {
				if match != nil && match != f {
					return nil
				}
				match, matchName = f, n
			}
		}
	}
	if match == nil {
		return nil
	}

	var suffix string
	if !match.internalFlags().flagOnly() && len(matchName) > 2 {
		suffix = "="
	}
	return []CompletionItem{
		{Value: matchName + suffix, HelpText: match.HelpText, PreventSpaceAfter: len(suffix) > 0},
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

func (c *Command) SetHidden(value bool) {
	c.setInternalFlags(internalFlagHidden, value)
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

func (c *Command) setDescription(name interface{}) {
	c.Description = name
}

func (c *Command) setCompletion(cv Completion) {
	c.Completion = cv
}

func (c *Command) ensureData() map[string]interface{} {
	if c.Data == nil {
		c.Data = map[string]interface{}{}
	}
	return c.Data
}

func (c *Command) setInternalFlags(f internalFlags, v bool) {
	if v {
		c.flags |= f
	} else {
		c.flags = c.flags & ^f
	}
}

func (c *Command) internalFlags() internalFlags {
	return c.flags
}

func (c *Command) rootData() *rootCommandData {
	if c.ifRoot == nil {
		c.ifRoot = newRootCommandData()
	}
	return c.ifRoot
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
	return execute(ctx, Pipeline(c.cmd.uses().Initializers, defaultCommand.Initializers))
}

func rootCommandInitializers(act Action) ActionFunc {
	return func(c *Context) error {
		if c.Command().fromApp == nil {
			return nil
		}

		return c.Do(act)
	}
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
				err := ctx.optionContext(sub).initialize()
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
				err := ctx.optionContext(sub).initialize()
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
		err := ctx.commandContext(sub, nil).initialize()
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

	if err := execute(ctx, c.cmd.uses().Before); err != nil {
		return err
	}

	return execute(ctx, Pipeline(c.cmd.Before, defaultCommand.Before))
}

func (c *commandContext) executeAfter(ctx *Context) error {
	if err := c.cmd.executeAfterHooks(ctx); err != nil {
		return err
	}
	return execute(ctx, Pipeline(c.cmd.uses().After, c.cmd.After, defaultCommand.After))
}

func (c *commandContext) executeAfterDescendent(ctx *Context) error {
	return c.cmd.executeAfterHooks(ctx)
}

func (c *commandContext) execute(ctx *Context) error {
	if !c.didSubcommandExecute {
		return execute(ctx, Pipeline(c.cmd.uses().Action, defaultCommand.Action, c.cmd.Action))
	}
	return nil
}

func (c *commandContext) lookupBinding(name string, occurs bool) []string {
	if name == "" {
		return c.args
	}
	return c.flagSet.bindings.lookup(name, occurs)
}
func (c *commandContext) set() BindingLookup {
	return c.flagSet
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
	if hasOnlyShortName(f) && impliesValueFlagOnly(f.Value) {
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
	if _, ok := c.LookupData(searchingAlternateCommandKey); ok {
		return nil, commandMissing(sub)
	}

	c.SetData(searchingAlternateCommandKey, true)
	defer c.SetData(searchingAlternateCommandKey, nil)
	if interceptErr == nil {
		if auto, ok := c.LookupData(commandNotFoundKey); ok {
			interceptErr = auto.(func(*Context, error) (*Command, error))
		}
	}

	if interceptErr != nil {
		res, err := interceptErr(c, commandMissing(sub))
		if res != nil || err != nil {
			return res, err
		}
	}
	return nil, commandMissing(sub)
}

func triggerRobustParsingAndCompletion(c *Context) error {
	if c.robustParsingMode() && c.App() != nil {
		cc := newCompletionData(c)
		comp := cc.ShellComplete
		if comp == nil {
			return nil
		}

		args, incomplete := comp.GetCompletionRequest()
		items := c.Complete(args, incomplete)
		c.Stdout.WriteString(comp.FormatCompletions(items))
		return Exit(0)
	}
	return nil
}

var _ target = (*Command)(nil)
var _ customizable = (*Command)(nil)
var _ hookable = (*Command)(nil)
var _ internalCommandContext = (*commandContext)(nil)
