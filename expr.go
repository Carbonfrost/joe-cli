package cli

import (
	"fmt"
	"strings"
)

type ExprEvaluator interface {
	Evaluate(c *Context, v interface{}, yield func(interface{}) error) error
}

type EvaluatorFunc func(*Context, interface{}, func(interface{}) error) error

type Expr struct {
	Name string
	Args []*Arg

	HelpText  string
	UsageText string

	// Evaluate provides the evaluation behavior for the expression.  The value should
	// implement ExprEvaluator or support runtime conversion to that interface via
	// the rules provided by the cli.Evaluator function.
	Evaluate interface{}

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	flags internalFlags
}

// Expression provides the parsed result of the expression that can be evaluated
// with the given inputs.  Clients typically don't implement this interface; it is returned
// from the Context.Expression() method assuming that expressions have be defined for the
// command or app.
type Expression interface {
	Value

	// Evaluate the expression binding with the given inputs
	Evaluate(*Context, ...interface{}) error

	// Walk iterates the contents of the expression.  The walker function is passed
	// each expression that is encountered along with the operands which were bound for it.
	// If the walker function returns an error, walking stops and returns that error.
	Walk(func(ExprBinding) error) error

	// Append will append the specified expression evaluator
	Append(ExprBinding)
}

// ExprBinding provides the relationship between an evaluator and the evaluation
// context.  Optionally, the original Expr is made available
type ExprBinding interface {
	ExprEvaluator
	Expr() *Expr
}

// ExprsByName is a sortable slice for exprs
type ExprsByName []*Expr

type yielder = func(interface{}) error

type exprPipelineFactory struct {
	exprs map[string]func() *boundExpr
}

type exprPipeline struct {
	items []ExprBinding
	args  []string
}

type exprSynopsis struct {
	name  string
	args  []*argSynopsis
	usage *usage
}

type boundExpr struct {
	expr *Expr
	set  *set
}

type exprBinding struct {
	ExprEvaluator
	expr *Expr
}

func BindExpression(exprFunc func(*Context) ([]*Expr, error)) ActionHandler {
	return ActionFunc(func(c *Context) error {
		exprs, err := exprFunc(c)
		if err != nil {
			return err
		}

		pipe := c.Value("").(*exprPipeline)
		fac := newExprPipelineFactory(exprs)
		return pipe.applyFactory(fac)
	})
}

func NewExprBinding(ev ExprEvaluator, exprlookup ...interface{}) ExprBinding {
	var (
		expr *Expr
	)
	switch len(exprlookup) {
	case 2:
		fallthrough
	case 1:
		expr = exprlookup[0].(*Expr)
	}
	return &exprBinding{
		ExprEvaluator: ev,
		expr:          expr,
	}
}

// Evaluator creates an expression evaluator for a given value.  The
// value must be bool or a function.  If a bool, then it works as a predicate
// for the corresponding invariant (i.e. false filters out all values, and true
// includes all values).  If a function, the signature must match either the
// ExprEvaluator.Evaluate function signature or a  variation that excludes
// the context and/or yielder.
// You can also use bool as a return type as in the same signature used by
// Predicate.  These are valid signatures:
//
//   * func(*Context, interface{}, func(interface{})error) error
//   * func(*Context, interface{}) error
//   * func(*Context, interface{}) bool
//   * func(*Context, interface{})
//   * func(interface{}, func(interface{})error) error
//   * func(interface{}) bool
//   * func(interface{}) error
//   * func(interface{})
//
func Evaluator(v interface{}) ExprEvaluator {
	switch a := v.(type) {
	case nil:
		return Evaluator(true)
	case ExprEvaluator:
		return a
	case func(*Context, interface{}, func(interface{}) error) error:
		return EvaluatorFunc(a)
	case func(*Context, interface{}) error:
		return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
			if err := a(c, v); err == nil {
				return y(v)
			} else {
				return err
			}
		})
	case func(*Context, interface{}) bool:
		return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
			if a(c, v) {
				return y(v)
			}
			return nil
		})
	case func(*Context, interface{}):
		return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
			a(c, v)
			return y(v)
		})
	case func(interface{}, func(interface{}) error) error:
		return EvaluatorFunc(func(_ *Context, v interface{}, y func(interface{}) error) error {
			return a(v, y)
		})
	case func(interface{}) error:
		return EvaluatorFunc(func(_ *Context, v interface{}, y func(interface{}) error) error {
			if err := a(v); err == nil {
				return y(v)
			} else {
				return err
			}
		})
	case func(interface{}) bool:
		return EvaluatorFunc(func(_ *Context, v interface{}, y func(interface{}) error) error {
			if a(v) {
				return y(v)
			}
			return nil
		})
	case func(interface{}):
		return EvaluatorFunc(func(_ *Context, v interface{}, y func(interface{}) error) error {
			a(v)
			return y(v)
		})
	case bool:
		return EvaluatorFunc(func(_ *Context, v interface{}, y func(interface{}) error) error {
			if a {
				return y(v)
			}
			return nil
		})
	}
	panic(fmt.Sprintf("unexpected type: %T", v))
}

