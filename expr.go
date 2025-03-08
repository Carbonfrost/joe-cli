package cli

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

//counterfeiter:generate . Evaluator

// Evaluator provides the evaluation function for an expression operator.
type Evaluator interface {
	// Evaluate performs the evaluation.  The v argument is the value of the prior
	// expression operator.  The yield argument is used to pass one or more additional
	// values to the next expression operator.
	Evaluate(c context.Context, v interface{}, yield func(interface{}) error) error
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
	// text of placeholders.  The placeholder is used in the synopsis for the expression operator as well
	// as error messages.
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the expression operator.  If left blank, a succinct synopsis
	// is generated from the type of the expression operator's arguments
	UsageText string

	// Category specifies the expression operator category.  When categories are used, operators are grouped
	// together on the help screen
	Category string

	// Description provides a long description.  The long description is
	// not used in any templates by default.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description interface{}

	// Evaluate provides the evaluation behavior for the expression.  The value should
	// implement Evaluator or support runtime conversion to that interface via
	// the rules provided by the cli.EvaluatorOf function.
	Evaluate interface{}

	// Before executes before the expression is evaluated.  Refer to cli.Action about the correct
	// function signature to use.
	Before interface{}

	// After executes after the expression is evaluated.  Refer to cli.Action about the correct
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

	flags exprFlags
	*exprSet
}

// Expression provides the parsed result of the expression that can be evaluated
// with the given inputs.
type Expression struct {
	items []ExprBinding
	args  []string

	// Exprs identifies the expression operators that are allowed
	Exprs []*Expr
}

// ExprBinding provides the relationship between an evaluator and the evaluation
// context.  Optionally, the original Expr is made available.
// A binding can support being reset if it exposes a method Reset(). Resetting
// a binding occurs when an expression is evaluated multiple times.
type ExprBinding interface {
	Evaluator
	Lookup

	// Expr retrieves the expression operator if it is available
	Expr() *Expr
}

type exprsByCategory []*exprCategory

type exprCategory struct {
	Category string
	Exprs    []*Expr
}

type exprFlags int

type expressionDescription struct {
	exp   *Expression
	templ *Template
}

//counterfeiter:generate . Yielder

// Yielder provides the signature of the function used to yield
// values to the expression pipeline
type Yielder func(interface{}) error

type boundExpr struct {
	*exprSet
	expr *Expr
}

type exprBinding struct {
	Evaluator
	Lookup
	expr *Expr
}

type exprSet struct {
	Lookup
	Binding
	BindingMap
}

const (
	exprFlagHidden = exprFlags(1 << iota)
	exprFlagRightToLeft
)

func newBoundExpr(e *Expr) *boundExpr {
	set := newExprSet(NewBinding(nil, e.Args, nil), nil)
	return &boundExpr{
		expr:    e,
		exprSet: set,
	}
}

func newExprSet(b Binding, all BindingMap) *exprSet {
	result := &exprSet{
		BindingMap: all,
		Binding:    b,
	}
	result.Lookup = LookupFunc(result.lookupValue)
	return result
}

// Initializer is an action that binds expression handling to an argument.  This
// is set up automatically when a command defines any expression operators.  The exprFunc
// argument is used to determine which expressions to used.  If nil, the default behavior
// is used which is to lookup Command.Exprs from the context
func (e *Expression) Initializer() Action {
	return Pipeline(&Prototype{
		Name:      "expression",
		UsageText: "<expression>",
		NArg:      TakeRemaining,
	}, func(c *Context) error {
		return c.SetDescription(&expressionDescription{
			exp:   e,
			templ: c.Template("Expressions"),
		})

	}, func(c *Context) error {
		for _, sub := range e.Exprs {
			_ = c.ProvideValueInitializer(sub, sub.Name, Setup{
				Uses:   sub.Uses,
				Before: sub.Before,
				After:  sub.After,
			})
		}
		return nil

	}, At(ActionTiming, ActionFunc(func(c *Context) (err error) {
		var all BindingMap
		e.items, all, err = parseExpressions(e.Exprs, e.args)
		if err != nil {
			return
		}

		for _, sub := range e.Exprs {
			// Provide a view of the binding map that is global
			// to each of the Exprs
			es := newExprSet(NewBinding(nil, sub.Args, nil), all)
			sub.exprSet = es
		}

		for _, eb := range e.items {
			// For the expression bindings participating in the
			// pipeline, apply the binding result data. These
			tryResetIfSupported(eb)

			if known, ok := eb.(*boundExpr); ok {
				err = known.BindingMap().ApplyTo(known.exprSet)
				if err != nil {
					return
				}
			}
		}

		return
	})))
}

func (e *expressionDescription) SortUsage() {
	slices.SortFunc(e.exp.Exprs, exprsByNameOrder)
}

