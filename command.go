// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/Carbonfrost/joe-cli/internal/privatekey"
	"github.com/Carbonfrost/joe-cli/internal/support"
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

// Command represents a command with arguments, flags, and expressions
//
// By default, if a command name starts with an underscore, it
// is hidden.  To stop this, either set Visible option explicitly or disable
// global behavior with the DisableAutoVisibility option.
//
// Within a command, all args and flags must have unique names. The behavior is
// not defined if a flag alias is duplicative, but no flag can have the same name
// as one of the aliases. If a flag name is blank, this is an error; however, if
// an arg name is blank, it is implicitly given a name using its index.
type Command struct {
	targetSupport
	hooksSupport

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
	Action any

	// Before executes before the app action or any sub-command action runs.  Refer to
	// cli.Action about the correct function signature to use.
	Before any

	// After executes after the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	After any

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	// Actions within the Uses and Before pipelines can modify the app Commands and Flags lists.  Any
	// commands or flags added to the list will be initialized
	Uses any

	// Category places the command into a category.  Categories are displayed on the default
	// help screen.
	Category string

	// Description provides a long description for the command.  The long description is
	// displayed on the help screen.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description any

	// Comment provides a short descriptive comment.  This is
	// usually a few words to summarize the purpose of the command.
	Comment string

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]any

	// Options sets common options for use with the command
	Options Option

	// HelpText describes the help text displayed for commands
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the command.  If left blank, a succinct synopsis
	// is generated that lists each visible flag and arg
	UsageText string

	// Completion provides the completion for use in the command.  By default, the
	// completion detects whether a flag or arg is being used and then delegates to
	// the completion present there
	Completion Completion

	fromApp *App
	ifRoot  *rootCommandData

	// flagOrder contains Flags in the order in which they are executed and hooked,
	// which differs from Flags only when DependsOn, OrderFirst, or OrderLast is used.
	// It is nil when the definition order applies.
	flagOrder []*Flag
}

type rootCommandData struct {
	templateFuncs map[string]any
	templates     *template.Template
}

type robustParseResult struct {
	bindings *BindingResult
	err      error
}

type commandCategory struct {
	Category string
	Commands []*Command
}

type commandsByCategory []*commandCategory

type commandContext struct {
	*set
}

// ExecuteSubcommand finds and executes a sub-command.  This action is intended to be used
// as the action on an argument.  The argument should be a list of strings, which represent
// the sub-command to locate and execute and the arguments to use.  If used within the
// Uses pipeline of an argument, a prototype applies these requirements for you and other
// good defaults to support completion and synopsis.  If no sub-command matches, an error
// is generated, which you can intercept with custom handling using interceptErr.  The interceptErr function
// should return a command to execute in lieu of returning the error.  If the interceptErr
// command is nil, it is interpreted as the command not existing and the app will exit with a generic "command
// not found error" message.  If it returns an error, then executing the sub-command fails with the error.
// However, if ErrSkipCommand is returned, then no command is executed, and no error is generated.
// It is uncommon to use this action because this action is implicitly bound to a synthetic argument when a
// command defines any sub-commands.
func ExecuteSubcommand(interceptErr func(context.Context, error) (*Command, error)) Action {
	return Pipeline(&Prototype{
		Name:       "command",
		UsageText:  "<command> [<args>]",
		Value:      List(),
		NArg:       -1,
		Options:    DisableSplitting,
		Completion: CompletionFunc(completeSubCommand),
	}, At(ActionTiming, ActionFunc(func(c *Context) error {
		invoke := c.List("")
		return subcommandCore(c, invoke, interceptErr)
	})))
}

func subcommandCore(c *Context, invoke []string, interceptErr func(context.Context, error) (*Command, error)) error {
	if len(invoke) == 0 {
		return nil
	}
	cmd, err := tryFindCommandOrIntercept(c, invoke[0], interceptErr)
	if err == ErrSkipCommand {
		return nil
	}
	if err != nil {
		return err
	}
	c.Parent().target().setInternalFlags(internalFlagDidSubcommandExecute, true)
	newCtx := c.Parent().newChild(cmd, ActionTiming)
	return newCtx.Execute(invoke)
}

