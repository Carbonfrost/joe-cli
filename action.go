package cli

import (
	"context"
	"fmt"
	"reflect"
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

var (
	emptyAction ActionHandler = ActionFunc(emptyActionImpl)
)

// Pipeline combines various actions into a single action
func Pipeline(actions ...interface{}) *ActionPipeline {
	myActions := make([]ActionHandler, len(actions))
	for i, a := range actions {
		myActions[i] = Action(a)
	}

	return &ActionPipeline{myActions}
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
	panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(item)))
}

func ContextValue(key, value interface{}) ActionFunc {
	return func(c *Context) error {
		c.Context = context.WithValue(c.Context, key, value)
		return nil
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
	return p
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

func unwind(x ActionHandler) []ActionHandler {
	if x == nil {
		return nil
	}
	if pipe, ok := x.(*ActionPipeline); ok {
		return pipe.items
	}
	return []ActionHandler{x}
}
