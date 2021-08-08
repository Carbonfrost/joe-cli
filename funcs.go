package cli

import (
	"github.com/pborman/getopt/v2"
)

type Arg interface {
	Set(string) error
}

type Flag interface {
	applyToSet(s *getopt.Set)
}
