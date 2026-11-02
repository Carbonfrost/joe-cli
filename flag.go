// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Carbonfrost/joe-cli/internal/privatekey"
	"github.com/Carbonfrost/joe-cli/internal/support"
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

// Flag represents a command line flag.  The only required attribute that must be set is Name.
// By default, the type of a flag is string; however, to use a more specific type you must
// either specify a pointer to a variable or use the built-in constants that identify the type to
// use:
//
//	&Flag{
//	  Name: "age",
//	  Value: &age, // var age int -- defined somewhere in scope
//	}
//
//	&Flag{
//	  Name: "age",
//	  Value: cli.Int(), // also sets int
//	}
//
// The corresponding, typed method to access the value of the flag by name is available from the Context.
// In this case, you can  obtain value of the --age=21 flag using Context.Int("flag"), which may be
// necessary when you don't use your own variable.
//
// By default, if a flag name starts with an underscore, it
// is hidden.  To stop this, either set Visible option explicitly or disable
// global behavior with the DisableAutoVisibility option.
type Flag struct {
	targetSupport
	hooksSupport

	// Name provides the name of the flag. This value must be set, and it is used to access
	// the flag's value via the context
	Name string

	// Aliases provides a list of alternative names for the flag.  In general, Name should
	// be used for the long name of the flag, and Aliases should contain the short name.
	// If there are additional names for compatibility reasons, they should be included
	// with Aliases but listed after the preferred names. Note that only one short name
	// and one long name is displayed on help screens by default.
	Aliases []string

	// HelpText contains text which briefly describes the usage of the flag.  If it contains
	// placeholders in the form {PLACEHOLDER}, then these name the purpose of the flag's
	// value.  If a flag has multiple values, then placeholders can also specify the index of
	// the corresponding value using the syntax {0:PLACEHOLDER}; otherwise, the order is
	// inferred start to end.
	// For style, generally the usage text should be limited to about 40 characters.
	// Sentence case is recommended for the usage text.    Uppercase is recommended for the
	// text of placeholders.  The placeholder is used in the synopsis for the flag as well
	// as error messages.
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the flag.  If left blank, a succinct synopsis
	// is generated from the type of the flag's value
	UsageText string

	// EnvVars specifies the name of environment variables that are read to provide the
	// default value of the flag.
	EnvVars []string

	// FilePath specifies a file that is loaded to provide the default value of the flag.
	FilePath string

	// Value provides the value of the flag.  Any of the following types are valid for the
	// value:
	//
	//   * *bool
	//   * *time.Duration
	//   * *float32
	//   * *float64
	//   * *int
	//   * *int16
	//   * *int32
	//   * *int64
	//   * *int8
	//   * *net.IP
	//   * *[]string
	//   * *map[string]string
	//   * **regexp.Regexp
	//   * *string
	//   * *uint
	//   * *uint16
	//   * *uint32
	//   * *uint64
	//   * *uint8
	//   * **url.URL
	//   * an implementation of Value interface
	//
	// If unspecified, the value will be a string pointer.
	// For more information about Values, see the Value type
	Value any

	// DefaultText provides a description of the default value for the flag.  This is displayed
	// on help screens but is otherwise unused
	DefaultText string

	// Options sets various options about how to treat the flag.  For example, options can
	// hide the flag or make its value optional.
	Options Option

	// Category specifies the flag category.  When categories are used, flags are grouped
	// together on the help screen
	Category string

	// Description provides a long description for the flag.  The long description is
	// not used in any templates by default.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description any

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]any

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before any

	// After executes after the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	After any

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses any

	// Action executes if the flag was set.  Refer to cli.Action about the correct
	// function signature to use.
	Action any

	// Completion specifies a callback function that determines the auto-complete results
	Completion Completion

	// Transform defines how to interpret the text passed to the *Flag.  This is generally used
	// when specialized syntax preprocesses the text, such as file references.  Refer to the
	// overview in cli.Transform for information.
	Transform TransformFunc

	optionalValue     any    // set when blank and optional
	optionalValueText string // when empty, use the placeholder of the value

	// persistentIn restricts the commands where the flag applies.  It is nil
	// unless PersistentIn was used.
	persistentIn ContextFilter

	bs BindingState
}