// Predicate provides a simple predicate which filters values.
func Predicate(filter func(v interface{}) bool) EvaluatorFunc {
	return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
		if ok := filter(v); ok {
			return y(v)
		}
		return nil
	})
}

func newExprPipelineFactory(exprs []*Expr) *exprPipelineFactory {
	res := &exprPipelineFactory{
		exprs: map[string]func() *boundExpr{},
	}
	for _, e := range exprs {
		e1 := e
		res.exprs[e.Name] = func() *boundExpr {
			return &boundExpr{
				expr: e1,
				set:  newSet().withArgs(e1.Args),
			}
		}
	}
	return res
}

func (e *Expr) Synopsis() string {
	return textUsage.expr(e.newSynopsis())
}

func (e *Expr) newSynopsis() *exprSynopsis {
	args := make([]*argSynopsis, len(e.actualArgs()))
	usage := parseUsage(e.HelpText)
	pp := usage.Placeholders()

	for i, a := range e.actualArgs() {
		if i < len(pp) {
			args[i] = a.newSynopsisCore(pp[i])
		} else {
			// Use a simpler default name with less noise from angle brackets
			args[i] = a.newSynopsisCore(strings.ToUpper(a.Name))
		}
	}

	return &exprSynopsis{
		name:  fmt.Sprintf("-%s", e.Name),
		usage: usage,
		args:  args,
	}
}

func (e *Expr) actualArgs() []*Arg {
	if e.Args == nil {
		return make([]*Arg, 0)
	}
	return e.Args
}

func (e EvaluatorFunc) Evaluate(c *Context, v interface{}, yield func(interface{}) error) error {
	if e == nil {
		return nil
	}
	return e(c, v, yield)
}

func (e ExprsByName) Len() int {
	return len(e)
}

func (e ExprsByName) Less(i, j int) bool {
	return e[i].Name < e[j].Name
}

func (e ExprsByName) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (b *boundExpr) Expr() *Expr {
	return b.expr
}

func (b *boundExpr) Evaluate(c *Context, v interface{}, yield func(interface{}) error) error {
	// TODO Pass along args
	args := []string{}
	ctx := c.exprContext(b.expr, args, b.set)
	return Evaluator(b.expr.Evaluate).Evaluate(ctx, v, yield)
}

func (p *exprPipeline) Set(arg string) error {
	if p.args == nil {
		p.args = make([]string, 0)
	}
	p.args = append(p.args, arg)
	return nil
}

func (p *exprPipeline) String() string {
	return strings.Join(p.args, " ")
}

func (p *exprPipeline) Evaluate(ctx *Context, items ...interface{}) error {
	yielders := make([]yielder, len(p.items))
	yielderThunk := func(i int) yielder {
		if i >= len(yielders) || yielders[i] == nil {
			return emptyYielder
		}
		return yielders[i]
	}

	for ik := range p.items {
		i := ik
		yielders[i] = func(in interface{}) error {
			return p.items[i].Evaluate(ctx, in, yielderThunk(i+1))
		}
	}
	for _, v := range items {
		err := yielderThunk(0)(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *exprPipeline) Walk(fn func(ExprBinding) error) error {
	if fn == nil {
		return nil
	}
	for _, e := range p.items {
		err := fn(e)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *exprPipeline) Append(expr ExprBinding) {
	p.items = append(p.items, expr)
}

func (e *exprPipelineFactory) parse(args []string) ([]ExprBinding, error) {
	results := make([]ExprBinding, 0)

	if len(args) == 0 {
		return results, nil
	}

Parsing:
	for len(args) > 0 {
		arg := args[0]
		args = args[1:]

		if arg[0] != '-' {
			return nil, argsMustPrecedeExprs(arg)
		}

		s, ok := e.exprs[arg[1:]]
		if !ok {
			return nil, unknownExpr(arg)
		}

		set := s()
		results = append(results, set)
		if len(set.set.positionalOptions) == 0 {
			continue Parsing
		}

		bind := newArgBinding(set.set.positionalOptions)
		for {
			arg = args[0]
			args = args[1:]

			err := bind.SetArg(arg, true)
			if err != nil {
				if isHardArgCountErr(err) {
					return nil, wrapExprError(set.expr.Name, err)
				}
			}

			if len(args) == 0 {
				if err := bind.Done(); err != nil {
					return nil, wrapExprError(set.expr.Name, err)
				}
				continue Parsing
			}
		}
	}
	return results, nil
}

func (p *exprPipeline) applyFactory(fac *exprPipelineFactory) error {
	exprs, err := fac.parse(p.args)
	p.items = exprs
	return err
}

func (b *exprBinding) Expr() *Expr {
	return b.expr
}

func emptyYielder(interface{}) error {
	return nil
}

var _ Expression = &exprPipeline{}
