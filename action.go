package cli

import (
	"context"
	"fmt"
)

// ActionFunc provides the basic function for
type ActionFunc func(*Context) error

//counterfeiter:generate . ActionHandler

// ActionHandler represents the building block of the various actions
// to perform when an app, command, flag, or argument is being evaluated.
type ActionHandler interface {
	Execute(*Context) error
}

type ActionPipeline struct {
	items []ActionHandler
}

type target interface {
	hooks() *hooks
	options() Option
	setCategory(name string)
	setData(name string, v interface{})
	setInternalFlags(internalFlags)
	internalFlags() internalFlags
}

type actionPipelines struct {
	Uses   ActionHandler // Must be strictly initializers (no automatic regrouping)
	Before ActionHandler
	Action ActionHandler
	After  ActionHandler
}

type actionHandlerTiming interface {
	ActionHandler
	timing() timing
}

type withTimingWrapper struct {
	ActionHandler
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
	emptyAction ActionHandler = ActionFunc(emptyActionImpl)

	defaultApp = actionPipelines{
		Uses: Pipeline(
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
	myActions := make([]ActionHandler, len(actions))
	for i, a := range actions {
		myActions[i] = Action(a)
	}

	return &ActionPipeline{myActions}
}

func Before(a ActionHandler) ActionHandler {
	return withTiming(a, beforeTiming)
}

func After(a ActionHandler) ActionHandler {
	return withTiming(a, afterTiming)
}

// Initializer marks an action handler as being for the initialization phase.  When such a handler
// is added to the Uses pipeline, it will automatically be associated correctly with the initialization
// of the value.  Otherwise, this handler is not special
func Initializer(a ActionHandler) ActionHandler {
	return withTiming(a, initialTiming)
}

func timingOf(a ActionHandler, defaultTiming timing) timing {
	switch val := a.(type) {
	case actionHandlerTiming:
		return val.timing()
	}
	return defaultTiming
}

func withTiming(a ActionHandler, t timing) ActionHandler {
	return withTimingWrapper{a, t}
}

func Action(item interface{}) ActionHandler {
	switch a := item.(type) {
	case nil:
		return nil
	case func(*Context) error:
		return ActionFunc(a)
	case ActionHandler:
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
func Data(name string, value interface{}) ActionHandler {
	return ActionFunc(func(c *Context) error {
		c.target().setData(name, value)
		return nil
	})
}

// Category sets the category of a command, flag, or expression.  This handler is generally
// set up inside a Uses pipeline.
func Category(name string) ActionHandler {
	return ActionFunc(func(c *Context) error {
		c.target().setCategory(name)
		return nil
	})
}

// HookBefore registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookBefore(pattern string, handler ActionHandler) ActionHandler {
	return ActionFunc(func(c *Context) error {
		c.demandInit().hookBefore(pattern, handler)
		return nil
	})
}

// HookAfter registers a hook that runs for the matching elements.  See ContextPath for
// the syntax of patterns and how they are matched.
func HookAfter(pattern string, handler ActionHandler) ActionHandler {
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
func OptionalValue(v interface{}) ActionHandler {
	return Initializer(ActionFunc(func(c *Context) error {
		c.Flag().setOptionalValue(v)
		return nil
	}))
}

func newActionPipelines(m map[timing][]ActionHandler) *actionPipelines {
	var pipe = func(h []ActionHandler) ActionHandler {
		if len(h) == 0 {
			return emptyAction
		}
		return &ActionPipeline{h}
	}
	return &actionPipelines{
		Uses:   pipe(m[initialTiming]),
		Before: pipe(m[beforeTiming]),
		After:  pipe(m[afterTiming]),
	}
}

func (af ActionFunc) Execute(c *Context) error {
	if af == nil {
		return nil
	}
	return af(c)
}

func (p *ActionPipeline) Append(x ActionHandler) *ActionPipeline {
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

func (p *actionPipelines) add(t timing, h ActionHandler) {
	switch t {
	case initialTiming:
		p.Uses = pipeline(p.Uses, h)
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

func (w withTimingWrapper) timing() timing {
	return w.t
}

func emptyActionImpl(*Context) error {
	return nil
}

func execute(af ActionHandler, c *Context) error {
	if af == nil {
		return nil
	}
	return af.Execute(c)
}

func executeAll(c *Context, x ...ActionHandler) error {
	for _, y := range x {
		if err := execute(y, c); err != nil {
			return err
		}
	}
	return nil
}

func doThenExit(a ActionHandler) ActionFunc {
	return func(c *Context) error {
		err := a.Execute(c)
		if err != nil {
			return err
		}
		return Exit(0)
	}
}

func pipeline(x, y ActionHandler) *ActionPipeline {
	return &ActionPipeline{
		items: append(unwind(x), unwind(y)...),
	}
}

func takeInitializers(uses ActionHandler, opts Option, c *Context) (*actionPipelines, error) {
	return pipeline(uses, opts.wrap()).takeInitializers(c)
}

func unwind(x ActionHandler) []ActionHandler {
	if x == nil {
		return nil
	}
	if pipe, ok := x.(*ActionPipeline); ok && pipe != nil {
		return pipe.items
	}
	return []ActionHandler{x}
}

func actionOrEmpty(v interface{}) ActionHandler {
	if v == nil {
		return emptyAction
	}
	return Action(v)
}

var (
	_ actionHandlerTiming = withTimingWrapper{}
)