// ValueHelp describes one of the values which is possible for a flag.  The values
// of a flag are aggregated into a table using the ValueHelpText and ValueManualText
// actions, and the table is rendered on the default help screen indented beneath the flag.
type ValueHelp struct {
	// Value provides one of the values which the flag can be set to.  Any type can be
	// used; the value is formatted using fmt.Sprint when it is displayed.
	Value any

	// HelpText contains text which briefly describes the value.  For style, generally
	// the help text should be limited to about 40 characters and use sentence case.
	HelpText string

	// ManualText provides the text shown in the manual for the value.  The default
	// templates don't use this value
	ManualText string
}

type flagsByCategory []*flagCategory

type flagCategory struct {
	Category string
	Flags    []*Flag
}

type option interface {
	BindingState
	target
	hookable
	Set(any) error

	reset()
	actualArgCounter() ArgCounter
	transformFunc() TransformFunc
	contextName() string
	value() any
	name() string
	envVars() []string
	filePath() string
	setTransform(fn TransformFunc)
}

type wrapOccurrenceContext struct {
	*optionContext
	index int
	val   any
}

func groupFlagsByCategory(flags []*Flag) flagsByCategory {
	res := flagsByCategory{}
	all := map[string]*flagCategory{}
	category := func(name string) *flagCategory {
		if c, ok := all[name]; ok {
			return c
		}
		c := &flagCategory{Category: name, Flags: []*Flag{}}
		all[name] = c
		res = append(res, c)
		return c
	}
	for _, f := range flags {
		cc := category(f.Category)
		cc.Flags = append(cc.Flags, f)
	}
	slices.SortFunc(res, func(a, b *flagCategory) int {
		return cmp.Compare(a.Category, b.Category)
	})
	return res
}

// Synopsis contains the name of the flag, its aliases, and the value placeholder.  The text of synopsis
// is inferred from the HelpText.  Up to one short and one long name will be used.
func (f *Flag) Synopsis() string {
	return sprintSynopsis(f.synopsis())
}

func (f *Flag) cacheSynopsis(syn *synopsis.Flag) *synopsis.Flag {
	f.SetData(synopsisKey, syn)
	return syn
}

func (f *Flag) synopsis() *synopsis.Flag {
	if f.Data != nil {
		if a, ok := f.LookupData(synopsisKey); ok {
			return a.(*synopsis.Flag)
		}
	}
	return f.cacheSynopsis(f.newSynopsis())
}

func (f *Flag) newSynopsis() *synopsis.Flag {
	syn := synopsis.NewFlag(f.Name, f.Aliases, f.HelpText, f.UsageText, f.value(), getGroup(f))
	syn.Style = synopsis.StyleFromData(f.Data)
	syn.Category = synopsis.CategoryFromData(f.Data)
	if isPseudoFlag(f) && syn.Value.Placeholder != "" {
		// The value of a selector can't be attached to the flag
		syn.Separator = " "
	}
	if f.internalFlags().optional() {
		syn.WithOptionalValue(f.optionalValueText)
	}
	return syn
}

// SetData sets internal data used by the flag
func (f *Flag) SetData(key any, value any) {
	f.privateData(&f.Data).Set(key, value)
}

// LookupData obtains internal data used by the flag
func (f *Flag) LookupData(key any) (any, bool) {
	return f.privateData(&f.Data).Lookup(key)
}

func (f *Flag) setDescription(value any) {
	f.Description = value
}

func (f *Flag) setHelpText(name string) {
	f.HelpText = name
}

func (f *Flag) setUsageText(s string) {
	f.UsageText = s
}

func (f *Flag) setManualText(name string) {
	f.ManualText = name
}

func (f *Flag) setCategory(name string) {
	f.Category = name
}

func (f *Flag) setDefaultText(name string) {
	f.DefaultText = name
}

func (f *Flag) setCompletion(c Completion) {
	f.Completion = c
}

