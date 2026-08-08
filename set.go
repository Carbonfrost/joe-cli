// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"strconv"
	"strings"
)

// BindingResult contains the occurrences of the values passed to each flag and arg
// along with the input args which produced them.
type BindingResult struct {
	args     []string
	bindings map[string][][]string
	names    []string
}

type set struct {
	Lookup
	Binding
	*BindingResult
}

type bindingImpl struct {
	shortOptions      map[string]string
	longOptions       map[string]string
	positionalOptions []string
	names             map[string]option
	numeric           string
}

type argBinding struct {
	names  []string
	takers []ArgCounter
	index  int
}

type argList []string

type argCountError int
type parserState int

// RawParseFlag enumerates rules for RawParse
type RawParseFlag int

// Binding provides the representation of how a flag set is bound to values.  It defines the
// names and aliases of flags, order of args, and how values passed to an arg or flag are counted.
type Binding interface {
	LookupOption(name string) (TransformFunc, ArgCounter, BindingState, bool)
	ResolveAlias(name string) (string, bool)
	PositionalArgNames() []string
	BehaviorFlags(name string) (optional bool)
	Numeric() string // gets the flag which handles numeric syntax
	Reset()
}

// BindingState defines the state of the binding operation.  This is generally *Arg or *Flag
type BindingState interface {
	SetOccurrenceData(v any) error
	SetOccurrence(values ...string) error
	Occurrences() int
	OccurrenceValue(int) any
	OccurrenceValues() []any
	Seen() bool
}

// directBindingState shares the value that was set on the Arg or Flag and
// simply updates it in place
type directBindingState struct {
	value         any
	optionalValue any
	flags         internalFlags
	count         int
}

func (d *directBindingState) SetOccurrence(values ...string) error {
	d.nextOccur()
	return optionSetMany(d.value, d.flags, d.optionalValue, values...)
}

