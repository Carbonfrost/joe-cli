package expr

import (
	"github.com/Carbonfrost/joe-cli"
)

// FromContext obtains the expression from the context
func FromContext(c *cli.Context, name string) *Expression {
	return c.Value(name).(*Expression)
}

// SetEvaluator provides an action used in the Uses pipeline of the Expr
// which updates its evaluator
func SetEvaluator(e Evaluator) cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		c.Target().(*Expr).Evaluate = e
		return nil
	})
}