func (f *Flag) data() map[string]any {
	return f.Data
}

func (f *Flag) description() any {
	return f.Description
}

func (f *Flag) helpText() string {
	return f.HelpText
}

func (f *Flag) usageText() string {
	return f.UsageText
}

func (f *Flag) manualText() string {
	return f.ManualText
}

func (f *Flag) setOptionalValue(v any) {
	f.setInternalFlags(internalFlagOptional, true)
	f.optionalValue = v
	f.optionalValueText = valueOptionalText(v) // Print the value (e.g. --secure[=TLS1.2])
}

func (f *Flag) setImpliedOptionalValue(v any) {
	f.setOptionalValue(v)
	f.optionalValueText = "" // Don't print smart optional values; the type or user-specified label is more useful (e.g --level=[NUMBER])
}

func (f *Flag) pipeline(t Timing) any {
	switch t {
	case AfterTiming:
		return f.After
	case BeforeTiming:
		return f.Before
	case InitialTiming:
		return f.Uses
	default:
		return f.Action
	}
}

func (f *Flag) options() *Option {
	return &f.Options
}

func (f *Flag) contextName() string {
	if len(f.Name) == 1 {
		return fmt.Sprintf("-%s", f.Name)
	}
	return fmt.Sprintf("--%s", f.Name)
}

// appliesIn detects whether the flag can be used within the command context c.
// Flags apply everywhere they are inherited unless PersistentIn narrowed them.
func (f *Flag) appliesIn(c *Context) bool {
	return f.persistentIn == nil || f.persistentIn.Matches(c)
}

func filterInApplicableFlags(c *Context, flags []*Flag) []*Flag {
	return slices.DeleteFunc(
		slices.Clone(flags),
		func(f *Flag) bool {
			return !f.appliesIn(c)
		},
	)
}

func (f *Flag) setTransform(fn TransformFunc) {
	f.Transform = fn
}

func (f *Flag) transformFunc() TransformFunc {
	return f.Transform
}

func (f *Flag) completion() Completion {
	if f.Completion != nil {
		return f.Completion
	}
	if v, ok := f.Value.(valueCompleter); ok {
		return v.Completion()
	}

	return nil
}

func (f *Flag) reset() {
	f.bs = nil
	optionReset(f.Value, f.internalFlags())
}

func (f Flag) actualArgCounter() ArgCounter {
	if f.flags.flagOnly() {
		return NoArgs()
	}
	switch value := f.Value.(type) {
	case *[]string:
		if f.flags.optional() {
			break
		}
		return ArgCount(1)
	case valueProvidesCounter:
		return value.NewCounter()
	}
	return &defaultCounter{requireSeen: !f.flags.optional()}
}

func (c *wrapOccurrenceContext) Bindings(name string) [][]string {
	return c.parentLookup.Bindings(name)
}

func (c *wrapOccurrenceContext) BindingNames() []string {
	return c.parentLookup.BindingNames()
}

func (c *wrapOccurrenceContext) rawOccurrences() [][]string {
	return c.parentLookup.Bindings(c.option.name())
}

func (c *wrapOccurrenceContext) numOccurs() int {
	return len(c.rawOccurrences())
}

func (c *wrapOccurrenceContext) Raw(name string) []string {
	if name == "" {
		v := c.rawOccurrences()[c.index]
		return v
	}

	return c.optionContext.Raw(name)
}

func (c *wrapOccurrenceContext) RawOccurrences(name string) []string {
	if name == "" {
		v := c.rawOccurrences()[c.index]
		return v[1:]
	}

	return c.optionContext.RawOccurrences(name)
}

func (c *wrapOccurrenceContext) lookupValue(name string) (any, bool) {
	if name == "" {
		return c.val, true
	}
	return c.optionContext.lookupValue(name)
}

// Seen returns true if the flag was used on the command line at least once
func (f *Flag) Seen() bool {
	return f.requireBindingState().Seen()
}

// OccurrenceValue obtains the value at the given occurrence. Refer to BindingLookup
// for more detail.
func (f *Flag) OccurrenceValue(index int) any {
	return f.requireBindingState().OccurrenceValue(index)
}

