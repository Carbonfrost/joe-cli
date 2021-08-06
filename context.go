package cli

import (
	"context"

	"github.com/pborman/getopt"
)

// Context provides the context in which the app, command, or flag is executing
type Context struct {
	context.Context

	set *getopt.Set
}

func (*Context) Value(name string) interface{} {
	panic("not implemented: context value")
}