// CommandNotFoundHandler is invoked when a command cannot be found.  It is passed the context of
// the parent attempting to invoke a command and the error previously encountered, and it returns
// the command if any that can substitute.  It implements Action: executing it registers the
// handler so that it is consulted when a command is not found.  Composition occurs with handlers
// already registered (see ComposeCommandNotFoundHandler); they each get called until one returns
// a command.
type CommandNotFoundHandler func(context.Context, error) (*Command, error)

// Execute registers the handler to be consulted when a command cannot be found.  A nil handler
// resets the behavior to the default.
func (h CommandNotFoundHandler) Execute(ctx context.Context) error {
	c := FromContext(ctx)
	cmd := c.Command()
	if h == nil {
		// Use a sentinel value, which is used to indicate the default behavior should be used
		c.SetData(privatekey.CommandNotFound, false)
		return nil
	}

	fn := h
	if existing, ok := cmd.Data[privatekey.CommandNotFound]; ok {
		if existingFn, ok := existing.(CommandNotFoundHandler); ok {
			// Compose with the previously registered handler
			fn = ComposeCommandNotFoundHandler(h, existingFn)
		}
	}
	c.SetData(privatekey.CommandNotFound, fn)
	return nil
}

// ComposeCommandNotFoundHandler combines handlers into a single handler.  Each handler is invoked
// in turn until one returns a command (without an error), in which case that command is used.
// Otherwise, the error from a handler is passed along to the next, and the result of the last
// handler is returned.
func ComposeCommandNotFoundHandler(handlers ...CommandNotFoundHandler) CommandNotFoundHandler {
	return func(c context.Context, err error) (*Command, error) {
		for i, h := range handlers {
			if h == nil {
				continue
			}
			cmd, hErr := h(c, err)
			if (cmd != nil && hErr == nil) || i == len(handlers)-1 {
				return cmd, hErr
			}
			err = hErr
		}
		return nil, err
	}
}

// HandleCommandNotFound assigns a default handler to invoke when a command cannot be found.
// Composition occurs with handlers registered to handle commands not found.  They each get
// called until one returns a command.
func HandleCommandNotFound(fn func(*Context, error) (*Command, error)) CommandNotFoundHandler {
	if fn == nil {
		return nil
	}
	return func(c context.Context, err error) (*Command, error) {
		return fn(FromContext(c), err)
	}
}

// ImplicitCommand indicates the command which is implicit when no sub-command matches.
// The main use case for this is to allow a command to be invoked by default without being
// named.  For example, you might have a sub-command called "exec" which can be omitted, making
// the following invocations equivalent:
//
//   - cloud exec tail -f /var/output/log
//   - cloud tail -f /var/output/log
//
// If the command named by the ImplicitCommand also does not exist, the original error
// is returned.
func ImplicitCommand(name string) CommandNotFoundHandler {
	return HandleCommandNotFound(func(c *Context, originalErr error) (*Command, error) {
		invoke := append([]string{name}, c.Args()...)
		err := subcommandCore(c, invoke, nil)
		if pe, ok := err.(*ParseError); ok && pe.Code == CommandNotFound {
			// The name is itself unknown, return the original error
			return nil, originalErr
		}
		if err != nil {
			return nil, err
		}

		return nil, ErrSkipCommand
	})
}

