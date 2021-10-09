package cli

import (
	"fmt"
	"sort"
	"strings"
)

//counterfeiter:generate . Evaluator

// Evaluator provides the evluation function for an expression operator.
type Evaluator interface {
	// Evaluate performs the evaluation.  The v argument is the value of the prior
	// expression operator.  The yield argument is used to pass one or more additional
	// values to the next expression operator.
	Evaluate(c *Context, v interface{}, yield func(interface{}) error) error
}

// EvaluatorFunc provides the basic function for an Evaluator
type EvaluatorFunc func(*Context, interface{}, func(interface{}) error) error

// Expr represents an operator in an expression.  An expression is composed of an
// ordered series of operators meant to describe how to process one or more values.
// A well-known implementation of an expression is in the Unix `find` command where
// each file is processed through a series of operands to filter a list of files.
type Expr struct {
	// Name provides the name of the expression operator. This value must be set, and it is used to access
	// the expression operator's value via the context
	Name string

	// Aliases provides a list of alternative names for the expression operator.  In general, Name should
	// be used for the long name of the expression operator, and Aliases should contain the short name.
	// If there are additional names for compatibility reasons, they should be included
	// with Aliases but listed after the preferred names. Note that only one short name
	// and one long name is displayed on help screens by default.
	Aliases []string

	// Args contains each of the arguments that are processed for the expression operators.  Expression
	// operators don't contain values directly; they process one or more arguments.
	Args []*Arg

	// HelpText contains text which briefly describes the usage of the expression operator.
	// For style, generally the usage text should be limited to about 40 characters.
	// Sentence case is recommended for the usage text.    Uppercase is recommended for the
	// text of placeholders.  The placeholder is used in the synoposis for the expression operator as well
	// as error messages.
	HelpText string

	// UsageText provides the usage for the expression operator.  If left blank, a succint synopsis
	// is generated from the type of the expression operator's arguments
	UsageText string

	// Category specifies the expression operator category.  When categories are used, operators are grouped
	// together on the help screen
	Category string

	// Evaluate provides the evaluation behavior for the expression.  The value should
	// implement Evaluator or support runtime conversion to that interface via
	// the rules provided by the cli.EvaluatorOf function.
	Evaluate interface{}

	// Before executes before the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// After executes after the command runs.  Refer to cli.Action about the correct
	// function signature to use.
	After interface{}

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses interface{}

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	// Options sets various options about how to treat the expression operator.  For example, options can
	// hide the expression operator.
	Options Option

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
	Evaluator
	Lookup

	// Expr retrieves the expression operator if it is available
	Expr() *Expr
}

// ExprsByName is a sortable slice for exprs
type ExprsByName []*Expr

// ExprsByCategory provides a slice that can sort on category names and the expression operators
// themselves
type ExprsByCategory []*ExprCategory

// ExprCategory names a category and the expression operators it contains
type ExprCategory struct {
	// Category is the name of the category
	Category string

	// Exprs contains the expression operators in the category
	Exprs []*Expr
}

//counterfeiter:generate . Yielder

// Yielder provides the signature of the function used to yield
// values to the expression pipeline
type Yielder func(interface{}) error

type exprPipelineFactory struct {
	exprs map[string]func(*Context) *boundExpr
}

type exprPipeline struct {
	items []ExprBinding
	args  []string
}

type exprSynopsis struct {
	long  string
	short string
	args  []*argSynopsis
	usage *usage
}

type boundExpr struct {
	*Context
	expr *Expr
	set  *set
}

type exprBinding struct {
	Evaluator
	Lookup
	expr *Expr
}

type exprContext struct {
	expr  *Expr
	args_ []string
	set_  *set
}

// BindExpression is an action that binds expression handling to an argument.  This
// is set up automatically when a command defines any expression operators.
func BindExpression(exprFunc func(*Context) ([]*Expr, error)) Action {
	return ActionFunc(func(c *Context) error {
		exprs, err := exprFunc(c)
		if err != nil {
			return err
		}

		pipe := c.Value("").(*exprPipeline)
		fac := newExprPipelineFactory(exprs)
		return pipe.applyFactory(c, fac)
	})
}

