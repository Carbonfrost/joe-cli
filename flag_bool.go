package gocli

import (
	"github.com/pborman/getopt"
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
}

// BoolArg provides a bool arg
type BoolArg struct {
	Name   string
	Value  func(*bool)
	Action ActionFunc
}

func (c *Context) Bool(name string) bool {
	return c.Value(name).(bool)
}

func (f *BoolFlag) applyToSet(s *getopt.Set) {
	for _, name := range f.Names() {
		long, short := flagName(name)
		s.BoolLong(long, short, f.Default, f.HelpText, name)
	}
}

func (f *BoolFlag) Names() []string {
	return names(f.Name)
}

func (f *BoolArg) Getopt(args []string) error {
	return nil
}
