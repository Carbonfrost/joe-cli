package cli

import (
	"fmt"
	"os"
	"strings"
)

type set struct {
	*lookupSupport
	shortOptions      map[rune]*internalOption
	longOptions       map[string]*internalOption
	positionalOptions []*internalOption
	values            genericValues
	bindings          bindingsMap
	isRTL             bool
}

type argBinding struct {
	items  []*internalOption
	takers []ArgCounter
	index  int
}

type argList []string
type bindingsMap map[string][][]string

type genericValues map[string]*generic

type internalOption struct {
	short         []rune
	long          []string
	value         *generic
	count         int
	uname         string
	narg          interface{}
	optionalValue interface{} // set when blank and optional
	flags         internalFlags
}

type argCountError int
type parserState int

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

func newSet(isRTL bool) *set {
	values := genericValues{}
	return &set{
		values:            values,
		shortOptions:      map[rune]*internalOption{},
		longOptions:       map[string]*internalOption{},
		positionalOptions: []*internalOption{},
		isRTL:             isRTL,
		lookupSupport: &lookupSupport{
			values,
		},
	}
}

func newArgBinding(args []*internalOption) *argBinding {
	items := make([]*internalOption, len(args))
	takers := make([]ArgCounter, len(args))

	for i, x := range args {
		items[i] = x
		takers[i] = x.actualArgCounter()
	}
	return &argBinding{
		items, takers, 0,
	}
}

