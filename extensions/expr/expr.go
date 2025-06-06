// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package expr

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/synopsis"
)

//counterfeiter:generate . Evaluator

// Evaluator provides the evaluation function for an expression operator.
type Evaluator interface {
	// Evaluate performs the evaluation.  The v argument is the value of the prior
	// expression operator.  The yield argument is used to pass one or more additional
	// values to the next expression operator.
	Evaluate(c context.Context, v any, yield func(any) error) error
}

// EvaluatorFunc provides the basic function for an Evaluator
type EvaluatorFunc func(*cli.Context, any, func(any) error) error

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
	Args []*cli.Arg

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
	Description any

	// Evaluate provides the evaluation behavior for the expression.  The value should
	// implement Evaluator or support runtime conversion to that interface via
	// the rules provided by the cli.EvaluatorOf function.
	Evaluate any

	// Before executes before the expression is evaluated.  Refer to cli.Action about the correct
	// function signature to use.
	Before any

	// After executes after the expression is evaluated.  Refer to cli.Action about the correct
	// function signature to use.
	After any

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	Uses any

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]any

	// Options sets various options about how to treat the expression operator.  For example, options can
	// hide the expression operator.
	Options cli.Option

	flags exprFlags
	*exprSet
}

// Expression provides the parsed result of the expression that can be evaluated
// with the given inputs.
type Expression struct {
	items []Binding
	args  []string

	// Exprs identifies the expression operators that are allowed
	Exprs []*Expr
}

// Binding provides the relationship between an evaluator and the evaluation
// context.  Optionally, the original Expr is made available.
// A binding can support being reset if it exposes a method Reset(). Resetting
// a binding occurs when an expression is evaluated multiple times.
type Binding interface {
	Evaluator
	cli.Lookup

	// Expr retrieves the expression operator if it is available
	Expr() *Expr
}

// Predicate provides a simple predicate which filters values.  The function
// takes the prior operand and returns true or false depending upon whether the
// operand should be yielded to the next step in the expression pipeline.
type Predicate func(v any) bool

// Invariant provides an evaluator which either always or never yields
// the input value
type Invariant bool

type composite []Evaluator

// Always or never yield the input value
const (
	AlwaysTrue  = Invariant(true)
	AlwaysFalse = Invariant(false)
)

type exprsByCategory []*exprCategory

type exprCategory struct {
	Category string
	Exprs    []*Expr
}

type exprFlags int

type expressionDescription struct {
	exp   *Expression
	templ *cli.Template
}

type evaluatorFunc func(context.Context, any, func(any) error) error

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Yielder

// Yielder provides the signature of the function used to yield
// values to the expression pipeline
type Yielder func(any) error

type boundExpr struct {
	*exprSet
	expr *Expr
}

type exprBinding struct {
	Evaluator
	cli.Lookup
	expr *Expr
}

type exprSet struct {
	cli.Lookup
	cli.Binding
	cli.BindingMap
}

const (
	exprFlagHidden = exprFlags(1 << iota)
	exprFlagRightToLeft
)

var (
	errStopWalk            = errors.New("stop walking")
	validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9@#+\._\*:-]+$`)
)

func newBoundExpr(e *Expr) *boundExpr {
	set := newExprSet(cli.NewBinding(nil, e.Args, nil), nil)
	return &boundExpr{
		expr:    e,
		exprSet: set,
	}
}

func newExprSet(b cli.Binding, all cli.BindingMap) *exprSet {
	result := &exprSet{
		BindingMap: all,
		Binding:    b,
	}
	result.Lookup = cli.LookupFunc(result.lookupValue)
	return result
}

