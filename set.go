package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// BindingMap contains the occurrences of the values passed to each flag and arg.
type BindingMap map[string][][]string

type set struct {
	*lookupSupport
	shortOptions      map[rune]*internalOption
	longOptions       map[string]*internalOption
	positionalOptions []*internalOption
	names             map[string]*internalOption
	bindings          BindingMap
}

type bindingImpl struct {
	shortOptions      map[string]*internalOption
	longOptions       map[string]*internalOption
	positionalOptions []*internalOption
	names             map[string]*internalOption
}

type argBinding struct {
	names  []string
	takers []ArgCounter
	index  int
}

type argList []string

type internalOption struct {
	short         []rune
	long          []string
	value         *generic
	count         int
	uname         string
	narg          interface{}
	optionalValue interface{} // set when blank and optional
	flags         internalFlags
	transform     transformFunc
}

type argCountError int
type parserState int

// RawParseFlag enumerates rules for RawParse
type RawParseFlag int

// Binding provides the representation of how a flag set is bound to values.  It defines the
// names and aliases of flags, order of args, and how values passed to an arg or flag are counted.
type Binding interface {
	Lookup(name string) (ArgCounter, bool)
	FlagName(name string) (string, bool)
	Args() []string
	IsOptionalValue(name string) bool
}

const (
	RawRTL = RawParseFlag(1 << iota)
	RawDisallowFlagsAfterArgs
	RawSkipProgramName
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
	argsOnly
)

func newSet() *set {
	return &set{
		names:             map[string]*internalOption{},
		shortOptions:      map[rune]*internalOption{},
		longOptions:       map[string]*internalOption{},
		bindings:          BindingMap{},
		positionalOptions: []*internalOption{},
	}
}