// SuggestCommand provides a CommandNotFoundHandler which suggests sub-commands
// that are similar to the one that could not be found.  When suggestions are found, they are
// recorded on the ParseError via its Suggestions attribute and rendered using
// the "Suggestions" template.  SuggestCommand is added to the default pipeline
// for the root command; the DisableSuggestions option opts out of it.
func SuggestCommand() CommandNotFoundHandler {
	return HandleCommandNotFound(func(c *Context, err error) (*Command, error) {
		if c.flagSetOrAncestor((internalFlags).disableSuggestions) {
			return nil, err
		}

		pe, ok := err.(*ParseError)
		if !ok {
			return nil, err
		}

		suggestions := suggestCommandNames(pe.Name, commandSuggestionNames(c.Command()))
		if len(suggestions) == 0 {
			return nil, err
		}

		pe.Suggestions = suggestions
		if tpl := c.Template("Suggestions"); tpl != nil {
			var buf strings.Builder
			if tpl.Execute(&buf, struct{ Suggestions []string }{suggestions}) == nil {
				pe.detail = strings.TrimRight(buf.String(), "\n")
			}
		}
		return nil, pe
	})
}

// commandSuggestionNames obtains the names and aliases of the visible sub-commands
// of the command, which are the candidates considered when suggesting a command.
func commandSuggestionNames(cmd *Command) []string {
	if cmd == nil {
		return nil
	}
	var names []string
	for _, sub := range cmd.VisibleSubcommands() {
		names = append(names, sub.Names()...)
	}
	return names
}

// suggestCommandNames returns the candidates which are similar to name, ordered
// from most to least similar (ties broken alphabetically).
func suggestCommandNames(name string, candidates []string) []string {
	if name == "" {
		return nil
	}

	threshold := len(name)/2 + 1
	type scored struct {
		name string
		dist int
	}

	var matches []scored
	for _, candidate := range candidates {
		dist := levenshtein(name, candidate)
		if dist <= threshold {
			matches = append(matches, scored{candidate, dist})
		}
	}

	slices.SortFunc(matches, func(a, b scored) int {
		return cmp.Or(cmp.Compare(a.dist, b.dist), cmp.Compare(a.name, b.name))
	})

	res := make([]string, len(matches))
	for i, m := range matches {
		res[i] = m.name
	}
	return res
}

// levenshtein computes the Levenshtein edit distance between two strings.
func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
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
	slices.SortFunc(res, func(a, b *commandCategory) int {
		return cmp.Compare(a.Category, b.Category)
	})
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

// Synopsis returns the UsageText for the command or produces a succinct representation
// that names each flag and arg
func (c *Command) Synopsis() string {
	return sprintSynopsis(c.newSynopsis())
}

// Command tries to obtain a sub-command by name or alias
func (c *Command) Command(name string) (*Command, bool) {
	r, _, ok := findCommandByName(c.Subcommands, name)
	return r, ok
}

// Flag tries to obtain a flag by name or alias
func (c *Command) Flag(name string) (*Flag, bool) {
	r, _, ok := findFlagByName(c.Flags, name)
	return r, ok
}

// Arg tries to obtain a arg by name or alias
func (c *Command) Arg(name any) (*Arg, bool) {
	a, _, ok := findArgByName(c.Args, name)
	return a, ok
}

// VisibleArgs filters all arguments in the command by whether they are not hidden
func (c *Command) VisibleArgs() []*Arg {
	return slices.DeleteFunc(slices.Clone(c.Args), func(a *Arg) bool {
		return a.internalFlags().hidden()
	})
}

// VisibleFlags filters all flags in the command by whether they are not hidden
func (c *Command) VisibleFlags() []*Flag {
	return filterInVisibleFlags(c.Flags)
}

func filterInVisibleFlags(flags []*Flag) []*Flag {
	return slices.DeleteFunc(
		slices.Clone(flags),
		func(f *Flag) bool {
			return f.internalFlags().hidden()
		},
	)
}

// VisibleSubcommands filters all sub-commands in the command by whether they are not hidden
func (c *Command) VisibleSubcommands() []*Command {
	return slices.DeleteFunc(slices.Clone(c.Subcommands), func(cmd *Command) bool {
		return cmd.internalFlags().hidden()
	})
}

// Names obtains the name of the command and its aliases
func (c *Command) Names() []string {
	return append([]string{c.Name}, c.Aliases...)
}

func (c *Command) buildSet(ctx *Context) *set {
	binding := NewBinding(c.Flags, c.Args, ctx.Parent())
	set := newSet(binding)
	ctx.state.getInternal().(*commandContext).set = set
	return set
}

