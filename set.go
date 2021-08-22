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
	takers []func() bool
	index  int
}

type internalOption struct {
	short    rune   // 0 means no short name
	long     string // "" means no long name
	isLong   bool   // True if they used the long name
	flag     bool   // true if a boolean flag
	optional bool   // true if we take an optional value
	where    string // file where the option was defined
	value    *generic
	count    int
	uname    string
	narg     int
}

type parserState int

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
	takers := make([]func() bool, len(args))

	for i, x := range args {
		items[i] = x
		takers[i] = takeArgs(x.narg)
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
				err := bind.SetArg(arg)
				if err != nil {
					// Not accepted as an argumnet, possibly a flag per usual out of
					// order
					if arg[0] == '-' && state == flagsOrArgs {
						break
					}
					return err
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
	return nil
}

func (s *set) defineFlag(name string, alias string, p interface{}) *internalOption {
	long, short := flagName(alias)
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
	if res.short == 0 && res.long == "" {
		fmt.Fprintf(os.Stderr, res.where+": invalid definition, missing name or alias")
		os.Exit(1)
	}

	if len(long) == 0 {
		s.shortOptions[short] = res
	} else {
		s.longOptions[long] = res
	}
	s.values[name] = p
	return res
}

func (s *set) defineArg(name string, v interface{}, narg int) *internalOption {
	opt := &internalOption{
		value: wrapGeneric(v),
		narg:  narg,
	}

	s.values[name] = v
	s.positionalOptions = append(s.positionalOptions, opt)
	return opt
}

func (a *argBinding) next() bool {
	a.index += 1
	return a.index < len(a.items)
}

func (a *argBinding) current() (*internalOption, func() bool) {
	if a.index < len(a.items) {
		return a.items[a.index], a.takers[a.index]
	}
	return nil, nil
}

func (a *argBinding) Done() error {
	// TODO Handle end of binding
	return nil
}

func (a *argBinding) SetArg(arg string) error {
	c, t := a.current()
	if c == nil {
		return unexpectedArgument(arg)
	}

	if t() {
		if err := c.Set(arg); err != nil {
			return err
		}
		return nil
	}

	if !a.next() {
		return unexpectedArgument(arg)
	}

	c, t = a.current()

	if t() {
		if err := c.Set(arg); err != nil {
			return err
		}
		return nil

	} else {
		return unexpectedArgument(arg)
	}
}

func (o *internalOption) Seen() bool {
	return o.count > 0
}

func (o *internalOption) Set(arg string) error {
	o.count += 1
	return o.value.Set(arg, o)
}

func (o *internalOption) Name() string {
	if !o.isLong && o.short != 0 {
		return "-" + string(o.short)
	}
	return "--" + o.long
}

func (o *internalOption) Value() *generic {
	return o.value
}

func (o *internalOption) Count() int {
	return o.count
}

func flagName(name string) (string, rune) {
	if len(name) == 1 {
		return "", []rune(name)[0]
	} else {
		return name, 0
	}
}

func takeArgs(narg int) func() bool {
	if narg < 0 {
		return func() bool {
			return true
		}
	}
	if narg == 0 {
		narg = 1
	}
	return func() bool {
		narg = narg - 1
		return narg >= 0
	}
}
