package cli

import (
	"github.com/pborman/getopt/v2"
)

// DON'T EDIT THIS FILE.  This file was generated from a template, flag.go.tpl

// BoolFlag provides a bool flag
type BoolFlag struct {
	Name        string
	Value       bool
	Default     string
	Destination *bool
	Action      ActionFunc
	HelpText    string

	option getopt.Option
}

// BoolArg provides a bool arg
type BoolArg struct {
	Name        string
	Value       bool
	Default     string
	Destination *bool
	Action      ActionFunc
	HelpText    string

	internal boolValue
}

func (c *Context) Bool(name string) bool {
	return c.Value(name).(bool)
}

func (f *BoolFlag) applyToSet(s *getopt.Set) {
	dest := f.Destination

	if dest == nil {
		var p bool
		dest = &p
	}
	for _, name := range f.Names() {
		long, short := flagName(name)
		f.option = s.FlagLong(dest, long, short, f.HelpText, name)
	}
}

func (f *BoolFlag) Names() []string {
	return names(f.Name)
}

func (a *BoolArg) Set(arg string) error {
	err := a.internal.Set(arg)
	if err != nil {
		return err
	}
	dest := a.Destination

	if dest == nil {
		var p bool
		dest = &p
	}
	*dest = bool(a.internal)
	return nil
}
