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
	setCategory(name string)
	setData(name string, v interface{})
}

type actionPipelines struct {
	Uses   ActionHandler // Must be strictly initializers (no automatic regrouping)
	Before ActionHandler
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
		Before: ActionFunc(triggerFlagsAndArgs),
	}

	defaultOption = actionPipelines{
		Before: Pipeline(
			ActionFunc(setupOptionFromOptions()),
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
	for _, a := range p.items {
		err = a.Execute(c)
		if err != nil {
			return
		}
	}
	return nil
}

func (p *ActionPipeline) takeInitializers(c *Context) (*actionPipelines, error) {
	res := map[timing][]ActionHandler{}
	var add = func(t timing, h ActionHandler) {
		if _, ok := res[t]; ok {
			res[t] = append(res[t], h)
		} else {
			res[t] = []ActionHandler{h}
		}
	}
	for _, h := range p.items {
		t := timingOf(h, initialTiming)
		if t == initialTiming {
			err := c.Do(h)
			if err != nil {
				return nil, err
			}
			continue
		}

		add(t, h)
	}
	return newActionPipelines(res), nil
}

func (w *withTimingWrapper) timing() timing {
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

func hookExecute(x, y ActionHandler, c *Context) error {
	if err := execute(x, c); err != nil {
		return err
	}
	return execute(y, c)
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

func takeInitializers(from ActionHandler, c *Context) (*actionPipelines, error) {
	if p, ok := from.(*ActionPipeline); ok {
		return p.takeInitializers(c)
	}

	return &actionPipelines{}, execute(from, c)
}

func unwind(x ActionHandler) []ActionHandler {
	if x == nil {
		return nil
	}
	if pipe, ok := x.(*ActionPipeline); ok {
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
