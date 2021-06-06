package gocli

import (
  "github.com/pborman/getopt"
)

// DON'T EDIT THIS FILE.  This file was generated from a template, flag.go.tpl
{{ $flagName := (printf "%s%s" .Name "Flag") }}
{{ $argName  := (printf "%s%s" .Name "Arg") }}

// {{ $flagName }} provides a {{ .Type }} flag
type {{ $flagName }} struct {
  Name        string
  Value       {{ .Type }}
  Default     string
  Destination *{{ .Type }}
  Action      ActionFunc
  HelpText    string
}

// {{ $argName }} provides a {{ .Type }} arg
type {{ $argName }} struct {
  Name   string
  Value  func(*{{ .Type }})
  Action ActionFunc
}

func (c *Context) {{ .Name }}(name string) {{ .Type }} {
  return c.Value(name).({{ .Type }})
}

func (f *{{ $flagName }}) applyToSet(s *getopt.Set) {
  for _, name := range f.Names() {
    long, short := flagName(name)
    s.{{ .Name }}Long(long, short, f.Default, f.HelpText, name)
  }
}

func (f *{{ $flagName }}) Names() []string {
  return names(f.Name)
}

func (f *{{ $argName }}) Getopt(args []string) error {
  return nil
}
