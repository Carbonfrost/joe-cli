package cli

import (
	"context"
	"fmt"
)

// ActionFunc provides the basic function for
type ActionFunc func(*Context) error

//counterfeiter:generate . Action

// Action represents the building block of the various actions
// to perform when an app, command, flag, or argument is being evaluated.
type Action interface {
	Execute(*Context) error
}

type ActionPipeline struct {
	items []Action
}

type target interface {
	hooks() *hooks
	options() Option
	appendAction(timing, Action)
	setCategory(name string)
	SetData(name string, v interface{})
	setInternalFlags(internalFlags)
	internalFlags() internalFlags
}

type actionPipelines struct {
	Initializers Action // Must be strictly initializers (no automatic regrouping)
	Before       Action
	Action       Action
	After        Action
}

type ActionTiming interface {
	Action
	timing() timing
}

type withTimingWrapper struct {
	Action
	t timing
}

type timing int

const (
	initialTiming timing = iota
	beforeTiming
	actionTiming
	afterTiming
)

var (
	emptyAction Action = ActionFunc(emptyActionImpl)

	defaultApp = actionPipelines{
		Initializers: Pipeline(
			ActionFunc(setupDefaultIO),
			ActionFunc(setupDefaultData),
			ActionFunc(addAppCommand("help", defaultHelpFlag(), defaultHelpCommand())),
			ActionFunc(addAppCommand("version", defaultVersionFlag(), defaultVersionCommand())),
		),
	}

	defaultCommand = actionPipelines{
		Before: Pipeline(
			ActionFunc(triggerFlagsAndArgs),
		),
		After: Pipeline(
			ActionFunc(triggerAfterFlagsAndArgs),
		),
	}

	defaultOption = actionPipelines{
		Before: Pipeline(
			ActionFunc(setupOptionRequireFS),
			ActionFunc(setupOptionFromEnv),
		),
	}

	defaultExpr = actionPipelines{}
)

// Pipeline combines various actions into a single action
func Pipeline(actions ...interface{}) *ActionPipeline {
	myActions := make([]Action, len(actions))
	for i, a := range actions {
		myActions[i] = ActionOf(a)
	}

	return &ActionPipeline{myActions}
}

func Before(a Action) Action {
	return withTiming(a, beforeTiming)
}

func After(a Action) Action {
	return withTiming(a, afterTiming)
}

// Initializer marks an action handler as being for the initialization phase.  When such a handler
// is added to the Uses pipeline, it will automatically be associated correctly with the initialization
// of the value.  Otherwise, this handler is not special
func Initializer(a Action) Action {
	return withTiming(a, initialTiming)
}

func timingOf(a Action, defaultTiming timing) timing {
	switch val := a.(type) {
	case ActionTiming:
		return val.timing()
	}
	return defaultTiming
}

func withTiming(a Action, t timing) Action {
	return withTimingWrapper{a, t}
}

func ActionOf(item interface{}) Action {
	switch a := item.(type) {
	case nil:
		return nil
	case func(*Context) error:
		return ActionFunc(a)
	case Action:
		return a
	case func(*Context):
		return ActionFunc(func(c *Context) error {
			a(c)
			return nil
		})
	case func(context.Context) error:
		return ActionFunc(func(c *Context) error {
			return a(c.Context)
		})
	case func(context.Context):
		return ActionFunc(func(c *Context) error {
			a(c.Context)
			return nil
		})
	case func() error:
		return ActionFunc(func(*Context) error {
			return a()
		})
	case func():
		return ActionFunc(func(*Context) error {
			a()
			return nil
		})
	}
	panic(fmt.Sprintf("unexpected type: %T", item))
}

func ContextValue(key, value interface{}) ActionFunc {
	return func(c *Context) error {
		c.Context = context.WithValue(c.Context, key, value)
		return nil
	}
}

func SetValue(v interface{}) ActionFunc {
	return func(c *Context) error {
		c.target().(option).Set(genericString(dereference(v)))
		return nil
	}
}

// Data sets metadata for a command, flag, arg, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Data(name string, value interface{}) Action {
	return ActionFunc(func(c *Context) error {
		c.target().SetData(name, value)
		return nil
	})
}

// Category sets the category of a command, flag, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Category(name string) Action {
	return ActionFunc(func(c *Context) error {
		c.target().setCategory(name)
		return nil
	})
}