// OccurrenceValues obtains the values which occurred for the given name in the
// binding result.  Refer to BindingLookup for more detail.
func (f *Flag) OccurrenceValues() []any {
	return f.requireBindingState().OccurrenceValues()
}

// Occurrences returns the number of times the flag was specified on the command line
func (f *Flag) Occurrences() int {
	return f.requireBindingState().Occurrences()
}

// ShortName gets the short name for the flag including the leading dash.  This is
// the Name of the flag if it contains exactly one character, or this is the
// first alias which contains exactly one character.  This is the empty string
// if the name and all aliases are long names.  The result starts with a dash.
func (f *Flag) ShortName() string {
	if len(f.Name) == 1 {
		return "-" + f.Name
	}
	for _, nom := range f.Aliases {
		if len(nom) == 1 {
			return "-" + nom
		}
	}
	return ""
}

// LongName gets the long name for the flag including the leading dashes.
// This is the Name of the flag if it contains more than one character, or this
// is the first alias which contains more than one character.  If the Name and
// all Aliases have exactly one character, then the value of Name is returned even
// if it has exactly one character.  Note that even in this case, the result starts
// with two leading dashes.
func (f *Flag) LongName() string {
	if len(f.Name) > 1 {
		return "--" + f.Name
	}
	for _, nom := range f.Aliases {
		if len(nom) > 1 {
			return "--" + nom
		}
	}
	return "--" + f.Name
}

// Names obtains the name of the flag and its aliases, including their leading
// dashes.
func (f *Flag) Names() []string {
	res := []string{
		optionName(f.Name),
	}
	for _, a := range f.Aliases {
		res = append(res, optionName(a))
	}
	return res
}

// Set will update the value of the flag.  A string is parsed and applied to the value,
// and any other value is set directly.  As a special case, nil resets the value to its
// zero value.  (See Set for more information about how values are set.)
func (f *Flag) Set(v any) error {
	return optionSet(f.Value, f.flags, f.optionalValue, v)
}

func optionSet(value any, flags internalFlags, optionalValue, arg any) error {
	switch a := arg.(type) {
	case nil:
		return setReset(value)

	case string:
		if trySetOptional(value, func() (any, bool) {
			return optionalValue, (a == "" && flags.optional())
		}) {
			return nil
		}

		return setCore(value, flags.disableSplitting(), a)
	}

	return setDirect(value, arg)
}

// SetOccurrence will update the value of the flag
func (f *Flag) SetOccurrence(values ...string) error {
	return f.requireBindingState().SetOccurrence(values...)
}

// SetOccurrenceData will update the value of the flag
func (f *Flag) SetOccurrenceData(v any) error {
	return f.requireBindingState().SetOccurrenceData(v)
}

func (f *Flag) requireBindingState() BindingState {
	if f.bs == nil {
		f.bs = &directBindingState{
			value:         f.Value,
			optionalValue: f.optionalValue,
			flags:         f.internalFlags(),
		}
		if f.internalFlags().eachOccurrence() {
			f.bs = f.bs.(*directBindingState).upgrade()
		}
	}
	return f.bs
}

func (f *Flag) name() string {
	return f.Name
}

func (f *Flag) envVars() []string {
	return f.EnvVars
}

func (f *Flag) filePath() string {
	return f.FilePath
}

func (f *Flag) value() any {
	var created bool
	f.Value, created = ensureDestination(f.Value, false)
	if created {
		f.setInternalFlags(internalFlagDestinationImplicitlyCreated, true)
	}
	f.setInternalFlags(isFlagType(f.Value), true)
	return f.Value
}

func (f *Flag) category() string {
	return f.Category
}

func (f *Flag) defaultText() string {
	return f.DefaultText
}

// VisibleFlags filters all flags in the flag category by whether they are not hidden
func (f *flagCategory) VisibleFlags() []*Flag {
	return filterInVisibleFlags(f.Flags)
}