func (s *set) startArgBinding(count int) (res *argBinding, err error) {
	res = newArgBinding(s.positionalOptions)

	// When in RTL mode, identify the first argument to actually fill by
	// counting the number of arguments required by arguments right-to-left.
	if s.isRTL {
		pos := s.positionalOptions
		skip := len(pos)

		for i := len(pos) - 1; i >= 0 && i < len(pos); i-- {
			current := pos[i]
			switch counter := current.actualArgCounter().(type) {
			case *varArgsCounter:
				count--
			case *discreteCounter:
				count -= counter.count
			case *optionalCounter:
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
			if err = res.Done(); err != nil {
				return
			}
			res.next()
		}
	}
	return
}

func (s *set) parse(args argList, disallowFlagsAfterArgs bool) error {
	if args.empty() {
		return nil
	}

	bind, err := s.startArgBinding(args.len())
	if err != nil {
		return err
	}
	s.bindings = map[string][][]string{}

	var (
		state        = flagsOrArgs
		anyArgs      bool
		appendOutput = s.bindings.appendOutput
	)

	// Skip program name
	args.pop()

Parsing:
	for !args.empty() {
		arg := args.pop()

		// end of options?
		if arg == "" || arg == "-" || arg[0] != '-' || state == argsOnly {
			for {
				if arg == "--" {
					state = argsOnly
					if err := bind.Done(); err != nil {
						return err
					}
					bind.next()
					continue Parsing
				}

				err := bind.SetArg(arg, state == flagsOrArgs)
				if err != nil {
					// Not accepted as an argument, possibly a flag per usual out of
					// order
					if arg[0] == '-' && state == flagsOrArgs {
						break
					}
					if isHardArgCountErr(err) {
						return err
					}
				}
				appendOutput(bind.name(), []string{arg})

				if args.empty() {
					if err := bind.Done(); err != nil {
						return err
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
		if len(s.longOptions) > 0 && arg[1] == '-' {
			e := strings.IndexRune(arg, '=')
			var value string
			if e > 0 {
				value = arg[e+1:]
				arg = arg[:e]
			}
			opt := s.longOptions[arg[2:]]
			// If we are processing long options then --f is -f
			// if f is not defined as a long option.
			// This lets you say --f=false
			if opt == nil && len(arg[2:]) == 1 {
				opt = s.shortOptions[rune(arg[2])]
			}
			if opt == nil {
				return unknownOption(arg[2:])
			}
			if disallowFlagsAfterArgs && anyArgs {
				return flagAfterArgError(arg[2:])
			}
			opt.flags |= internalFlagSpecifiedLong
			// If we require an option and did not have an =
			// then use the next argument as an option.
			if !opt.flags.flagOnly() && e < 0 && !opt.flags.optional() {
				if args.empty() {
					return missingArgument(opt)
				}

				if outputs, err := args.take(opt); err != nil {
					return setFlagError(opt, value, err)
				} else {
					appendOutput(opt.uname, append([]string{"--" + arg[2:]}, outputs...))
				}

				continue Parsing
			}

			opt.count++

			if err := opt.value.Set(value, opt); err != nil {
				return setFlagError(opt, value, err)
			}
			appendOutput(opt.uname, []string{arg, value})
			continue Parsing
		}

		// Short option processing
		arg = arg[1:] // strip -
		for i, c := range arg {
			opt := s.shortOptions[c]
			if opt == nil {
				// In traditional getopt, if - is not registered
				// as an option, a lone - is treated as
				// if there were a -- in front of it.
				if arg == "-" {
					// TODO Handle solitary dash
					continue
				}
				return unknownOption(c)
			}

			opt.flags = opt.flags & (^internalFlagSpecifiedLong)

			var value string
			if !opt.flags.flagOnly() {
				value = arg[1+i:]
				if value == "" && !opt.flags.optional() {
					if args.empty() {
						return missingArgument(opt)
					}

					if outputs, err := args.take(opt); err != nil {
						return setFlagError(opt, value, err)
					} else {
						appendOutput(opt.uname, append([]string{"-" + arg}, outputs...))
					}
					continue Parsing
				}
			}

			if err := opt.value.Set(value, opt); err != nil {
				return setFlagError(opt, value, err)
			}
			opt.count++
			appendOutput(opt.uname, []string{"-" + arg, value})
		}
	}

	return bind.Done()
}

func (s *set) defineFlag(res *internalOption) {
	if len(res.short) == 0 && len(res.long) == 0 {
		fmt.Fprintf(os.Stderr, "invalid flag definition, missing name or alias")
		os.Exit(1)
	}

	for _, short := range res.short {
		s.shortOptions[short] = res
	}
	for _, long := range res.long {
		s.longOptions[long] = res
	}

	s.values[res.uname] = res.value
}

func (s *set) defineArg(res *internalOption) {
	if res.uname == "" {
		res.uname = fmt.Sprintf("_%d", len(s.positionalOptions)+1)
	}

	s.values[res.uname] = res.value
	s.positionalOptions = append(s.positionalOptions, res)
}

func (s *set) withArgs(args []*Arg) *set {
	for _, a := range args {
		a.applyToSet(s)
	}
	return s
}

func (s genericValues) lookupValue(name string) (interface{}, bool) {
	if g, ok := s[name]; ok {
		return g.p, true
	}
	return nil, false
}

func (m bindingsMap) appendOutput(name string, args []string) {
	if e, ok := m[name]; ok {
		m[name] = append(e, args)
	} else {
		m[name] = [][]string{args}
	}
}

func (m bindingsMap) lookup(name string, occurs bool) []string {
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

func (a *argList) take(opt *internalOption) (output []string, err error) {
	bind := newArgBinding([]*internalOption{opt})
	output = make([]string, 0)
	const possibleFlag = true
	for !a.empty() {
		arg := a.current()
		err = bind.SetArg(arg, possibleFlag)
		if err != nil {
			// Not accepted as an argument, possibly a flag per usual out of
			// order
			if arg[0] == '-' && possibleFlag {
				break
			}

			// HACK This should not unwrap the Error type, which is a bit fragile
			if e, ok := err.(*Error); ok {
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
		arg = a.pop()
	}

	err = bind.Done()
	return
}

func (a *argBinding) next() bool {
	a.index++
	return a.index < len(a.items)
}

func (a *argBinding) name() string {
	return a.items[a.index].uname
}

func (a *argBinding) hasCurrent() bool {
	return a.index < len(a.items)
}

func (a *argBinding) current() (*internalOption, ArgCounter) {
	if a.index < len(a.items) {
		return a.items[a.index], a.takers[a.index]
	}
	return nil, nil
}

func (a *argBinding) Done() error {
	if _, c := a.current(); c != nil {
		return c.Done()
	}
	return nil
}

func (a *argBinding) SetArg(arg string, possibleFlag bool) error {
	for {
		c, t := a.current()
		if c == nil {
			return unexpectedArgument(arg)
		}

		err := t.Take(arg, possibleFlag)
		if err == nil {
			c.Occurrence()
			return c.Set(arg)
		}

		if isHardArgCountErr(err) {
			return err
		}

		a.next()
	}
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

func isNextExpr(arg string, err error) bool {
	// HACK This should not unwrap the Error type, which is a bit fragile
	if e, ok := err.(*Error); ok {
		if e.Code == UnexpectedArgument {
			return arg[0] == '-'
		}
	}
	return false
}

func allowFlag(arg string, possibleFlag bool) bool {
	return len(arg) > 0 && (possibleFlag && arg[0] == '-')
}

func (o *internalOption) Seen() bool {
	return o.count > 0
}

func (o *internalOption) Set(arg string) error {
	return o.value.Set(arg, o)
}

func (o *internalOption) Occurrence() {
	o.count++
}

func (o *internalOption) Name() string {
	if !o.flags.specifiedLong() && len(o.short) > 0 {
		return "-" + string(o.short[0])
	}
	return "--" + o.long[0]
}

func (o *internalOption) Value() *generic {
	return o.value
}

func (o *internalOption) Occurrences() int {
	return o.count
}

func (o *internalOption) actualArgCounter() ArgCounter {
	if o.narg == nil {
		if o.value == nil || o.value.p == nil {
			return ArgCount(0)
		}
		switch value := o.value.p.(type) {
		case *[]string:
			return ArgCount(TakeUntilNextFlag)
		case valueProvidesCounter:
			return value.NewCounter()
		}
	}
	return ArgCount(o.narg)
}

var _ Lookup = (*set)(nil)