func (e *expressionDescription) String() string {
	var buf bytes.Buffer
	data := struct {
		Description *exprDescriptionData
	}{
		Description: exprDescription(e.exp),
	}
	err := e.templ.Execute(&buf, data)
	if err != nil {
		return err.Error()
	}
	return buf.String()
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
//   - func(*Context, interface{}, func(interface{})error) error
//   - func(*Context, interface{}) error
//   - func(*Context, interface{}) bool
//   - func(*Context, interface{})
//   - func(context.Context, interface{}, func(interface{})error) error
//   - func(context.Context, interface{}) error
//   - func(context.Context, interface{}) bool
//   - func(context.Context, interface{})
//   - func(interface{}, func(interface{})error) error
//   - func(interface{}) bool
//   - func(interface{}) error
//   - func(interface{})
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
			err := a(c, v)
			if err == nil {
				return y(v)
			}
			return err
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
	case func(context.Context, interface{}, func(interface{}) error) error:
		return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
			return a(c, v, y)
		})
	case func(context.Context, interface{}) error:
		return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
			err := a(c, v)
			if err == nil {
				return y(v)
			}
			return err
		})
	case func(context.Context, interface{}) bool:
		return EvaluatorFunc(func(c *Context, v interface{}, y func(interface{}) error) error {
			if a(c, v) {
				return y(v)
			}
			return nil
		})
	case func(context.Context, interface{}):
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
			err := a(v)
			if err == nil {
				return y(v)
			}
			return err
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
func Predicate(filter func(v any) bool) Evaluator {
	return EvaluatorFunc(func(_ *Context, v any, y func(any) error) error {
		if ok := filter(v); ok {
			return y(v)
		}
		return nil
	})
}

