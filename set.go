package cli

import (
	"fmt"
	"os"
	"strings"
)

type set struct {
	shortOptions      map[rune]*internalOption
	longOptions       map[string]*internalOption
	positionalOptions []*internalOption
	values            map[string]interface{}
}

type argBinding struct {
	items  []*internalOption
	takers []ArgCounter
	index  int
}

type internalOption struct {
	short      []rune   // 0 means no short name
	long       []string // "" means no long name
	isLong     bool     // True if they used the long name
	flag       bool     // true if a boolean flag
	optional   bool     // true if we take an optional value
	where      string   // file where the option was defined
	value      *generic
	count      int
	uname      string
	narg       interface{}
	persistent bool // true when the option is a clone of a persistent flag
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

func newSet() *set {
	return &set{
		values:            map[string]interface{}{},
		shortOptions:      map[rune]*internalOption{},
		longOptions:       map[string]*internalOption{},
		positionalOptions: []*internalOption{},
	}
}

func newArgBinding(args []*internalOption) *argBinding {
	items := make([]*internalOption, len(args))
	takers := make([]ArgCounter, len(args))

	for i, x := range args {
		items[i] = x
		takers[i] = ArgCount(x.narg)
	}
	return &argBinding{
		items, takers, 0,
	}
}

func (s *set) lookupValue(name string) (interface{}, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.values[name]
	return v, ok
}

func (s *set) parse(args []string) error {
	if len(args) == 0 {
		return nil
	}

	bind := newArgBinding(s.positionalOptions)
	var state parserState = flagsOrArgs

	// Skip program name
	args = args[1:]

Parsing:
	for len(args) > 0 {
		arg := args[0]
		args = args[1:]

		// end of options?
		if arg == "" || arg[0] != '-' {
			for {
				err := bind.SetArg(arg, state == flagsOrArgs)
				if err != nil {
					// Not accepted as an argumnet, possibly a flag per usual out of
					// order
					if arg[0] == '-' && state == flagsOrArgs {
						break
					}
					if isHardArgCountErr(err) {
						return err
					}
				}

				if len(args) == 0 {
					if err := bind.Done(); err != nil {
						return err
					}
					continue Parsing
				}

				arg = args[0]
				args = args[1:]
			}
		}

		if arg == "-" {
			goto ShortParsing
		}

		// explicitly request end of options?
		if arg == "--" {
			state = argsOnly
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
			opt.isLong = true
			// If we require an option and did not have an =
			// then use the next argument as an option.
			if !opt.flag && e < 0 && !opt.optional {
				if len(args) == 0 {
					return missingArgument(opt)
				}
				value = args[0]
				args = args[1:]
			}
			opt.count++

			if err := opt.value.Set(value, opt); err != nil {
				return setFlagError(opt, value, err)
			}
			continue Parsing
		}

		// Short option processing
		arg = arg[1:] // strip -
	ShortParsing:
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
			opt.isLong = false
			opt.count++
			var value string
			if !opt.flag {
				value = arg[1+i:]
				if value == "" && !opt.optional {
					if len(args) == 0 {
						return missingArgument(opt)
					}
					value = args[0]
					args = args[1:]
				}
			}
			if err := opt.value.Set(value, opt); err != nil {
				return setFlagError(opt, value, err)
			}

			if !opt.flag {
				continue Parsing
			}
		}
	}

	return bind.Done()
}

func (s *set) defineFlag(name string, aliases []string, p interface{}) *internalOption {
	long, short := canonicalNames(name, aliases)
	res := &internalOption{
		short: short,
		long:  long,
		value: wrapGeneric(p),
		uname: name,
	}

	switch p.(type) {
	case *bool:
		res.flag = true
	}

	res.where = calledFrom()
	if len(res.short) == 0 && len(res.long) == 0 {
		fmt.Fprintf(os.Stderr, res.where+": invalid definition, missing name or alias")
		os.Exit(1)
	}

	for _, short := range res.short {
		s.shortOptions[short] = res
	}
	for _, long := range res.long {
		s.longOptions[long] = res
	}

	s.values[name] = p
	return res
}

func (s *set) defineArg(name string, v interface{}, narg interface{}) *internalOption {
	opt := &internalOption{
		value: wrapGeneric(v),
		narg:  narg,
	}

	s.values[name] = v
	s.positionalOptions = append(s.positionalOptions, opt)
	return opt
}

func (s *set) withArgs(args []*Arg) *set {
	for _, a := range args {
		a.applyToSet(s)
	}
	return s
}

func (a *argBinding) next() bool {
	a.index += 1
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
	return len(arg) > 0 && (possibleFlag && arg[0] == '-')
}

func (o *internalOption) Seen() bool {
	return o.count > 0
}

func (o *internalOption) Set(arg string) error {
	o.count += 1
	return o.value.Set(arg, o)
}

func (o *internalOption) Name() string {
	if !o.isLong && len(o.short) > 0 {
		return "-" + string(o.short[0])
	}
	return "--" + o.long[0]
}

func (o *internalOption) Value() *generic {
	return o.value
}

func (o *internalOption) Count() int {
	return o.count
}