// Undocumented determines whether the category is undocumented (i.e. has no HelpText set
// on any of its flags)
func (f *flagCategory) Undocumented() bool {
	for _, x := range f.Flags {
		if x.HelpText != "" {
			return false
		}
	}
	return true
}

func hasOnlyShortName(f *Flag) bool {
	return len(f.Name) == 1
}

func impliesValueFlagOnly(p any) bool {
	switch val := p.(type) {
	case *bool:
		return true
	case interface{ IsBoolFlag() bool }:
		return val.IsBoolFlag()
	}
	return false
}

func ensureDestination(dest any, multi bool) (newValue any, created bool) {
	if dest == nil {
		if !multi {
			return String(), true
		}
		return List(), true
	}
	return dest, false
}

func findFlagByName(items []*Flag, v any) (*Flag, int, bool) {
	switch name := v.(type) {
	case *Flag:
		for index, match := range items {
			if name == match {
				return name, index, true
			}
		}
		return nil, -1, false
	case string:
		trimmed := strings.TrimLeft(name, "-")
		for index, sub := range items {
			if sub.Name == trimmed {
				return sub, index, true
			}
			if slices.Contains(sub.Aliases, trimmed) {
				return sub, index, true
			}
		}
	default:
		panic(fmt.Sprintf("unexpected type: %T", v))
	}
	return nil, -1, false
}

func isFlagType(p any) internalFlags {
	if impliesValueFlagOnly(p) {
		return internalFlagFlagOnly
	}
	return 0
}

func flagsByNameOrder(x, y *Flag) int {
	return cmp.Compare(x.Name, y.Name)
}

// pseudoCategory is the help screen category for flags that name a pseudo command
const pseudoCategory = "Operations"

// PseudoCommand makes a flag which invokes a command, which is called a pseudo command.
// Rather than being named by an argument, as sub-commands are, the pseudo command is
// specified by a flag.  This is the way that tools such as dns-sd work, where
// each operation is selected by a flag and each operation has its own flags and args:
//
//	&cli.Flag{
//	    Name:    "browse",
//	    Aliases: []string{"B"},
//	    Uses:    cli.PseudoCommand(&cli.Command{
//	        Args:   cli.Args("type", new(string), "domain", new(string)),
//	        Action: browse,
//	    }),
//	}
//
// With this definition, app -B _http._tcp local runs the command with the arguments
// that follow the flag.  The flag takes the rest of the command line, so it is expected
// to be last among the flags of its command, and any flags which follow it are for
// the pseudo command.  The pseudo command also inherits the persistent flags of its
// command, so app -B -v _http._tcp works when -v is persistent.  When the flag is used, the
// Action of the command that defines it does not run as is the case with ordinary sub-commands.
//
// PseudoCommand is instended to be used in the Uses pipeline of a flag, and it defines a prototype
// which can apply useful defaults including making the flag non-persistent and ordered last, which
// is required for it to work properly, and including listing it under the category "Operations" on
// the help screen and expanding its alternative synopsis (see Command.Synopses).
//
// The PseudoCommand can be used in one of two modes, which depends on how many commands the flag has once
// its initialization is complete: a singleton command or several commands. With a singleton command, the name,
// aliases, and help text of the command are copied to the flag by default. With several commands, the flag
// selects among them by naming one in its value, app -g browse _http._tcp local. The usage lists
// each command as an alternative synopsis, and the help screen describes them with the values of the flag.
// Either way, the defaults only apply when the flag does not set them itself.
//
// The commands can also be provided from within the same pipeline.  The AddCommand, AddCommands, and RemoveCommand
// actions apply to the pseudo commands of the flag rather than to the sub-commands of its command,
// provided that they appear after PseudoCommand in the pipeline.
//
// It is an internal error if the flag has no commands once it has been initialized, or if the
// names or aliases of the commands of a selector are not unique.  This function panics if any
// of the commands is nil.
//
// Note that the syntax of flags have quirks when used as pseudo commands.  A value cannot be
// attached to the flag (e.g. `--browse=x`) for a singleton pseudo command, but when multiple are available
// then the syntax only works for commands that take no arguments (`--global=browse`). In general,
// the form with spaces instead of equal signs is always preferred.
// The --help flag only works when it appears before the pseudo command.
func PseudoCommand(cmds ...*Command) Action {
	if slices.Contains(cmds, nil) {
		panic("cli: PseudoCommand requires non-nil commands")
	}

	// Each flag gets its own state because commands can be added to or removed from
	// it, and the action can be used by more than one flag
	return ActionFunc(func(c *Context) error {
		return Do(c, pseudoCommandPipeline(&pseudoOperations{commands: slices.Clone(cmds)}))
	})
}

