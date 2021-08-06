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
}

// StringArg provides a string arg
type StringArg struct {
	Name   string
	Value  func(*string)
	Action ActionFunc
}

func (c *Context) String(name string) string {
	return c.Value(name).(string)
}

func (f *StringFlag) applyToSet(s *getopt.Set) {
	for _, name := range f.Names() {
		long, short := flagName(name)
		s.StringLong(long, short, f.Default, f.HelpText, name)
	}
}

func (f *StringFlag) Names() []string {
	return names(f.Name)
}

func (f *StringArg) Getopt(args []string) error {
	return nil
}
