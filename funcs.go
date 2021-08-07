package cli

import (
	"github.com/pborman/getopt"
)

type Arg interface {
	Set(string) error
}

type Flag interface {
	applyToSet(s *getopt.Set)
}