// Initializer is an action that binds expression handling to an argument.  This
// is set up automatically when a command defines any expression operators.  The exprFunc
// argument is used to determine which expressions to used.  If nil, the default behavior
// is used which is to lookup Command.Exprs from the context
func (e *Expression) Initializer() cli.Action {
	return cli.Pipeline(&cli.Prototype{
		Name:      "expression",
		UsageText: "<expression>",
		NArg:      cli.TakeRemaining,
	}, func(c *cli.Context) error {
		return c.SetDescription(&expressionDescription{
			exp:   e,
			templ: c.Template("Expressions"),
		})

	}, func(c *cli.Context) error {
		for _, sub := range e.Exprs {
			// As a special case, the evaluator can implement Action
			// and be treated as part of the initialization pipeline.
			// (One case of this is in bind.Evaluator)
			var evalAsAction cli.Action
			if e, ok := sub.Evaluate.(cli.Action); ok {
				evalAsAction = e
			}

			_ = c.ProvideValueInitializer(sub, sub.Name, cli.Setup{
				Uses:   cli.Pipeline(sub.Uses, evalAsAction, sub.Options),
				Before: sub.Before,
				After:  sub.After,
			})
		}

		return finalizeExprs(e)

	}, cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) (err error) {
		var all cli.BindingMap
		e.items, all, err = parseExpressions(e.Exprs, e.args)
		if err != nil {
			return
		}

		for _, sub := range e.Exprs {
			// Provide a view of the binding map that is global
			// to each of the Exprs
			es := newExprSet(cli.NewBinding(nil, sub.Args, nil), all)
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

// NewBinding creates an expression binding.  The ev parameter is how
// the expression is evaluated.  The remaining arguments specify the *Expr
// expression operator to use and optionally a Lookup.   The main use case
// for this function is to create a custom evaluation step that is appended to
// the expression pipeline
func NewBinding(ev Evaluator, exprlookup ...any) Binding {
	var (
		expr   *Expr
		lookup cli.Lookup = cli.LookupValues{}
	)
	switch len(exprlookup) {
	case 2:
		lookup = exprlookup[1].(cli.Lookup)
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
// includes all values).  If an error value, then it always returns such an error.
// If a function, the signature must match either the
// Evaluator.Evaluate function signature or a  variation that excludes
// the context and/or yielder.
// You can also use bool as a return type as in the same signature used by
// Predicate.  These are valid signatures:
//
//   - func(*cli.Context, any, func(any)error) error
//   - func(*cli.Context, any) error
//   - func(*cli.Context, any) bool
//   - func(*cli.Context, any)
//   - func(context.Context, any, func(any)error) error
//   - func(context.Context, any) error
//   - func(context.Context, any) bool
//   - func(context.Context, any)
//   - func(any, func(any)error) error
//   - func(any) bool
//   - func(any) error
//   - func(any)
func EvaluatorOf(v any) Evaluator {
	switch a := v.(type) {
	case nil:
		return EvaluatorOf(true)
	case Evaluator:
		return a
	case func(*cli.Context, any, func(any) error) error:
		return EvaluatorFunc(a)
	case func(*cli.Context, any) error:
		return EvaluatorFunc(func(c *cli.Context, v any, y func(any) error) error {
			err := a(c, v)
			if err == nil {
				return y(v)
			}
			return err
		})
	case func(*cli.Context, any) bool:
		return EvaluatorFunc(func(c *cli.Context, v any, y func(any) error) error {
			if a(c, v) {
				return y(v)
			}
			return nil
		})
	case func(*cli.Context, any):
		return EvaluatorFunc(func(c *cli.Context, v any, y func(any) error) error {
			a(c, v)
			return y(v)
		})
	case func(context.Context, any, func(any) error) error:
		return evaluatorFunc(func(c context.Context, v any, y func(any) error) error {
			return a(c, v, y)
		})
	case func(context.Context, any) error:
		return evaluatorFunc(func(c context.Context, v any, y func(any) error) error {
			err := a(c, v)
			if err == nil {
				return y(v)
			}
			return err
		})
	case func(context.Context, any) bool:
		return evaluatorFunc(func(c context.Context, v any, y func(any) error) error {
			if a(c, v) {
				return y(v)
			}
			return nil
		})
	case func(context.Context, any):
		return evaluatorFunc(func(c context.Context, v any, y func(any) error) error {
			a(c, v)
			return y(v)
		})
	case func(any, func(any) error) error:
		return evaluatorFunc(func(_ context.Context, v any, y func(any) error) error {
			return a(v, y)
		})
	case func(any) error:
		return evaluatorFunc(func(_ context.Context, v any, y func(any) error) error {
			err := a(v)
			if err == nil {
				return y(v)
			}
			return err
		})
	case func(any) bool:
		return Predicate(a)

	case func(any):
		return Predicate(func(v any) bool {
			a(v)
			return true
		})
	case func() bool:
		return Predicate(func(any) bool {
			return a()
		})
	case func() error:
		return evaluatorFunc(func(context.Context, any, func(any) error) error {
			return a()
		})
	case bool:
		return Invariant(a)
	case error:
		return Error(a)
	}
	panic(fmt.Sprintf("unexpected type: %T", v))
}

// Evaluate implements the Evaluator interface for Predicate
func (p Predicate) Evaluate(c context.Context, v any, yield func(any) error) error {
	if ok := p(v); ok {
		return yield(v)
	}
	return nil
}

func (i Invariant) Evaluate(_ context.Context, v any, y func(any) error) error {
	if i {
		return y(v)
	}
	return nil
}

// ComposeEvaluator produces an evaluator which considers each evaluator in
// turn. If any evaluator yields the value, evaluation stops. If any evaluator
// returns an error, the evaluation stops and returns the error. This evaluator
// can be thought of as a logical conjunction in the case where evaluators
// work like Boolean predicates. Indeed, [Predicate] is often the type of the evaluators
// passed this function.
func ComposeEvaluator(e ...Evaluator) Evaluator {
	return composite(e)
}

// Error provides an evaluator which yields an error. The zero value
// is also useful, returning a generic error
func Error(err error) Evaluator {
	return EvaluatorOf(func(_ context.Context, v any, _ func(any) error) error {
		if err == nil {
			return fmt.Errorf("unsupported value: %T", v)
		}
		return err
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
	buf := cli.NewBuffer()
	buf.SetColorCapable(false)
	e.newSynopsis().WriteTo(buf)
	return buf.String()
}

// Arg gets the expression operator by name
func (e *Expr) Arg(name any) (*cli.Arg, bool) {
	// Somewhat hackish to use the command for this but this
	// spares having to make the API exported
	cmd := &cli.Command{Args: e.Args}
	return cmd.Arg(name)
}

func (e *Expr) newSynopsis() *synopsis.Expr {
	args := make([]*synopsis.Arg, len(e.LocalArgs()))
	usage := synopsis.ParseUsage(e.HelpText)
	pp := usage.Placeholders()

	for i, a := range e.LocalArgs() {
		if i < len(pp) {
			args[i] = argSynopsis(a, pp[i])
		} else {
			// Use a simpler default name with less noise from angle brackets
			args[i] = argSynopsis(a, strings.ToUpper(a.Name))
		}
	}

	return synopsis.NewExpr(e.Name, e.Aliases, usage, args)
}

func argSynopsis(a *cli.Arg, name string) *synopsis.Arg {
	return synopsis.NewArg(name, a.NArg)
}

// LocalArgs retrieves the arguments
func (e *Expr) LocalArgs() []*cli.Arg {
	return e.Args
}

// SetLocalArgs sets arguments
func (e *Expr) SetLocalArgs(args []*cli.Arg) error {
	e.Args = args
	return nil
}

// SetData sets the specified metadata on the expression operator
func (e *Expr) SetData(name string, v any) {
	e.Data = setData(e.Data, name, v)
}

// LookupData obtains the data if it exists
func (e *Expr) LookupData(name string) (any, bool) {
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

func (e *Expr) SetUsageText(value string) {
	e.UsageText = value
}

func (e *Expr) SetManualText(name string) {
	e.ManualText = name
}

func (e *Expr) SetDescription(value string) {
	e.Description = value
}

func (e *Expr) SetHidden(value bool) {
	e.setInternalFlags(exprFlagHidden, value)
}

func (e *Expr) SetAliases(a []string) {
	e.Aliases = append(e.Aliases, a...)
}

func (e *Expr) internalFlags() exprFlags {
	return e.flags
}

func (e *Expr) setInternalFlags(f exprFlags, v bool) {
	if v {
		e.flags |= f
	} else {
		e.flags &= ^f
	}
}

// Evaluate provides the evaluation of the function and implements the Evaluator interface
func (e EvaluatorFunc) Evaluate(c context.Context, v any, yield func(any) error) error {
	if e == nil {
		return nil
	}
	return e(cli.FromContext(c), v, yield)
}

func (e evaluatorFunc) Evaluate(c context.Context, v any, yield func(any) error) error {
	if e == nil {
		return nil
	}
	return e(c, v, yield)
}

func (c composite) Evaluate(ctx context.Context, v any, yield func(any) error) error {
	if yield == nil {
		yield = emptyYielder
	}
	var yielded bool
	yieldWrapper := func(any) error {
		err := yield(v)
		yielded = true
		return err
	}

	for _, e := range c {
		err := e.Evaluate(ctx, v, yieldWrapper)
		if err != nil {
			return err
		}
		if yielded {
			yielded = false
			break
		}
	}
	return nil
}

func (f exprFlags) hidden() bool {
	return f&exprFlagHidden == exprFlagHidden
}

func (f exprFlags) rightToLeft() bool {
	return f&exprFlagRightToLeft == exprFlagRightToLeft
}

func (f exprFlags) toRaw() cli.RawParseFlag {
	var flags cli.RawParseFlag
	if f.rightToLeft() {
		flags |= cli.RawRTL
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

func (b *boundExpr) LocalArgs() []*cli.Arg {
	return b.expr.Args
}

func (b *boundExpr) BindingMap() cli.BindingMap {
	return b.exprSet.BindingMap
}

func (b *boundExpr) Reset() {
	b.exprSet.Binding.Reset()
}

func (b *boundExpr) Evaluate(c context.Context, v any, yield func(any) error) error {
	ctx := cli.FromContext(c).ValueContextOf(b.Expr().Name, b)
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

func (e *Expression) Evaluate(ctx context.Context, items ...any) error {
	return e.evaluateCore(ctx, items...)
}

func (e *Expression) evaluateCore(ctx context.Context, items ...any) error {
	yielders := make([]Yielder, len(e.items))
	yielderThunk := func(i int) Yielder {
		if i >= len(yielders) || yielders[i] == nil {
			return emptyYielder
		}
		return yielders[i]
	}

	for ik := range e.items {
		i := ik
		yielders[i] = func(in any) error {
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

// Bindings enumerates all the bindings on the expression
func (e *Expression) Bindings() iter.Seq[Binding] {
	return func(yield func(Binding) bool) {
		e.Walk(func(f Binding) error {
			if !yield(f) {
				return errStopWalk
			}
			return nil
		})
	}
}

func (e *Expression) Walk(fn func(Binding) error) error {
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

func (e *Expression) Prepend(expr Binding) {
	e.items = append([]Binding{expr}, e.items...)
}

func (e *Expression) Append(expr Binding) {
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

func parseExpressions(exprOperands []*Expr, args []string) ([]Binding, cli.BindingMap, error) {
	exprs := map[string]*Expr{}
	for _, e := range exprOperands {
		exprs[e.Name] = e
		for _, alias := range e.Aliases {
			exprs[alias] = e
		}
	}

	results := make([]Binding, 0)
	all := cli.BindingMap{}
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
		bin, err := cli.RawParse(args, boundExpr.exprSet, boundExpr.expr.internalFlags().toRaw())

		var pe *cli.ParseError
		if err != nil {
			pe = err.(*cli.ParseError)
			args = pe.Remaining

			switch pe.Code {
			case cli.UnexpectedArgument:
				return nil, nil, argsMustPrecedeExprs(args[0])
			case cli.ExpectedArgument:
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

func (s *exprSet) lookupValue(name string) (any, bool) {
	if _, _, g, ok := s.Binding.LookupOption(name); ok {
		return g.(*cli.Arg).Value, true
	}
	return nil, false
}

func emptyYielder(any) error {
	return nil
}

func exprsByNameOrder(x *Expr, y *Expr) int {
	return cmp.Compare(x.Name, y.Name)
}

func unknownExpr(name string) error {
	return &cli.ParseError{
		Code: cli.UnknownExpr,
		Name: name,
		Err:  fmt.Errorf("unknown expression: %s", name),
	}
}

func argsMustPrecedeExprs(arg string) error {
	return &cli.ParseError{
		Code:  cli.ArgsMustPrecedeExprs,
		Value: arg,
		Err:   fmt.Errorf("arguments must precede expressions: %q", arg),
	}
}

func renderHelp(us *synopsis.Usage) string {
	sb := cli.NewBuffer()
	us.HelpText(sb)
	return sb.String()
}

func setData(data map[string]any, name string, v any) map[string]any {
	if v == nil {
		delete(data, name)
		return data
	}
	if data == nil {
		return map[string]any{
			name: v,
		}
	}
	data[name] = v
	return data
}

func finalizeExprs(e *Expression) error {
	// Check for duplicative and invalid names of expressions
	names := map[string]bool{}
	var errs []error

	for i, e := range e.Exprs {
		if e.Name == "" {
			errs = append(errs, fmt.Errorf("expr at index #%d must have a name", i))
			continue
		}
		if err := checkValidIdentifier(e.Name); err != nil {
			errs = append(errs, err)
		} else if names[e.Name] {
			errs = append(errs, fmt.Errorf("duplicate name used: %q", e.Name))
		}
		names[e.Name] = true
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors initializing expression: %w", errors.Join(errs...))
	}
	return nil
}

func checkValidIdentifier(name string) error {
	if !validIdentifierPattern.MatchString(name) {
		return fmt.Errorf("not a valid name")
	}
	return nil
}

var (
	_ cli.Value         = (*Expression)(nil)
	_ cli.BindingLookup = (*boundExpr)(nil)
	_ cli.BindingLookup = (*Expr)(nil)
	_ cli.Binding       = (*boundExpr)(nil)
)
