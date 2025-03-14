package expr

import (
	"github.com/Carbonfrost/joe-cli"
)

// FromContext obtains the expression from the context
func FromContext(c *cli.Context, name string) *Expression {
	return c.Value(name).(*Expression)
}
