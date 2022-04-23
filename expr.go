package cli

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/juju/ansiterm"
	"github.com/juju/ansiterm/tabwriter"
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
	pipelinesSupport
	customizableSupport

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

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the expression operator.  If left blank, a succint synopsis
	// is generated from the type of the expression operator's arguments
	UsageText string

	// Category specifies the expression operator category.  When categories are used, operators are grouped
	// together on the help screen
	Category string

	// Description provides a long description.  The long description is
	// not used in any templates by default
	Description string

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

	flags internalFlags
}

// Expression provides the parsed result of the expression that can be evaluated
// with the given inputs.
type Expression struct {
	boundContext *Context // where the expr pipeline was bound (should be "<expression>" argument)
	items        []ExprBinding
	args         []string

	// Exprs identifies the expression operators that are allowed
	Exprs []*Expr
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
	exprs map[string]*boundExpr
}

type exprSynopsis struct {
	Long         string
	Short        string
	usage        *usage
	Names        []string
	Args         []*argSynopsis
	RequiredArgs []*argSynopsis
	OptionalArgs []*argSynopsis
}

type boundExpr struct {
	Lookup
	expr *Expr
	set  *set
}

type exprBinding struct {
	Evaluator
	Lookup
	expr *Expr
}

// BindExpression is an action that binds expression handling to an argument.  This
// is set up automatically when a command defines any expression operators.  The exprFunc
// argument is used to determine which expressions to used.  If nil, the default behavior
// is used which is to lookup Command.Exprs from the context
func (e *Expression) Initializer() Action {
	return ActionOf(func(c *Context) error {
		arg := c.Arg()
		if arg.Name == "" {
			arg.Name = "expression"
		}
		arg.UsageText = "<expression>"
		arg.NArg = -1
		arg.Action = Pipeline(arg.Action, func(c *Context) error {
			pipe := c.Value("").(*Expression)
			fac := newExprPipelineFactory(e.Exprs)
			return pipe.applyFactory(c, fac)
		})
		arg.Before = Pipeline(arg.Before, func(c *Context) {
			// Provide a more up-to-date description if Exprs changed
			c.Arg().Description = e.renderDescription(c)
		})

		c.Arg().Description = e.renderDescription(c)
		e.boundContext = c

		for _, sub := range e.Exprs {
			err := c.ProvideValueInitializer(sub, sub.Name, Setup{
				Uses:   Pipeline(sub.Uses, initializeFlagsArgs),
				Before: Pipeline(sub.Before, triggerBeforeArgs),
				After:  Pipeline(sub.After, triggerAfterArgs),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *Expression) renderDescription(c *Context) string {
	var buf bytes.Buffer
	tpl := c.Template("Expressions")

	data := struct {
		Description *exprDescriptionData
		Debug       bool
	}{
		Description: exprDescription(e),
		Debug:       tpl.Debug,
	}

	w := ansiterm.NewTabWriter(&buf, 1, 8, 2, ' ', tabwriter.StripEscape)
	_ = tpl.Execute(w, data)
	_ = w.Flush()
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
		exprs: map[string]*boundExpr{},
	}
	for _, e := range exprs {
		set := newSet(e.internalFlags().rightToLeft()).withArgs(e.Args)
		fac := &boundExpr{
			expr:   e,
			set:    set,
			Lookup: set,
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
	return sprintSynopsis(e, false)
}

func (e *Expr) WriteSynopsis(w Writer) {
	synopsisTemplate.ExecuteTemplate(w, "ExpressionSynopsis", e.newSynopsis())
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
	names := func() []string {
		if len(long) == 0 {
			return []string{fmt.Sprintf("-%s", string(short[0]))}
		}
		if len(short) == 0 {
			return []string{fmt.Sprintf("-%s", long[0])}
		}
		return []string{fmt.Sprintf("-%s", string(short[0])), fmt.Sprintf("-%s", string(long[0]))}
	}

	return &exprSynopsis{
		Long:         longName(long),
		Short:        shortName(short),
		usage:        usage,
		Args:         args,
		RequiredArgs: args,
		Names:        names(),
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

// LookupData obtains the data if it exists
func (e *Expr) LookupData(name string) (interface{}, bool) {
	v, ok := e.ensureData()[name]
	return v, ok
}

func (e *Expr) setCategory(name string) {
	e.Category = name
}

func (e *Expr) setHelpText(name string) {
	e.HelpText = name
}

func (e *Expr) setManualText(name string) {
	e.ManualText = name
}

func (e *Expr) setDescription(value string) {
	e.Description = value
}

func (e *Expr) ensureData() map[string]interface{} {
	if e.Data == nil {
		e.Data = map[string]interface{}{}
	}
	return e.Data
}

func (e *Expr) setInternalFlags(f internalFlags) {
	e.flags |= f
}

func (e *Expr) internalFlags() internalFlags {
	return e.flags
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
func (e *ExprCategory) VisibleExprs() []*Expr {
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
	// TODO Pass along args
	args := []string{}
	ctx := c.exprContext(b.expr, args, b.set).setTiming(ActionTiming)
	return EvaluatorOf(b.expr.Evaluate).Evaluate(ctx, v, yield)
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

func (e *Expression) Evaluate(ctx *Context, items ...interface{}) error {
	return e.evaluateCore(ctx, items...)
}

func (e *Expression) evaluateCore(ctx *Context, items ...interface{}) error {
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

	ParseExprFlag:
		boundExpr, ok := e.exprs[arg[1:]]
		if !ok {
			return nil, unknownExpr(arg)
		}

		results = append(results, boundExpr)
		if len(boundExpr.set.positionalOptions) == 0 {
			continue Parsing
		}

		bind, err := boundExpr.set.startArgBinding(len(args))
		if err != nil {
			return nil, err
		}
		for len(args) > 0 {
			arg = args[0]
			args = args[1:]

			err := bind.SetArg(arg, true)
			if err != nil {
				if isNextExpr(arg, err) {
					goto ParseExprFlag
				}
				if isHardArgCountErr(err) {
					return nil, wrapExprError(boundExpr.expr.Name, err)
				}
			}
			if !bind.hasCurrent() {
				break
			}
		}
		if err := bind.Done(); err != nil {
			return nil, wrapExprError(boundExpr.expr.Name, err)
		}
	}
	return results, nil
}

func (e *Expression) applyFactory(c *Context, fac *exprPipelineFactory) error {
	exprs, err := fac.parse(c, e.args)
	e.items = exprs
	return err
}

func (b *exprBinding) Expr() *Expr {
	return b.expr
}

func emptyYielder(interface{}) error {
	return nil
}

var _ Value = (*Expression)(nil)
var _ targetConventions = (*Expr)(nil)