// NewExprBinding creates an expression binding.  The ev parameter is how
// the expression is evaluated.  The remaining arguments specify the *Expr
// expression operator to use and optionally a Lookup.   The main use case
// for this function is to create a custom evaluation step that is appended to
// the expression pipeline
func NewExprBinding(ev Evaluator, exprlookup ...interface{}) ExprBinding {
	var (
		expr   *Expr
		lookup Lookup = LookupValues{}
	)
	switch len(exprlookup) {
	case 2:
		lookup = exprlookup[1].(Lookup)
		fallthrough
	case 1:
		expr = exprlookup[0].(*Expr)
	}
	return &exprBinding{
		Evaluator: ev,
		Lookup:    lookup,
		expr:      expr,
	}
}

// EvaluatorOf creates an expression evaluator for a given value.  The
// value must be bool or a function.  If a bool, then it works as a predicate
// for the corresponding invariant (i.e. false filters out all values, and true
// includes all values).  If a function, the signature must match either the
// Evaluator.Evaluate function signature or a  variation that excludes
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
func EvaluatorOf(v interface{}) Evaluator {
	switch a := v.(type) {
	case nil:
		return EvaluatorOf(true)
	case Evaluator:
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

// Predicate provides a simple predicate which filters values.  The filter function
// takes the prior operand and returns true or false depending upon whether the
// operand should be yielded to the next step in the expression pipeline.
func Predicate(filter func(v interface{}) bool) EvaluatorFunc {
	return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
		if ok := filter(v); ok {
			return y(v)
		}
		return nil
	})
}

// GroupExprsByCategory groups together expression operators by category and sorts the groupings.
func GroupExprsByCategory(exprs []*Expr) ExprsByCategory {
	res := ExprsByCategory{}
	all := map[string]*ExprCategory{}
	category := func(name string) *ExprCategory {
		if c, ok := all[name]; ok {
			return c
		}
		c := &ExprCategory{Category: name, Exprs: []*Expr{}}
		all[name] = c
		res = append(res, c)
		return c
	}
	for _, e := range exprs {
		cc := category(e.Category)
		cc.Exprs = append(cc.Exprs, e)
	}
	sort.Sort(res)
	return res
}

func newExprPipelineFactory(exprs []*Expr) *exprPipelineFactory {
	res := &exprPipelineFactory{
		exprs: map[string]func(*Context) *boundExpr{},
	}
	for _, e := range exprs {
		e1 := e
		fac := func(c *Context) *boundExpr {
			// TODO Pass along args
			args := []string{}
			set := newSet().withArgs(e1.Args)
			return &boundExpr{
				expr:    e1,
				set:     set,
				Context: c.exprContext(e1, args, set).setTiming(actionTiming),
			}
		}
		res.exprs[e.Name] = fac
		for _, alias := range e.Aliases {
			res.exprs[alias] = fac
		}
	}
	return res
}

// Names obtains the name of the expression operator and its aliases
func (e *Expr) Names() []string {
	return append([]string{e.Name}, e.Aliases...)
}

// Synopsis retrieves the synopsis for the expression operator.
func (e *Expr) Synopsis() string {
	return textUsage.expr(e.newSynopsis())
}