func ensureSubcommands(ctx context.Context) error {
	cmd := FromContext(ctx).target().(*Command)

	if len(cmd.Subcommands) > 0 {
		if cmd.Action == nil {
			cmd.Action = DisplayHelpScreen()
		}
		return Do(ctx, AddArg(&Arg{
			Name: "command",
			Uses: Pipeline(
				ExecuteSubcommand(nil),
				tagged,
			),
		}))
	}
	return nil
}

func completeSubCommand(c *Context) []CompletionItem {
	cc := c.CompletionRequest()
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
			for _, alias := range s.Aliases {
				if detect(alias) {
					res = append(res, CompletionItem{Value: alias, HelpText: s.HelpText})
				}
			}
		}
		return res
	}

	cmd, err := tryFindCommandOrIntercept(c, invoke[0], nil)
	if err != nil {
		return nil
	}

	newCtx := c.Parent().newChild(cmd, ActionTiming)
	return newCtx.Complete(invoke, cc.Incomplete)
}

func (c *Command) completion() Completion {
	if c.Completion != nil {
		return c.Completion
	}
	return CompletionFunc(defaultCommandCompletion)
}

func defaultCommandCompletion(c *Context) []CompletionItem {
	cc := c.CompletionRequest()
	cmd := c.Target().(*Command)
	var items []CompletionItem

	if strings.HasPrefix(cc.Incomplete, "-") {
		// If a search only finds one match, then complete the flag
		items = findSolitaryMatch(c)
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
				return actualCompletion(flag.completion()).Complete(c)
			}
			return nil
		}

		arg, ok := cmd.Arg(name)
		if !ok && len(cmd.Args) > 0 {
			arg = cmd.Args[0]
		}
		if arg == nil {
			return nil
		}
		return actualCompletion(arg.completion()).Complete(c)
	}

	// Request completion of the last argument that was seen
	if len(cmd.Args) > 0 {
		var last *Arg
		for _, a := range cmd.Args {
			last = a
			if len(cc.Bindings.Bindings(a.Name)) == 0 {
				break
			}
		}
		return actualCompletion(last.completion()).Complete(c.newChild(last, ActionTiming))
	}

	return items
}