// HookBefore registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookBefore(pattern string, handler Action) Action {
	return ActionFunc(func(c *Context) error {
		c.demandInit().hookBefore(pattern, handler)
		return nil
	})
}

// HookAfter registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookAfter(pattern string, handler Action) Action {
	return ActionFunc(func(c *Context) error {
		c.demandInit().hookAfter(pattern, handler)
		return nil
	})
}

// OptionalValue makes the flag's value optional, and when its value is not specified, the implied value
// is set to this value v.  Say that a flag is defined as:
//
//   &Flag {
//     Name: "secure",
//     Value: cli.String(),
//     Uses: cli.Optional("TLS1.2"),
//   }
//
// This example implies that --secure without a value is set to the value TLS1.2 (presumably other versions
// are allowed).  This example is a fair use case of this feature: making a flag opt-in to some sort of default
// configuration and allowing an expert configuration by using a value.
// In general, making the value of a non-Boolean flag optional is not recommended when
// the command also allows arguments because it can make the syntax ambiguous.
func OptionalValue(v interface{}) Action {
	return Initializer(ActionFunc(func(c *Context) error {
		c.Flag().setOptionalValue(v)
		return nil
	}))
}

func newActionPipelines(m map[timing][]Action) *actionPipelines {
	var pipe = func(h []Action) Action {
		if len(h) == 0 {
			return emptyAction
		}
		return &ActionPipeline{h}
	}
	return &actionPipelines{
		Initializers: pipe(m[initialTiming]),
		Before:       pipe(m[beforeTiming]),
		Action:       pipe(m[actionTiming]),
		After:        pipe(m[afterTiming]),
	}
}

func (af ActionFunc) Execute(c *Context) error {
	if af == nil {
		return nil
	}
	return af(c)
}

func (p *ActionPipeline) Append(x Action) *ActionPipeline {
	return &ActionPipeline{
		items: append(p.items, unwind(x)...),
	}
}

func (p *ActionPipeline) Execute(c *Context) (err error) {
	if p == nil {
		return nil
	}
	for _, a := range p.items {
		err = a.Execute(c)
		if err != nil {
			return
		}
	}
	return nil
}

func (p *ActionPipeline) takeInitializers(c *Context) (*actionPipelines, error) {
	res := &actionPipelines{}
	for _, h := range p.items {
		res.add(timingOf(h, initialTiming), h)
	}

	return res, nil
}

func (p *actionPipelines) add(t timing, h Action) {
	switch t {
	case initialTiming:
		p.Initializers = pipeline(p.Initializers, h)
	case beforeTiming:
		p.Before = pipeline(p.Before, h)
	case actionTiming:
		p.Action = pipeline(p.Action, h)
	case afterTiming:
		p.After = pipeline(p.After, h)
	default:
		panic("unreachable!")
	}
}

func (p *actionPipelines) exceptInitializers() *actionPipelines {
	return &actionPipelines{
		Before: p.Before,
		Action: p.Action,
		After:  p.After,
	}
}

func (w withTimingWrapper) timing() timing {
	return w.t
}

func emptyActionImpl(*Context) error {
	return nil
}

func execute(af Action, c *Context) error {
	if af == nil {
		return nil
	}
	return af.Execute(c)
}

func executeAll(c *Context, x ...Action) error {
	for _, y := range x {
		if err := execute(y, c); err != nil {
			return err
		}
	}
	return nil
}

func doThenExit(a Action) ActionFunc {
	return func(c *Context) error {
		err := a.Execute(c)
		if err != nil {
			return err
		}
		return Exit(0)
	}
}

func pipeline(x, y Action) *ActionPipeline {
	return &ActionPipeline{
		items: append(unwind(x), unwind(y)...),
	}
}

func takeInitializers(uses Action, opts Option, c *Context) (*actionPipelines, error) {
	return pipeline(uses, opts.wrap()).takeInitializers(c)
}

func unwind(x Action) []Action {
	if x == nil {
		return nil
	}
	switch pipe := x.(type) {
	case *ActionPipeline:
		if pipe == nil {
			return nil
		}
		res := make([]Action, 0, len(pipe.items))
		for _, p := range pipe.items {
			res = append(res, unwind(p)...)
		}
		return res
	}
	return []Action{x}
}

func actionOrEmpty(v interface{}) Action {
	if v == nil {
		return emptyAction
	}
	return ActionOf(v)
}

var (
	_ ActionTiming = withTimingWrapper{}
)