func groupExprsByCategory(exprs []*Expr) exprsByCategory {
	res := exprsByCategory{}
	all := map[string]*exprCategory{}
	category := func(name string) *exprCategory {
		if c, ok := all[name]; ok {
			return c
		}
		c := &exprCategory{Category: name, Exprs: []*Expr{}}
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

// Names obtains the name of the expression operator and its aliases
func (e *Expr) Names() []string {
	return append([]string{e.Name}, e.Aliases...)
}

// Synopsis retrieves the synopsis for the expression operator.
func (e *Expr) Synopsis() string {
	return sprintSynopsis(e.newSynopsis())
}

// Arg gets the expression operator by name
func (e *Expr) Arg(name interface{}) (*Arg, bool) {
	a, _, ok := findArgByName(e.Args, name)
	return a, ok
}

func (e *Expr) newSynopsis() *synopsis.Expr {
	args := make([]*synopsis.Arg, len(e.LocalArgs()))
	usage := synopsis.ParseUsage(e.HelpText)
	pp := usage.Placeholders()

	for i, a := range e.LocalArgs() {
		if i < len(pp) {
			args[i] = a.newSynopsis().WithUsage(pp[i])
		} else {
			// Use a simpler default name with less noise from angle brackets
			args[i] = a.newSynopsis().WithUsage(strings.ToUpper(a.Name))
		}
	}

	return synopsis.NewExpr(e.Name, e.Aliases, usage, args)
}

func (e *Expr) LocalArgs() []*Arg {
	return e.Args
}

// SetData sets the specified metadata on the expression operator
func (e *Expr) SetData(name string, v interface{}) {
	e.Data = setData(e.Data, name, v)
}

// LookupData obtains the data if it exists
func (e *Expr) LookupData(name string) (interface{}, bool) {
	v, ok := e.Data[name]
	return v, ok
}

func (e *Expr) SetName(name string) {
	e.Name = name
}

func (e *Expr) SetCategory(name string) {
	e.Category = name
}

func (e *Expr) SetHelpText(name string) {
	e.HelpText = name
}

func (e *Expr) SetManualText(name string) {
	e.ManualText = name
}

func (e *Expr) SetDescription(value string) {
	e.Description = value
}

func (e *Expr) internalFlags() exprFlags {
	return e.flags
}

// Evaluate provides the evaluation of the function and implements the Evaluator interface
func (e EvaluatorFunc) Evaluate(c context.Context, v interface{}, yield func(interface{}) error) error {
	if e == nil {
		return nil
	}
	return e(FromContext(c), v, yield)
}

func (f exprFlags) hidden() bool {
	return f&exprFlagHidden == exprFlagHidden
}

func (f exprFlags) rightToLeft() bool {
	return f&exprFlagRightToLeft == exprFlagRightToLeft
}

func (f exprFlags) toRaw() RawParseFlag {
	var flags RawParseFlag
	if f.rightToLeft() {
		flags |= RawRTL
	}
	return flags
}

// VisibleExprs filters all operators in the expression operators category by whether
// they are not hidden
func (e *exprCategory) VisibleExprs() []*Expr {
	res := make([]*Expr, 0, len(e.Exprs))
	for _, o := range e.Exprs {
		if o.flags.hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

// Undocumented determines whether the category is undocumented (i.e. has no HelpText set
// on any of its expression operators)
func (e *exprCategory) Undocumented() bool {
	for _, x := range e.Exprs {
		if x.HelpText != "" {
			return false
		}
	}
	return true
}

func (e exprsByCategory) Less(i, j int) bool {
	return e[i].Category < e[j].Category
}

func (e exprsByCategory) Len() int {
	return len(e)
}

func (e exprsByCategory) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (b *boundExpr) Expr() *Expr {
	return b.expr
}

func (b *boundExpr) LocalArgs() []*Arg {
	return b.expr.Args
}

func (b *boundExpr) BindingMap() BindingMap {
	return b.exprSet.BindingMap
}

func (b *boundExpr) Reset() {
	b.exprSet.Binding.Reset()
}

func (b *boundExpr) Evaluate(c context.Context, v any, yield func(any) error) error {
	ctx := FromContext(c).ValueContextOf(b.Expr().Name, b)
	tryResetIfSupported(b)

	err := b.BindingMap().ApplyTo(b.exprSet)
	if err != nil {
		return err
	}

	return EvaluatorOf(b.expr.Evaluate).Evaluate(ctx, v, yield)
}

func tryResetIfSupported(v any) {
	if r, ok := v.(interface{ Reset() }); ok {
		r.Reset()
	}
}

func (e *Expression) Set(arg string) error {
	if e.args == nil {
		e.args = make([]string, 0)
	}
	e.args = append(e.args, arg)
	return nil
}

func (e *Expression) String() string {
	return strings.Join(e.args, " ")
}

func (e *Expression) Evaluate(ctx context.Context, items ...interface{}) error {
	return e.evaluateCore(ctx, items...)
}

func (e *Expression) evaluateCore(ctx context.Context, items ...interface{}) error {
	yielders := make([]Yielder, len(e.items))
	yielderThunk := func(i int) Yielder {
		if i >= len(yielders) || yielders[i] == nil {
			return emptyYielder
		}
		return yielders[i]
	}

	for ik := range e.items {
		i := ik
		yielders[i] = func(in interface{}) error {
			return e.items[i].Evaluate(ctx, in, yielderThunk(i+1))
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

func (e *Expression) Walk(fn func(ExprBinding) error) error {
	if fn == nil {
		return nil
	}
	for _, x := range e.items {
		err := fn(x)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Expression) Prepend(expr ExprBinding) {
	e.items = append([]ExprBinding{expr}, e.items...)
}

func (e *Expression) Append(expr ExprBinding) {
	e.items = append(e.items, expr)
}

// VisibleExprs filters all expression operators by whether they are not hidden
func (e *Expression) VisibleExprs() []*Expr {
	res := make([]*Expr, 0, len(e.Exprs))
	for _, o := range e.Exprs {
		if o.internalFlags().hidden() {
			continue
		}
		res = append(res, o)
	}
	return res
}

func parseExpressions(exprOperands []*Expr, args []string) ([]ExprBinding, BindingMap, error) {
	exprs := map[string]*Expr{}
	for _, e := range exprOperands {
		exprs[e.Name] = e
		for _, alias := range e.Aliases {
			exprs[alias] = e
		}
	}

	results := make([]ExprBinding, 0)
	all := BindingMap{}
	for len(args) > 0 {
		arg := args[0]
		args = args[1:]

		expr, ok := exprs[arg[1:]]
		if !ok {
			return nil, nil, unknownExpr(arg)
		}

		// Copy to a "bound expression" to create instancing for the
		// use of the expression operator.
		boundExpr := newBoundExpr(expr)
		results = append(results, boundExpr)
		bin, err := RawParse(args, boundExpr.exprSet, boundExpr.expr.internalFlags().toRaw())

		var pe *ParseError
		if err != nil {
			pe = err.(*ParseError)
			args = pe.Remaining

			switch pe.Code {
			case UnexpectedArgument:
				return nil, nil, argsMustPrecedeExprs(args[0])
			case ExpectedArgument:
				return nil, nil, pe
			}
		}

		// Update the bound expression with the data which was copied,
		// and collect it within a global view across the whole pipeline
		boundExpr.exprSet.BindingMap = bin
		for k, v := range bin {
			all[k] = append(all[k], v...)
		}

		// If the parse completed successfully, there is nothing else to do
		if pe == nil {
			break
		}
	}
	return results, all, nil
}

func (b *exprBinding) Expr() *Expr {
	return b.expr
}

func (s *exprSet) lookupValue(name string) (interface{}, bool) {
	if _, _, g, ok := s.Binding.LookupOption(name); ok {
		return g.(option).value(), true
	}
	return nil, false
}

func emptyYielder(interface{}) error {
	return nil
}

func exprsByNameOrder(x *Expr, y *Expr) int {
	return cmp.Compare(x.Name, y.Name)
}

var (
	_ Value         = (*Expression)(nil)
	_ BindingLookup = (*boundExpr)(nil)
	_ BindingLookup = (*Expr)(nil)
	_ Binding       = (*boundExpr)(nil)
)