func findSolitaryMatch(c *Context) []CompletionItem {
	cc := c.CompletionRequest()
	cmd := c.Target().(*Command)
	flagName, _, hasArg := strings.Cut(cc.Incomplete, "=")
	var match *Flag
	var matchName string

	for _, f := range cmd.VisibleFlags() {
		for _, n := range f.synopsis().Names {
			if n == cc.Incomplete || (hasArg && strings.HasPrefix(n, flagName)) {
				return actualCompletion(f.completion()).Complete(c.newChild(f, ActionTiming))
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

func (c *Command) newSynopsis() *synopsis.Command {
	flags := make([]*synopsis.Flag, len(c.Flags))
	args := make([]*synopsis.Arg, len(c.Args))
	for i, f := range c.Flags {
		flags[i] = f.synopsis()
	}
	for i, a := range c.Args {
		args[i] = a.newSynopsis()
	}

	syn := synopsis.NewCommand(
		c.Name, flags, args, c.internalFlags().rightToLeft(),
	)
	syn.Style = synopsis.StyleFromData(c.Data)
	syn.CondenseCategories = !c.internalFlags().disableSynopsisCategories()
	return syn
}

// SetData sets internal data used by the command
func (c *Command) SetData(key any, value any) {
	c.privateData(&c.Data).Set(key, value)
}

// LookupData obtains internal data used by the command
func (c *Command) LookupData(key any) (any, bool) {
	return c.privateData(&c.Data).Lookup(key)
}

func (c *Command) setCategory(name string) {
	c.Category = name
}

func (c *Command) setDefaultText(_ string) {
}

func (c *Command) setManualText(name string) {
	c.ManualText = name
}

func (c *Command) setHelpText(name string) {
	c.HelpText = name
}

func (c *Command) setUsageText(s string) {
	c.UsageText = s
}

func (c *Command) setDescription(name any) {
	c.Description = name
}

func (c *Command) setCompletion(cv Completion) {
	c.Completion = cv
}

func (c *Command) description() any {
	return c.Description
}

func (c *Command) helpText() string {
	return c.HelpText
}

func (c *Command) usageText() string {
	return c.UsageText
}

func (c *Command) manualText() string {
	return c.ManualText
}

func (c *Command) category() string {
	return c.Category
}

func (c *Command) defaultText() string {
	return ""
}

func (c *Command) data() map[string]any {
	return c.Data
}

func (c *Command) rootData() *rootCommandData {
	if c.ifRoot == nil {
		c.ifRoot = newRootCommandData()
	}
	return c.ifRoot
}

func (c *Command) options() *Option {
	return &c.Options
}

func (c *Command) pipeline(t Timing) any {
	switch t {
	case AfterTiming:
		return c.After
	case BeforeTiming:
		return c.Before
	case InitialTiming:
		return c.Uses
	default:
		return c.Action
	}
}

func (c *Command) contextName() string {
	return c.Name
}

func initializeFlagsArgs(ctx *Context) error {
	var (
		flagStart int
		argStart  int
		anyFlags  = true
		anyArgs   = true
	)

	// New flags and/or args may have been introduced, so allow these to also initialize.
	// They can ONLY be appended to the slice, not inserted elsewhere
	for anyFlags || anyArgs {
		flags := ctx.LocalFlags()[flagStart:]
		flagStart = len(ctx.LocalFlags())
		if err := initializeChildren(ctx, flags); err != nil {
			return err
		}
		anyFlags = len(flags) > 0

		args := ctx.LocalArgs()[argStart:]
		argStart = len(ctx.LocalArgs())
		if err := initializeChildren(ctx, args); err != nil {
			return err
		}
		anyArgs = len(args) > 0
	}

	return nil
}

func initializeSubcommands(ctx *Context) error {
	cmd := ctx.target().(*Command)
	anySubcommands := true
	var subcommandStart int

	for anySubcommands {
		subs := cmd.Subcommands[subcommandStart:]
		subcommandStart = len(cmd.Subcommands)
		if err := initializeChildren(ctx, subs); err != nil {
			return err
		}
		anySubcommands = len(subs) > 0
	}
	return nil
}

func initializeChildren[T target](ctx *Context, subs []T) error {
	for _, sub := range subs {
		if sub.internalFlags().initialized() {
			continue
		}

		originalName := sub.contextName()
		child := ctx.newChild(sub, InitialTiming)
		err := child.initialize()
		if err != nil {
			return err
		}
		// The name has changed, so hooks need to run again
		// on the flag
		if sub.contextName() != originalName {
			// TODO: These errors might have to propagate
			_ = child.reinitialize()
		}
		child.state.close()
	}
	return nil
}

func copyContextToOrigin(ctx *Context) error {
	ctx.state.close()
	return nil
}

func finalizeArgsAndFlags(c *Context) error {
	// Set up default names for args; generate an error for unnamed flags; generate
	// an error for duplicative names
	names := map[string]bool{}
	var errs []error
	for _, f := range c.PersistentFlags() {
		names[f.Name] = true
	}
	checkPersistent := func(name string) string {
		// Generate an additional context if the name is persistent
		for _, parent := range c.Lineage()[1:] {
			for _, f := range parent.LocalFlags() {
				if f.Name == name {
					return fmt.Sprintf(" (persistent from %s)", parent.Name())
				}
			}
		}
		return ""
	}

	for i, a := range c.LocalArgs() {
		if a.Name == "" {
			a.Name = "_" + strconv.Itoa(1+i)
			continue
		}
		errs = append(errs, support.ValidateNames(names, a.Name, nil, checkPersistent)...)
	}

	for i, f := range c.LocalFlags() {
		if f.Name == "" {
			errs = append(errs, fmt.Errorf("flag at index #%d must have a name", i))
			continue
		}
		errs = append(errs, support.ValidateNames(names, f.Name, f.Aliases, checkPersistent)...)
	}

	for _, f := range c.LocalFlags() {
		promoteOptionalAliases(f, &f.Aliases, names)
	}

	if len(errs) > 0 {
		return c.internalError(fmt.Errorf("errors initializing command: %w", errors.Join(errs...)))
	}
	return nil
}

func orderFlags(c *Context) error {
	if !c.IsCommand() {
		return nil
	}

	cmd := c.Command()
	flags := cmd.Flags
	deps := make([][]int, len(flags))
	var errs []error
	anyOrdering := false

	for i, f := range flags {
		if f.internalFlags().orderClass() != 0 {
			anyOrdering = true
		}

		names := dependsOnNames(f)
		if len(names) == 0 {
			continue
		}
		anyOrdering = true

		for _, name := range names {
			trimmed := strings.TrimLeft(name, "-")
			self := optionName(f.Name)

			if trimmed == "" {
				errs = append(errs, fmt.Errorf("flag %s: a dependency must be named", self))
				continue
			}
			if dep, index, found := findFlagByName(flags, trimmed); found {
				if dep == f {
					errs = append(errs, fmt.Errorf("flag %s can't depend upon itself", self))
					continue
				}
				deps[i] = append(deps[i], index)
				continue
			}
			if _, _, found := findArgByName(cmd.Args, trimmed); found {
				errs = append(errs, fmt.Errorf("flag %s can't depend upon arg %s", self, trimmed))
				continue
			}
			// Naming a persistent flag is redundant rather than an error because
			// ancestor flags always run first anyway
			if _, ok := c.Parent().LookupFlag(trimmed); ok {
				continue
			}
			errs = append(errs, fmt.Errorf("flag %s depends upon %s, which does not exist", self, optionName(trimmed)))
		}
	}

	if len(errs) > 0 {
		return c.internalError(fmt.Errorf("errors initializing command: %w", errors.Join(errs...)))
	}
	if !anyOrdering {
		cmd.flagOrder = nil
		return nil
	}

	order, err := topoSortFlags(flags, deps)
	if err != nil {
		return c.internalError(fmt.Errorf("errors initializing command: %w", err))
	}
	cmd.flagOrder = order
	return nil
}

func dependsOnNames(f *Flag) []string {
	names, _ := f.Data[dependsOnDataKey].([]string)
	return names
}

// topoSortFlags implements Kahn's algorithm, which selects among the flags that are
// available at each step using the sort key implied by OrderFirst, OrderLast, and the
// index of the flag.  This makes the dependencies hard constraints and the absolute
// ordering a tiebreaker.
func topoSortFlags(flags []*Flag, deps [][]int) ([]*Flag, error) {
	remaining := make([]int, len(flags))
	dependents := make([][]int, len(flags))
	for i, dd := range deps {
		for _, j := range dd {
			// Guard against a dependency named more than once
			if slices.Contains(dependents[j], i) {
				continue
			}
			dependents[j] = append(dependents[j], i)
			remaining[i]++
		}
	}

	// Among the available flags, the one with the lowest order class wins, and ties
	// are broken by the order in which the flags were defined
	better := func(x, y int) bool {
		return cmp.Or(
			cmp.Compare(flags[x].internalFlags().orderClass(), flags[y].internalFlags().orderClass()),
			cmp.Compare(x, y),
		) < 0
	}

	available := make([]int, 0, len(flags))
	for i := range flags {
		if remaining[i] == 0 {
			available = append(available, i)
		}
	}

	res := make([]*Flag, 0, len(flags))
	for len(available) > 0 {
		best := 0
		for k, i := range available[1:] {
			if better(i, available[best]) {
				best = k + 1
			}
		}
		next := available[best]
		available = slices.Delete(available, best, best+1)
		res = append(res, flags[next])

		for _, d := range dependents[next] {
			remaining[d]--
			if remaining[d] == 0 {
				available = append(available, d)
			}
		}
	}

	if len(res) < len(flags) {
		var names []string
		for i, f := range flags {
			if remaining[i] > 0 {
				names = append(names, optionName(f.Name))
			}
		}
		return nil, fmt.Errorf("cyclic flag dependency among %s", listOfValues(names, false, "and"))
	}
	return res, nil
}

func finalizeSubcommands(c *Context) error {
	// Set up default names for args; generate an error for unnamed flags; generate
	// an error for duplicative names
	names := map[string]bool{}
	var errs []error
	for _, name := range c.Path() {
		names[name] = true
	}

	for i, c := range c.LocalCommands() {
		if c.Name == "" {
			errs = append(errs, fmt.Errorf("command at index #%d must have a name", i))
			continue
		}
		errs = append(errs, support.ValidateNames(names, c.Name, c.Aliases, nil)...)
	}

	for _, cmd := range c.LocalCommands() {
		promoteOptionalAliases(cmd, &cmd.Aliases, names)
	}

	if len(errs) > 0 {
		return c.internalError(fmt.Errorf("errors initializing command: %w", errors.Join(errs...)))
	}
	return nil
}

func promoteOptionalAliases(data target, aliases *[]string, names map[string]bool) {
	support.PromoteOptionalAliases(data, aliases, names)
}

func getGroup(f *Flag) synopsis.OptionGroup {
	if f.internalFlags().hidden() {
		return synopsis.Hidden
	}
	if f.internalFlags().exits() {
		return synopsis.ActionGroup
	}
	if hasOnlyShortName(f) && impliesValueFlagOnly(f.Value) {
		if f.internalFlags().required() {
			return synopsis.OnlyShortNoValue
		}
		return synopsis.OnlyShortNoValueOptional
	}
	if f.internalFlags().required() {
		return synopsis.Other
	}
	return synopsis.OtherOptional
}

func commandsByNameOrder(x, y *Command) int {
	return cmp.Compare(x.Name, y.Name)
}

func findCommandByName(cmds []*Command, v any) (*Command, int, bool) {
	if cmd, ok := v.(*Command); ok {
		for index, sub := range cmds {
			if cmd == sub {
				return cmd, index, true
			}
		}
		return nil, -1, false
	}

	name, ok := v.(string)
	if !ok {
		return nil, -1, false
	}
	for index, sub := range cmds {
		if sub.Name == name {
			return sub, index, true
		}
		if slices.Contains(sub.Aliases, name) {
			return sub, index, true
		}
	}
	return nil, -1, false
}

func tryFindCommandOrIntercept(c *Context, sub string, interceptErr func(context.Context, error) (*Command, error)) (*Command, error) {
	if res, ok := c.Command().Command(sub); ok {
		return res, nil
	}
	if c.flagSetOrAncestor((internalFlags).searchingAlternateCommand) {
		return nil, commandMissing(sub)
	}

	c.target().setInternalFlags(internalFlagSearchingAlternateCommand, true)
	defer c.target().setInternalFlags(internalFlagSearchingAlternateCommand, false)
	if interceptErr == nil {
		if auto, ok := c.LookupData(privatekey.CommandNotFound); ok {
			// Invalid casts are ignored because a sentinel value can be set  to indicate that
			// the default behavior should be used
			if h, ok := auto.(CommandNotFoundHandler); ok {
				interceptErr = h
			}
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

func triggerRobustParsingAndCompletion(ctx context.Context) error {
	c := FromContext(ctx)
	if c.robustParsingMode() && c.App() != nil {
		cc := newCompletionData(c)
		comp := cc.ShellComplete
		if comp == nil {
			return nil
		}

		args, incomplete := comp.GetCompletionRequest()
		items := c.Complete(args, incomplete)
		c.Print(comp.FormatCompletions(items))
		return Exit(0)
	}
	return nil
}

var _ target = (*Command)(nil)
var _ hookable = (*Command)(nil)
