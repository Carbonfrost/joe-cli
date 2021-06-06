package gocli

import (
	"github.com/pborman/getopt"
)

// ActionFunc provides the basic function for
type ActionFunc func(*Context) error

type Arg interface {
	Getopt(args []string) error
}

type Flag interface {
	applyToSet(s *getopt.Set)
}

func emptyAction(*Context) error {
	return nil
}

func execute(af ActionFunc, c *Context) error {
	if af == nil {
		return nil
	}
	return af(c)
}