func newArgBinding(bind Binding) *argBinding {
	args := bind.Args()
	takers := make([]ArgCounter, len(args))
	names := make([]string, len(args))
	for i, a := range args {
		names[i] = a
		if c, ok := bind.Lookup(a); ok {
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

// RawParse does low-level parsing that will parse from the given input arguments.   (This is for
// advanced use.) The bindings parameter determines how to resolve flags and args.  The return values
// are a map of data corresponding to the raw occurrences using the same names.  An error,
// if it occurs is ParseError, which can provide more information about why the
// parse did not complete.
func RawParse(arguments []string, b Binding, flags RawParseFlag) (bindings BindingMap, err error) {
	args := argList(arguments)
	bindings = BindingMap{}
	positionalOpts := newArgBinding(b)

	if args.empty() {
		err = positionalOpts.Done()
		return
	}

	pos := positionalOpts.takers
	disallowFlagsAfterArgs := flags.disallowFlagsAfterArgs()

	// When in RTL mode, identify the first argument to actually fill by
	// counting the number of arguments required by arguments right-to-left.
	count := len(arguments)
	if flags.rightToLeft() {
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

		for i := 0; i < skip; i++ {
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
	if flags.skipProgramName() {
		args.pop()
	}

Parsing:
	for !args.empty() {
		arg := args.pop()

		// end of options?
		if arg == "" || arg == "-" || arg[0] != '-' || state == argsOnly {
			for {
				if arg == "--" {
					state = argsOnly
					if err = positionalOpts.Done(); err != nil {
						return
					}
					positionalOpts.next()
					continue Parsing
				}

				err = positionalOpts.SetArg(arg, state == flagsOrArgs)
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
				argName := "<" + positionalOpts.name() + ">"
				appendOutput(positionalOpts.name(), []string{argName, arg})

				if args.empty() {
					if err = positionalOpts.Done(); err != nil {
						err = argTakerError(argName, "", err, nil)
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
			e := strings.IndexRune(arg, '=')
			var value string
			if e > 0 {
				value = arg[e+1:]
				arg = arg[:e]
			}

			// Lookup the flag uname given the possible alias name.
			// If the flag name is only one character, then also check
			// whether it can be handled as a short flag (--f=false)
			flag, ok := b.FlagName(arg[2:])

			if !ok {
				err = unknownOption(arg[2:], append([]string{optionName(arg[2:])}, args...))
				return
			}

			if disallowFlagsAfterArgs && anyArgs {
				err = flagAfterArgError(arg[2:])
				return
			}

			// If we require an option and did not have an =
			// then use the next argument as an option.
			opt, _ := b.Lookup(flag)
			if e < 0 {
				var outputs []string
				var oldArgs = append([]string{optionName(arg[2:])}, args...)
				if outputs, err = args.take(flag, opt); err != nil {
					err = argTakerError(optionName(arg[2:]), "", err, oldArgs)
					return
				}
				appendOutput(flag, append([]string{optionName(arg[2:])}, outputs...))

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
			flag, ok := b.FlagName(string(c))
			if !ok {
				err = unknownOption(c, append([]string{"-" + arg[i:]}, args...))
				return
			}

			opt, _ := b.Lookup(flag)
			value := arg[1+i:]

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
					if err != nil {
						// The flag previously checked for value but didn't
						// support them
						var oldArgs = append([]string{short + value}, args...)
						err = flagUnexpectedArgument(short, value, oldArgs)
						return
					}
					appendOutput(flag, []string{short, value})
					continue Parsing
				}

				if err == EndOfArguments {
					// Should be flag-only
					appendOutput(flag, []string{short, ""})
					continue
				}

				return
			}

			if b.IsOptionalValue(flag) {
				appendOutput(flag, []string{short, ""})
				err = opt.Done()
				if err != nil {
					return
				}
				continue
			}

			var outputs []string
			var oldArgs = append([]string{short}, args...)
			if outputs, err = args.take(flag, opt); err != nil {
				err = argTakerError(optionName(flag), value, err, oldArgs)
				return
			}

			appendOutput(flag, append([]string{short}, outputs...))
		}
	}

	err = positionalOpts.Done()
	return
}

func (s *set) parse(args argList, flags RawParseFlag) error {
	err := s.parseBindings(args, flags)
	if err != nil {
		return err
	}
	return applyBindings(s.bindings, s.names)
}

func applyBindings(bindings BindingMap, names map[string]*internalOption) error {
	for k, v := range bindings {
		opt := names[k]

		if opt.transform != nil {
			for occurrence, values := range v {
				opt.startOccurrence(occurrence + 1)
				d, err := opt.transform(values)
				if err != nil {
					return err
				}
				if err := opt.setViaTransformOutput(d); err != nil {
					return err
				}
			}

			continue
		}

		for occurrence, values := range v {
			opt.startOccurrence(occurrence + 1)
			for _, value := range values[1:] {
				err := opt.Set(value)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *set) parseBindings(args argList, flags RawParseFlag) error {
	bindings, err := RawParse(args, &bindingImpl{
		shortOptions:      setShortOptions(s.shortOptions),
		longOptions:       s.longOptions,
		positionalOptions: s.positionalOptions,
		names:             s.names,
	}, flags)
	s.bindings = bindings
	return err
}

func setShortOptions(m map[rune]*internalOption) map[string]*internalOption {
	res := map[string]*internalOption{}
	for k, v := range m {
		res[string(k)] = v
	}
	return res
}

func (b *bindingImpl) Lookup(name string) (ArgCounter, bool) {
	a, ok := b.names[name]
	if ok {
		return a.actualArgCounter(), true
	}
	return nil, false
}

func (b *bindingImpl) FlagName(name string) (string, bool) {
	if _, ok := b.names[name]; ok {
		return name, true
	}
	if len(name) == 1 {
		if r, ok := b.shortOptions[name]; ok {
			return r.uname, ok
		}
		return "", false
	}
	if r, ok := b.longOptions[name]; ok {
		return r.uname, ok
	}
	return "", false
}

func (b *bindingImpl) IsOptionalValue(name string) bool {
	if o, ok := b.names[name]; ok {
		return o.flags.optional()
	}
	return false
}

func (b *bindingImpl) Args() []string {
	args := make([]string, len(b.positionalOptions))
	for i, o := range b.positionalOptions {
		args[i] = o.uname
	}
	return args
}

func (s *set) defineFlag(res *internalOption) {
	if len(res.short) == 0 && len(res.long) == 0 {
		panic("invalid flag definition, missing name or alias")
	}

	for _, short := range res.short {
		s.shortOptions[short] = res
	}
	for _, long := range res.long {
		s.longOptions[long] = res
	}

	s.names[res.uname] = res
}

func (s *set) defineArg(res *internalOption) {
	if res.uname == "" {
		res.uname = fmt.Sprintf("_%d", len(s.positionalOptions)+1)
	}

	s.names[res.uname] = res
	s.positionalOptions = append(s.positionalOptions, res)
}

func (s *set) withArgs(args []*Arg) *set {
	for _, a := range args {
		a.applyToSet(s)
	}
	return s
}

func (s *set) lookupValue(name string) (interface{}, bool) {
	if g, ok := s.names[name]; ok {
		return g.value.p, true
	}
	return nil, false
}

func (m BindingMap) appendOutput(name string, args []string) {
	if e, ok := m[name]; ok {
		m[name] = append(e, args)
	} else {
		m[name] = [][]string{args}
	}
}

func (m BindingMap) lookup(name string, occurs bool) []string {
	res := make([]string, 0)
	var index int
	if occurs {
		index = 1
	}
	for _, v := range m[name] {
		res = append(res, v[index:]...)
	}
	return res
}

func (m BindingMap) Raw(name string) []string {
	return m.lookup(name, false)
}

func (m BindingMap) RawOcurrences(name string) []string {
	return m.lookup(name, true)
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

func (a argList) len() int {
	return len(a)
}

func (a argList) empty() bool {
	return a.len() == 0
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
		err = bind.SetArg(arg, possibleFlag)
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

func (a *argBinding) SetArg(arg string, possibleFlag bool) error {
	for {
		t := a.current()
		if t == nil {
			return unexpectedArgument(arg, []string{arg})
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

func (o *internalOption) Seen() bool {
	return o.count > 0
}

func (o *internalOption) Set(arg string) error {
	return o.value.Set(arg, o)
}

func (o *internalOption) setViaTransformOutput(v interface{}) error {
	if s, ok := o.value.p.(valueSetData); ok {
		switch val := v.(type) {
		case string:
			return s.SetData(strings.NewReader(val))
		case io.Reader:
			return s.SetData(val)
		case []byte:
			return s.SetData(bytes.NewReader(val))
		}
	}

	if s, ok := o.value.p.(*[]byte); ok {
		switch val := v.(type) {
		case io.Reader:
			buf := bytes.NewBuffer(*s)
			if _, err := io.Copy(buf, val); err != nil {
				return err
			}
			*s = buf.Bytes()
			return nil

		case []byte:
			buf := bytes.NewBuffer(*s)
			buf.Write(val)
			*s = buf.Bytes()
			return nil
		}
	}

	switch val := v.(type) {
	case string:
		return o.Set(val)
	case io.Reader:
		bb, err := io.ReadAll(val)
		if err != nil {
			return err
		}
		return o.Set(string(bb))
	case []byte:
		return o.Set(string(val))
	}

	panic(fmt.Sprintf("unexpected transform output %T", v))
}

func (o *internalOption) startOccurrence(n int) {
	o.count = n
	o.value.applyValueConventions(o.flags, n)
}

func (o *internalOption) Name() string {
	if len(o.short) > 0 {
		return optionName(o.short[0])
	}
	return optionName(o.long[0])
}

func (o *internalOption) Value() *generic {
	return o.value
}

func (o *internalOption) Occurrences() int {
	return o.count
}

func (o *internalOption) actualArgCounter() ArgCounter {
	if o.flags.flagOnly() {
		return NoArgs()
	}
	if o.narg == nil {
		if o.value != nil {
			switch value := o.value.p.(type) {
			case *[]string:
				return ArgCount(TakeUntilNextFlag)
			case valueProvidesCounter:
				return value.NewCounter()
			}
		}

		return &defaultCounter{
			requireSeen: o.isFlag() && !o.flags.optional(),
		}
	}
	return ArgCount(o.narg)
}

func (o *internalOption) isFlag() bool {
	// HACK Detecting whether a flag
	return len(o.short)+len(o.long) > 0
}

var _ lookupCore = (*set)(nil)
