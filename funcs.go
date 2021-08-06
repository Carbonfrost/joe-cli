package cli

import (
	"github.com/pborman/getopt"
)

type Arg interface {
	Getopt(args []string) error
}

type Flag interface {
	applyToSet(s *getopt.Set)
}