func pseudoCommandPipeline(ops *pseudoOperations) Action {
	// The name, aliases, and help text of the flag depend on how many commands there are,
	// which isn't known until its pipeline is complete, so they're defaulted afterwards
	proto := &Prototype{
		Category: pseudoCategory,
		Options:  NonPersistent | OrderLast,
		Value:    &pseudoValue{ops: ops},
	}

	return Pipeline(
		proto,
		Data(privatekey.PseudoCommand, ops),
		ActionFunc(func(c *Context) error {
			f := c.Flag()

			// The pseudo commands aren't sub-commands, so they are initialized as though they
			// were after the flag and its peers have all been initialized
			return c.WithCommand(ActionFunc(func(parent *Context) error {
				if len(ops.commands) == 0 {
					return parent.internalError(errPseudoNoCommands)
				}

				for _, cmd := range ops.commands {
					cmd.SetData(privatekey.PseudoFlag, f)
				}

				_, err := initializeChildren(parent, func() []*Command { return ops.commands }, map[*Command]bool{})
				if err != nil {
					return err
				}

				if ops.selector() {
					// Names can be provided late by the commands, so they're checked after
					// they've been initialized
					if err := ops.checkNames(); err != nil {
						return parent.internalError(err)
					}
				}

				originalName := f.contextName()
				ops.applyDefaults(f)
				if f.contextName() != originalName {
					// Hooks that match the flag by name need to run again
					child := parent.newChild(f, InitialTiming)
					err = child.reinitialize()
					child.state.close()
					if err != nil {
						return err
					}
				}

				if ops.selector() {
					for _, cmd := range ops.commands {
						c.SetValueHelpText(cmd.Name, cmd.HelpText)
					}
				}
				return nil
			}))
		}),
		At(ActionTiming, ActionFunc(executePseudoCommand)),
	)
}

type pseudoOperations struct {
	commands []*Command
}

func (o *pseudoOperations) selector() bool {
	return len(o.commands) > 1
}

func (o *pseudoOperations) applyDefaults(f *Flag) {
	name, aliases, helpText := "global", []string{"g"}, "Run an operation"
	if !o.selector() {
		cmd := o.commands[0]
		name, aliases, helpText = cmd.Name, cmd.Aliases, cmd.HelpText
	}

	if f.Name == "" {
		f.Name = name

		// Only set up aliases if the name was set
		if len(f.Aliases) == 0 {
			f.Aliases = slices.Clone(aliases)
		}
	}
	if f.HelpText == "" {
		f.HelpText = helpText
	}
	if !o.selector() && o.commands[0].Name == "" {
		o.commands[0].Name = f.Name
	}
}

func (o *pseudoOperations) checkNames() error {
	names := map[string]bool{}
	var errs []error
	for i, cmd := range o.commands {
		if cmd.Name == "" {
			errs = append(errs, fmt.Errorf("command at index #%d must have a name", i))
			continue
		}
		errs = append(errs, support.ValidateNames(names, cmd.Name, cmd.Aliases, nil)...)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors initializing pseudo commands: %w", errors.Join(errs...))
	}
	return nil
}

func (o *pseudoOperations) resolve(tokens []string) (*Command, []string, error) {
	if !o.selector() {
		return o.commands[0], tokens, nil
	}
	if len(tokens) == 0 {
		return nil, nil, expectedArgument(1)
	}

	cmd, _, ok := findCommandByName(o.commands, tokens[0])
	if !ok {
		return nil, nil, commandMissing(tokens[0])
	}
	return cmd, tokens[1:], nil
}