func optionSetMany(value any, flags internalFlags, optionalValue any, values ...string) error {
	for _, arg := range values {
		err := optionSet(value, flags, optionalValue, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *directBindingState) SetOccurrenceData(v any) error {
	d.nextOccur()
	return SetData(d.value, v)
}

func (d *directBindingState) Seen() bool {
	return d.count > 0
}

func (d *directBindingState) Occurrences() int {
	return d.count
}

func (d *directBindingState) OccurrenceValue(_ int) any {
	return dereference(d.value)
}

func (d *directBindingState) OccurrenceValues() []any {
	return []any{d.OccurrenceValue(-1)}
}

func (d *directBindingState) nextOccur() {
	d.count++
	optionApplyValueConventions(d.value, d.flags, d.count == 1)
}

func (d *directBindingState) upgrade() *multiBindingState {
	return &multiBindingState{initial: d}
}

// multiBindingState shares the value that was set on the Arg or Flag and
// simply updates it in place
type multiBindingState struct {
	initial *directBindingState
	all     []*directBindingState
	actual  any // original initial value - should be flag.Value which needs the final occurrence value
}

func (m *multiBindingState) SetOccurrence(values ...string) error {
	err := m.nextOccur().SetOccurrence(values...)
	if err != nil {
		return err
	}
	if m.actual != nil && len(values) > 0 {
		d := m.initial
		return optionSetMany(m.actual, d.flags, d.optionalValue, values...)
	}
	return nil
}

func (m *multiBindingState) SetOccurrenceData(v any) error {
	return m.nextOccur().SetOccurrenceData(v)
}

func (m *multiBindingState) Seen() bool {
	return m.Occurrences() > 0
}

func (m *multiBindingState) Occurrences() int {
	return len(m.all)
}

func (m *multiBindingState) OccurrenceValue(index int) any {
	return dereference(m.all[index].value)
}

func (m *multiBindingState) OccurrenceValues() []any {
	all := make([]any, len(m.all))
	for i, v := range m.all {
		all[i] = dereference(v.value)
	}
	return all
}

func (m *multiBindingState) nextOccur() *directBindingState {
	// Obtain either the zero value or Reset() the value
	resetOnFirstOccur := !m.initial.flags.merge()

	// Create a copy of the value on each occurrence (unless merge semantics
	// are in place)
	if m.Occurrences() == 0 {
		m.actual = m.initial.value
		m.initial.value = valueCloneZero(m.initial.value)

	} else if resetOnFirstOccur {
		m.initial.value = valueCloneZero(m.initial.value)
	}

	clone := *m.initial
	optionReset(clone.value, m.initial.flags)
	m.all = append(m.all, &clone)
	return m.all[len(m.all)-1]
}

// Raw flags used by the internal parser
const (
	RawRTL = RawParseFlag(1 << iota)
	RawDisallowFlagsAfterArgs
	RawSkipProgramName
	RawParseUnknownFlagsAsArgs
	RawSkipFlagParsing
)

const (
	argCannotUseFlag = argCountError(iota) // arg looks like a flag and cannot be used
	argExpectedMore                        // more arguments were expected

	_argStartSoftErrors // start of soft errors -- these are errors that cause the next
	// argument to be parsed in the binding

	// EndOfArguments signals no more arguments taken by this arg counter.
	EndOfArguments
)

const (
	flagsOrArgs = parserState(iota)
	didPushbackArg
	argsOnly
)

func newSet(b Binding) *set {
	result := &set{
		Binding:       b,
		BindingResult: newBindingResult(nil),
	}
	result.Lookup = LookupFunc(result.lookupValue)
	return result
}

// NewBinding creates a binding. This expert API is to provide a representation
// of the naming and parsing rules for flags, args, and the parent
func NewBinding(flags []*Flag, args []*Arg, parent interface{ Flags() []*Flag }) Binding {
	b := &bindingImpl{
		names:        map[string]option{},
		shortOptions: map[string]string{},
		longOptions:  map[string]string{},

		positionalOptions: []string{},
	}
	for _, f := range flags {
		b.defineFlag(f)
	}

	if parent != nil {
		for _, f := range parent.Flags() {
			if f.internalFlags().nonPersistent() {
				continue
			}
			b.defineFlag(f)
			f.setInternalFlags(internalFlagPersistent, true)
		}
	}

	for _, a := range args {
		b.defineArg(a)
	}
	return b
}

func newArgBinding(bind Binding) *argBinding {
	args := bind.PositionalArgNames()
	takers := make([]ArgCounter, len(args))
	names := make([]string, len(args))
	for i, a := range args {
		names[i] = a
		if _, c, _, ok := bind.LookupOption(a); ok {
			takers[i] = c
		} else {
			takers[i] = ArgCount(TakeUntilNextFlag)
		}
	}

	return &argBinding{
		names:  names,
		takers: takers,
		index:  0,
	}
}

func newArgBindingSingle(name string, taker ArgCounter) *argBinding {
	return &argBinding{
		names:  []string{name},
		takers: []ArgCounter{taker},
		index:  0,
	}
}

func (f RawParseFlag) rightToLeft() bool {
	return f&RawRTL == RawRTL
}

func (f RawParseFlag) disallowFlagsAfterArgs() bool {
	return f&RawDisallowFlagsAfterArgs == RawDisallowFlagsAfterArgs
}

func (f RawParseFlag) skipProgramName() bool {
	return f&RawSkipProgramName == RawSkipProgramName
}

func (f RawParseFlag) parseUnknownFlagsAsArgs() bool {
	return f&RawParseUnknownFlagsAsArgs == RawParseUnknownFlagsAsArgs
}

func (f RawParseFlag) skipFlagParsing() bool {
	return f&RawSkipFlagParsing == RawSkipFlagParsing
}

// RawParse does low-level parsing that will parse from the given input arguments.   (This is for
// advanced use.) The bindings parameter determines how to resolve flags and args.  The return value
// is the binding result, which contains the raw occurrences indexed by the same names.  An error,
// if it occurs is ParseError, which can provide more information about why the
// parse did not complete.
func RawParse(arguments []string, b Binding, flags RawParseFlag) (bindings *BindingResult, err error) {
	args := argList(arguments)
	bindings = newBindingResult(arguments)
	positionalOpts := newArgBinding(b)

	disallowFlagsAfterArgs := flags.disallowFlagsAfterArgs()
	parseUnknownFlagsAsArgs := flags.parseUnknownFlagsAsArgs()

	// When in RTL mode, identify the first argument to actually fill by
	// counting the number of arguments required by arguments right-to-left.
	count := len(arguments)
	if flags.rightToLeft() {
		pos := positionalOpts.takers
		skip := len(pos)

		for i := len(pos) - 1; i >= 0 && i < len(pos); i-- {
			current := pos[i]
			switch counter := current.(type) {
			case *varArgsCounter:
				count--
			case *discreteCounter:
				count -= counter.count
			case *defaultCounter:
				count--
			default:
				count--
			}
			skip--
			if skip == 0 || count <= 1 {
				break
			}
		}

		for range skip {
			if err = positionalOpts.Done(); err != nil {
				return
			}
			positionalOpts.next()
		}
	}

	var (
		state        = flagsOrArgs
		anyArgs      bool
		appendOutput = bindings.appendOutput
	)

	// Skip program name
	if !args.empty() && flags.skipProgramName() {
		args.pop()
	}
	if flags.skipFlagParsing() {
		state = argsOnly
	}

Parsing:
	for !args.empty() {
		arg := args.pop()

		// end of options?
		if arg == "" || arg == "-" || arg[0] != '-' || state == argsOnly || state == didPushbackArg {
			for {
				if arg == "--" {
					state = argsOnly
					if err = positionalOpts.Done(); err != nil {
						return
					}
					positionalOpts.next()
					continue Parsing
				}

				// Note that when pushed back, possibleFlag must be false
				err = positionalOpts.SetArg(arg, args, state == flagsOrArgs)
				if err != nil {
					// Not accepted as an argument, possibly a flag per usual out of
					// order
					if len(arg) > 0 && arg[0] == '-' && state == flagsOrArgs {
						break
					}
					if isHardArgCountErr(err) {
						return
					}
				}
				appendOutput(positionalOpts.name(), []string{positionalOpts.argName(), arg})

				if state == didPushbackArg {
					state = flagsOrArgs
				}

				if args.empty() {
					if err = positionalOpts.Done(); err != nil {
						err = argTakerError(positionalOpts.argName(), "", err, nil)
						return
					}
					continue Parsing
				}

				arg = args.pop()
				anyArgs = true
			}
		}

		// explicitly request end of options?
		if arg == "--" {
			state = argsOnly
		}

		if state == argsOnly {
			continue Parsing
		}

		// Long option processing
		if arg[1] == '-' {
			var value string
			var hasValue bool
			arg, value, hasValue = strings.Cut(arg, "=")

			// Lookup the flag uname given the possible alias name.
			// If the flag name is only one character, then also check
			// whether it can be handled as a short flag (--f=false)
			flag, ok := b.ResolveAlias(arg[2:])

			if !ok {
				if parseUnknownFlagsAsArgs {
					state = didPushbackArg
					args.pushBack(arg)
					continue Parsing
				}
				err = unknownOption(arg[2:], prepend(optionName(arg[2:]), args...))
				return
			}

			if disallowFlagsAfterArgs && anyArgs {
				err = flagAfterArgError(arg[2:])
				return
			}

			// If we require an option and did not have an =
			// then use the next argument as an option.
			_, opt, _, _ := b.LookupOption(flag)
			if !hasValue {
				var outputs []string
				oldArgs := prepend(optionName(arg[2:]), args...)
				if outputs, err = args.take(flag, opt); err != nil {
					err = argTakerError(optionName(arg[2:]), "", err, oldArgs)
					return
				}
				appendOutput(flag, prepend(optionName(arg[2:]), outputs...))

				continue Parsing
			}

			appendOutput(flag, []string{arg, value})
			continue Parsing
		}

		// Short option processing
		arg = arg[1:] // strip -
		if disallowFlagsAfterArgs && anyArgs {
			err = flagAfterArgError(arg)
			return
		}

		for i, c := range arg {
			short := "-" + string(c)
			flag, ok := b.ResolveAlias(string(c))

			var looksNumeric bool
			if !ok && i == 0 && b.Numeric() != "" {
				if _, err := strconv.Atoi(arg); err == nil {
					flag = b.Numeric()
					looksNumeric = true
					ok = true
				}
			}

			if !ok {
				if i == 0 && parseUnknownFlagsAsArgs {
					state = didPushbackArg
					args.pushBack("-" + arg)
					continue Parsing
				}
				err = unknownOption(c, prepend("-"+arg[i:], args...))
				return
			}

			_, opt, _, _ := b.LookupOption(flag)
			value := arg[1+i:]

			// For numeric flags, treat the whole expression as the value
			if looksNumeric {
				value = arg
			}

			if value != "" {
				err = instanceTake(value, false, opt)
				if err == nil {
					appendOutput(flag, []string{short, value})
					continue Parsing
				}

				// If an equal sign is present, this is the syntax -s=value,
				// which implies trying to set a value.  Include the = in
				// the binding
				if value[0] == '=' {
					oldArgs := prepend(short+value, args...)
					err = flagUnexpectedArgument(short, value, oldArgs)
					return
				}

				if err == EndOfArguments {
					// Should be flag-only
					appendOutput(flag, []string{short, ""})
					continue
				}

				return
			}

			if optional := b.BehaviorFlags(flag); optional {
				appendOutput(flag, []string{short, ""})
				err = opt.Done()
				if err != nil {
					return
				}
				continue
			}

			var outputs []string
			oldArgs := prepend(short, args...)
			if outputs, err = args.take(flag, opt); err != nil {
				err = argTakerError(optionName(flag), value, err, oldArgs)
				return
			}

			appendOutput(flag, prepend(short, outputs...))
		}
	}

	err = positionalOpts.Done()
	return
}

func (s *set) parse(args argList, flags RawParseFlag) error {
	bindings, err := RawParse(args, s.Binding, flags)
	s.BindingResult = bindings
	if err != nil {
		return err
	}
	return s.BindingResult.ApplyTo(s)
}

// Raw obtains the values which were specified for a flag or arg.  The empty
// name designates the set itself, whose raw value is the input args.
func (s *set) Raw(name string) []string {
	if name == "" {
		return s.Args()
	}
	return s.BindingResult.Raw(name)
}

// RawOccurrences obtains the values which were specified for a flag or arg,
// excluding the name that was used.  The empty name designates the set itself,
// whose occurrences are the input args excluding the name of the command.
func (s *set) RawOccurrences(name string) []string {
	if name == "" {
		return trimFirstArg(s.Args())
	}
	return s.BindingResult.RawOccurrences(name)
}

func (b *bindingImpl) Reset() {
	for _, a := range b.names {
		a.reset()
	}
}

func (b *bindingImpl) LookupOption(name string) (TransformFunc, ArgCounter, BindingState, bool) {
	a, ok := b.names[name]
	if ok {
		return a.transformFunc(), a.actualArgCounter(), a, true
	}
	return nil, nil, nil, false
}

func (b *bindingImpl) ResolveAlias(name string) (string, bool) {
	if _, ok := b.names[name]; ok {
		return name, true
	}
	if len(name) == 1 {
		if r, ok := b.shortOptions[name]; ok {
			return r, ok
		}
		return "", false
	}
	if r, ok := b.longOptions[name]; ok {
		return r, ok
	}
	return "", false
}

func (b *bindingImpl) BehaviorFlags(name string) (optional bool) {
	if o, ok := b.names[name]; ok {
		return o.internalFlags().optional()
	}
	return false
}

func (b *bindingImpl) Numeric() string {
	return b.numeric
}

func (b *bindingImpl) PositionalArgNames() []string {
	return b.positionalOptions
}

func (b *bindingImpl) defineFlag(f *Flag) {
	name := f.Name
	if len(name) == 0 {
		return
	}

	for _, alias := range f.Aliases {
		if len(alias) == 1 {
			b.shortOptions[alias] = name
		} else {
			b.longOptions[alias] = name
		}
	}
	if f.internalFlags().numeric() {
		b.numeric = f.Name
	}
	b.names[name] = f
}

func (b *bindingImpl) defineArg(a *Arg) {
	b.names[a.Name] = a
	b.positionalOptions = append(b.positionalOptions, a.Name)
}

func (s *set) lookupValue(name string) (any, bool) {
	if s.Binding == nil {
		return nil, false
	}
	if _, _, g, ok := s.Binding.LookupOption(name); ok {
		return g.(option).value(), true
	}
	return nil, false
}

func (s *set) OccurrenceValue(name string, index int) any {
	if _, _, g, ok := s.Binding.LookupOption(name); ok {
		return g.(option).OccurrenceValue(index)
	}
	return nil
}

func (s *set) OccurrenceValues(name string) []any {
	if _, _, g, ok := s.Binding.LookupOption(name); ok {
		return g.(option).OccurrenceValues()
	}
	return nil
}

func newBindingResult(args []string) *BindingResult {
	return &BindingResult{
		args:     args,
		bindings: map[string][][]string{},
	}
}

// MergeBindingResults combines binding results into a single result.  Occurrences
// are appended in the order of the results given, and names are ordered by when
// they first occurred.  The input args of the combined result are taken from the
// first result which provides them.  This is for advanced use.
func MergeBindingResults(results ...*BindingResult) *BindingResult {
	merged := newBindingResult(nil)
	for _, m := range results {
		if m == nil {
			continue
		}
		if merged.args == nil {
			merged.args = m.args
		}
		for _, name := range m.names {
			for _, occur := range m.bindings[name] {
				merged.appendOutput(name, occur)
			}
		}
	}
	return merged
}

func (m *BindingResult) appendOutput(name string, args []string) {
	if e, ok := m.bindings[name]; ok {
		m.bindings[name] = append(e, args)
		return
	}
	m.bindings[name] = [][]string{args}
	m.names = append(m.names, name)
}

func (m *BindingResult) lookup(name string, occurs bool) []string {
	res := make([]string, 0)
	if m == nil {
		return res
	}
	var index int
	if occurs {
		index = 1
	}
	for _, v := range m.bindings[name] {
		res = append(res, v[index:]...)
	}
	return res
}

// Args obtains the input args which were parsed to produce the result.  For the
// args of a command, this includes the name of the command itself.
func (m *BindingResult) Args() []string {
	if m == nil {
		return []string{}
	}
	return append([]string{}, m.args...)
}

// Raw obtains values which were specified for a flag or arg
// including the flag or arg name
func (m *BindingResult) Raw(name string) []string {
	return m.lookup(name, false)
}

// RawOccurrences obtains values which were specified for a flag or arg
// but not including the flag or arg name
func (m *BindingResult) RawOccurrences(name string) []string {
	return m.lookup(name, true)
}

// ApplyTo uses the given binding to apply the values in the
// result
func (m *BindingResult) ApplyTo(b Binding) error {
	if m == nil {
		return nil
	}
	for _, name := range m.names {
		transform, _, value, ok := b.LookupOption(name)
		if !ok {
			continue
		}
		err := rawApplyToOption(m.bindings[name], transform, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// Bindings obtains values which were specified for a flag or arg
// including the flag or arg name and grouped into occurrences.
func (m *BindingResult) Bindings(name string) [][]string {
	if m == nil {
		return nil
	}
	return m.bindings[name]
}

// BindingNames obtains the names of the flags/args which are available,
// in the order in which they first occurred.
func (m *BindingResult) BindingNames() []string {
	if m == nil {
		return []string{}
	}
	return append([]string{}, m.names...)
}

func trimFirstArg(args []string) []string {
	if len(args) == 0 {
		return args
	}
	return args[1:]
}

func rawApplyToOption(v [][]string, transform TransformFunc, value BindingState) error {
	if transform != nil {
		for _, values := range v {
			d, err := transform(values)
			if err != nil {
				return err
			}
			if err := value.SetOccurrenceData(d); err != nil {
				return err
			}
		}
		return nil
	}

	for _, values := range v {
		err := value.SetOccurrence(values[1:]...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *argList) next() bool {
	r := *a
	if len(r) == 0 {
		return false
	}
	*a = r[1:]
	return true
}

func (a *argList) pop() string {
	res := a.current()
	a.next()
	return res
}

func (a *argList) pushBack(s string) {
	r := *a
	*a = prepend(s, r...)
}

func (a argList) empty() bool {
	return len(a) == 0
}

func (a argList) current() string {
	return a[0]
}

func (a *argList) take(name string, opt ArgCounter) (output []string, err error) {
	bind := newArgBindingSingle(name, opt)
	output = make([]string, 0)
	const possibleFlag = true
	for !a.empty() {
		arg := a.current()
		err = bind.SetArg(arg, *a, possibleFlag)
		if err != nil {
			// Not accepted as an argument, possibly a flag per usual out of
			// order
			if len(arg) > 0 && arg[0] == '-' && possibleFlag {
				break
			}

			// HACK This should not unwrap the Error type, which is a bit fragile
			if e, ok := err.(*ParseError); ok {
				if e.Code == UnexpectedArgument {
					break
				}
			}

			if isHardArgCountErr(err) {
				return
			}
			break
		}
		output = append(output, arg)
		_ = a.pop()
	}

	if len(output) == 0 {
		output = []string{""}
	}

	err = bind.Done()
	return
}

func (a *argBinding) next() bool {
	a.index++
	return a.index < len(a.takers)
}

func (a *argBinding) name() string {
	return a.names[a.index]
}

func (a *argBinding) argName() string {
	return "<" + a.name() + ">"
}

func (a *argBinding) current() ArgCounter {
	if a.index < len(a.takers) {
		return a.takers[a.index]
	}
	return nil
}

func (a *argBinding) Done() error {
	if c := a.current(); c != nil {
		return c.Done()
	}
	return nil
}

func (a *argBinding) SetArg(arg string, remainder []string, possibleFlag bool) error {
	for {
		t := a.current()
		if t == nil {
			return unexpectedArgument(arg, prepend(arg, remainder...))
		}

		err := t.Take(arg, possibleFlag)
		if err == nil {
			return nil
		}

		if isHardArgCountErr(err) {
			return err
		}

		a.next()
	}
}

func instanceTake(arg string, possibleFlag bool, t ArgCounter) error {
	err := t.Take(arg, possibleFlag)
	if err != nil {
		return err
	}
	return t.Done()
}

func (e argCountError) Error() string {
	switch e {
	case argCannotUseFlag:
		return "cannot use; looks like a flag"
	case EndOfArguments:
		return "no more arguments to take"
	case argExpectedMore:
		return "more arguments expected"
	case _argStartSoftErrors: // to please exhaustive
	}
	panic("unreachable!")
}

// isHardArgCountErr represents errors that must be returned to the outer
// parser loop so that it can either fail the parse or try parsing a flag
func isHardArgCountErr(e error) bool {
	if f, ok := e.(argCountError); ok {
		return f < _argStartSoftErrors
	}
	return true
}

func allowFlag(arg string, possibleFlag bool) bool {
	if arg == "-" {
		// Solitary dash is treated as if an argument
		return false
	}
	return len(arg) > 0 && (possibleFlag && arg[0] == '-')
}

func prepend(arg string, args ...string) []string {
	return append([]string{arg}, args...)
}

var _ lookupCore = (*set)(nil)
var _ BindingLookup = (*set)(nil)
var _ Binding = (*set)(nil)
