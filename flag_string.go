package cli

import (
	"github.com/pborman/getopt"
)

// DON'T EDIT THIS FILE.  This file was generated from a template, flag.go.tpl

// StringFlag provides a string flag
type StringFlag struct {
	Name        string
	Value       string
	Default     string
	Destination *string
	Action      ActionFunc
	HelpText    string

	option getopt.Option
}

// StringArg provides a string arg
type StringArg struct {
	Name        string
	Value       string
	Default     string
	Destination *string
	Action      ActionFunc
	HelpText    string

	internal stringValue
}

func (c *Context) String(name string) string {
	return c.Value(name).(string)
}

func (f *StringFlag) applyToSet(s *getopt.Set) {
	dest := f.Destination

	if dest == nil {
		var p string
		dest = &p
	}
	for _, name := range f.Names() {
		long, short := flagName(name)
		f.option = s.StringVarLong(dest, long, short, f.HelpText, name)
	}
}

func (f *StringFlag) Names() []string {
	return names(f.Name)
}

func (a *StringArg) Set(arg string) error {
	err := a.internal.Set(arg)
	if err != nil {
		return err
	}
	dest := a.Destination

	if dest == nil {
		var p string
		dest = &p
	}
	*dest = string(a.internal)
	return nil
}