type pseudoValue struct {
	ops  *pseudoOperations
	args []string
}

// Set adds an argument for the pseudo command.  The parser passes a blank value when the flag is
// followed by nothing at all, so a blank value is ignored unless it follows other arguments.
func (v *pseudoValue) Set(s string) error {
	if s == "" && len(v.args) == 0 {
		return nil
	}
	v.args = append(v.args, s)
	return nil
}

func (v *pseudoValue) String() string {
	return strings.Join(v.args, " ")
}

func (v *pseudoValue) Get() any {
	return slices.Clone(v.args)
}

func (v *pseudoValue) Reset() {
	v.args = nil
}

func (v *pseudoValue) NewCounter() ArgCounter {
	return ArgCount(TakeRemaining)
}

// Synopsis is blank for a flag that has a singleton pseudo command because the flag doesn't have
// a value as far as users are concerned.  For a selector, it lists the names of the commands.
func (v *pseudoValue) Synopsis() string {
	if v.ops == nil || !v.ops.selector() {
		return ""
	}

	names := make([]string, len(v.ops.commands))
	for i, cmd := range v.ops.commands {
		names[i] = cmd.Name
	}
	return synopsis.Choices(names)
}

func pseudoOf(t target) (*pseudoOperations, bool) {
	if f, ok := t.(*Flag); ok && f.Data == nil {
		return nil, false
	}
	data, ok := t.LookupData(privatekey.PseudoCommand)
	if !ok {
		return nil, false
	}
	ops, ok := data.(*pseudoOperations)
	return ops, ok
}

func pseudoOfContext(c *Context) (*pseudoOperations, bool) {
	if f, ok := c.target().(*Flag); ok {
		return pseudoOf(f)
	}
	return nil, false
}

func pseudoOfCommand(cmd *Command) (*pseudoOperations, bool) {
	data, ok := cmd.LookupData(privatekey.PseudoFlag)
	if !ok {
		return nil, false
	}
	f, ok := data.(*Flag)
	if !ok {
		return nil, false
	}
	return pseudoOf(f)
}

func isPseudoFlag(f *Flag) bool {
	_, ok := pseudoOf(f)
	return ok
}

func executePseudoCommand(c *Context) error {
	ops, ok := pseudoOf(c.target())
	if !ok {
		return nil
	}

	rest, _ := c.Value("").([]string)
	cmd, args, err := ops.resolve(rest)
	if err != nil {
		return err
	}

	c.Parent().target().setInternalFlags(internalFlagDidSubcommandExecute, true)
	return c.Parent().newChild(cmd, ActionTiming).Execute(append([]string{cmd.Name}, args...))
}

func completePseudoCommand(c *Context, cmd *Command, cc *CompletionRequest) ([]CompletionItem, bool) {
	for _, f := range cmd.Flags {
		ops, ok := pseudoOf(f)
		if !ok || len(cc.Bindings.Bindings(f.Name)) == 0 {
			continue
		}

		tokens := cc.Bindings.RawOccurrences(f.Name)
		if len(tokens) == 1 && tokens[0] == "" {
			// The parser passes a blank when the flag has nothing after it
			tokens = nil
		}

		if ops.selector() && len(tokens) == 0 {
			return completePseudoCommandNames(ops, cc), true
		}

		pseudo, args, err := ops.resolve(tokens)
		if err != nil {
			return nil, true
		}
		invoke := append([]string{pseudo.Name}, args...)
		return c.newChild(pseudo, ActionTiming).Complete(invoke, cc.Incomplete), true
	}
	return nil, false
}

func completePseudoCommandNames(ops *pseudoOperations, cc *CompletionRequest) []CompletionItem {
	var items []CompletionItem
	for _, cmd := range ops.commands {
		for _, name := range cmd.Names() {
			if strings.HasPrefix(name, cc.Incomplete) {
				items = append(items, CompletionItem{Value: name, HelpText: cmd.HelpText})
			}
		}
	}
	return items
}

var _ Value = (*pseudoValue)(nil)
var _ target = (*Flag)(nil)
