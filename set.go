package cli

import (
	"github.com/pborman/getopt/v2"
)

type Set struct {
	internal *getopt.Set
	values   map[string]interface{}
}

type optionWrapper struct {
	getopt.Option

	count int
	value getopt.Value
}

func newSet() *Set {
	return &Set{
		internal: getopt.New(),
		values:   map[string]interface{}{},
	}
}

func (s *Set) LookupValue(name string) (interface{}, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.values[name]
	return v, ok
}

func (s *Set) Args() []string {
	return s.internal.Args()
}

func (s *Set) Getopt(args []string) error {
	return s.internal.Getopt(args, nil)
}

func (s *Set) defineFlag(name string, alias string, v interface{}) getopt.Option {
	long, short := flagName(alias)
	s.values[name] = v
	return s.internal.FlagLong(wrapFlagLong(v), long, short, "help text unsed", name)
}

func (s *Set) defineArg(name string, v interface{}) *optionWrapper {
	s.values[name] = v
	return &optionWrapper{
		value: wrapGeneric(v),
	}
}

func (o *optionWrapper) Value() getopt.Value {
	return o.value
}

func (o *optionWrapper) Count() int {
	return o.count
}

func flagName(name string) (string, rune) {
	if len(name) == 1 {
		return "", []rune(name)[0]
	} else {
		return name, 0
	}
}