// Arg gets the expression operator by name
func (e *Expr) Arg(name string) (*Arg, bool) {
	return findArgByName(e.Args, name)
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
	long, short := canonicalNames(e.Name, e.Aliases)

	return &exprSynopsis{
		long:  longName(long),
		short: shortName(short),
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

// SetData sets the specified metadata on the expression operator
func (e *Expr) SetData(name string, v interface{}) {
	e.ensureData()[name] = v
}

func (e *Expr) setCategory(name string) {
	e.Category = name
}

func (e *Expr) ensureData() map[string]interface{} {
	if e.Data == nil {
		e.Data = map[string]interface{}{}
	}
	return e.Data
}

func (e *Expr) hooks() *hooks {
	return nil
}

func (e *Expr) setInternalFlags(f internalFlags) {
	e.flags |= f
}

func (e *Expr) internalFlags() internalFlags {
	return e.flags
}

func (e *Expr) options() Option {
	return e.Options
}

func (e *Expr) appendAction(t timing, ah Action) {
}

// Evaluate provides the evaluation of the function and implements the Evaluator interface
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

// VisibleExprs filters all operatorss in the expression operators category by whether
// they are not hidden
func (f *ExprCategory) VisibleExprs() []*Expr {
	res := make([]*Expr, 0, len(f.Exprs))
	for _, o := range f.Exprs {
		if o.flags.hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

// Undocumented determines whether the category is undocumented (i.e. has no HelpText set
// on any of its expression operators)
func (e *ExprCategory) Undocumented() bool {
	for _, x := range e.Exprs {
		if x.HelpText != "" {
			return false
		}
	}
	return true
}

func (e ExprsByCategory) Less(i, j int) bool {
	return e[i].Category < e[j].Category
}

func (e ExprsByCategory) Len() int {
	return len(e)
}

func (e ExprsByCategory) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (b *boundExpr) Expr() *Expr {
	return b.expr
}

func (b *boundExpr) Evaluate(c *Context, v interface{}, yield func(interface{}) error) error {
	return EvaluatorOf(b.expr.Evaluate).Evaluate(b.Context, v, yield)
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
	yielders := make([]Yielder, len(p.items))
	yielderThunk := func(i int) Yielder {
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

func (e *exprPipelineFactory) parse(c *Context, args []string) ([]ExprBinding, error) {
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

		set := s(c)
		results = append(results, set)
		if len(set.set.positionalOptions) == 0 {
			continue Parsing
		}

		bind := newArgBinding(set.set.positionalOptions)
		for len(args) > 0 {
			arg = args[0]
			args = args[1:]

			err := bind.SetArg(arg, true)
			if err != nil {
				if isHardArgCountErr(err) {
					return nil, wrapExprError(set.expr.Name, err)
				}
			}
		}
		if err := bind.Done(); err != nil {
			return nil, wrapExprError(set.expr.Name, err)
		}
	}
	return results, nil
}

func (p *exprPipeline) applyFactory(c *Context, fac *exprPipelineFactory) error {
	exprs, err := fac.parse(c, p.args)
	p.items = exprs
	return err
}

func (b *exprBinding) Expr() *Expr {
	return b.expr
}

func (e *exprSynopsis) names() string {
	if len(e.long) == 0 {
		return fmt.Sprintf("-%s", e.short)
	}
	if len(e.short) == 0 {
		return fmt.Sprintf("-%s", e.long)
	}
	return fmt.Sprintf("-%s, -%s", e.short, e.long)
}

func (e *exprContext) initialize(c *Context) error {
	return executeAll(c, ActionOf(e.expr.Uses), nil)
}

func (e *exprContext) hooks() *hooks {
	return nil
}

func (e *exprContext) executeBefore(ctx *Context) error {
	return executeAll(ctx, ActionOf(e.expr.Before), defaultExpr.Before)
}

func (e *exprContext) executeAfter(ctx *Context) error {
	return executeAll(ctx, ActionOf(e.expr.After), defaultExpr.After)
}

func (e *exprContext) executeBeforeDescendent(ctx *Context) error { return nil }
func (e *exprContext) executeAfterDescendent(ctx *Context) error  { return nil }
func (e *exprContext) execute(ctx *Context) error                 { return nil }
func (e *exprContext) app() (*App, bool)                          { return nil, false }
func (e *exprContext) args() []string                             { return e.args_ }
func (e *exprContext) set() *set {
	if e.set_ == nil {
		e.set_ = newSet()
	}
	return e.set_
}
func (e *exprContext) setDidSubcommandExecute() {}
func (e *exprContext) target() target           { return e.expr }
func (e *exprContext) lookupValue(name string) (interface{}, bool) {
	return e.set_.lookupValue(name)
}
func (e *exprContext) Name() string { return e.expr.Name }

func emptyYielder(interface{}) error {
	return nil
}

func findExprByName(items []*Expr, name string) (*Expr, bool) {
	for _, sub := range items {
		if sub.Name == name {
			return sub, true
		}
	}
	return nil, false
}

var _ Expression = &exprPipeline{}
