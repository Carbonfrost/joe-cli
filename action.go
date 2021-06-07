package gocli

import (
  "fmt"
  "reflect"
)

// ActionFunc provides the basic function for
type ActionFunc func(*Context) error

// Action represents the building block of the various actions
// to perform when an app, command, flag, or argument is being evaluated.
type Action interface {
    Execute(*Context) error
}

var (
  emptyAction Action = ActionFunc(emptyActionImpl)
)

// Pipeline combines various actions into a single action
func Pipeline(actions ...interface{}) ActionFunc{
    myActions := make([]Action, 0, len(actions))
    for i, a := range actions {
        myActions[i] = NewAction(a)
    }

    return func(c *Context) (err error) {
        for _, a := range myActions {
            err = a.Execute(c)
            if err != nil {
                return
            }
        }
        return nil
    }
}

func NewAction(item interface{}) Action{
    switch a := item.(type) {
    case func(*Context)error:
        return ActionFunc(a)
    case Action:
        return a
    case nil:
        return nil
    case func(*Context):
        return ActionFunc(func(c *Context)error {
            a(c)
            return nil
        })
    case func()error:
        return ActionFunc(func(*Context)error {
            return a()
        })
    case func():
        return ActionFunc(func(*Context)error {
            a()
            return nil
        })
    }
    panic(fmt.Sprintf("unexpected type: %s", reflect.TypeOf(item)))
}

func (af ActionFunc) Execute(c *Context) error {
    return af(c)
}

func emptyActionImpl(*Context) error {
  return nil
}

func execute(af ActionFunc, c *Context) error {
  if af == nil {
    return nil
  }
  return af(c)
}
