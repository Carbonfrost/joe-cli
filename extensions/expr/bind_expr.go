package expr

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

type evaluatorInit struct {
	Evaluator
	cli.Action
}

// ActionEvaluator provides an evaluator which can also be used as an action
type ActionEvaluator interface {
	Evaluator
	cli.Action
}

// NewActionEvaluator provides an evaluator which also provides an action.
// The typical use is to implement an initializer within the action
// that sets up the required flags and actions that the evaluator
// depends on.
func NewActionEvaluator(action cli.Action, eval Evaluator) ActionEvaluator {
	return &evaluatorInit{
		Action:    cli.ActionOf(action), // allow action to be nil
		Evaluator: eval,
	}
}

// BindEvaluator produces an evaluator from the bound values.
func BindEvaluator[T any, E Evaluator](factory func(T) E, t bind.Binder[T]) Evaluator {
	return newEvaluator(initializers(t),
		func(c context.Context, v any, yield func(any) error) error {
			a0, err := bind1(c, t)
			if err != nil {
				return err
			}
			return factory(a0).Evaluate(c, v, yield)
		})
}

// BindEvaluator2 produces an evaluator from the bound values.
func BindEvaluator2[T, U any, E Evaluator](eval func(T, U) E, t bind.Binder[T], u bind.Binder[U]) Evaluator {
	return newEvaluator(initializers(t, u),
		func(c context.Context, v any, yield func(any) error) error {
			a0, a1, err := bind2(c, t, u)
			if err != nil {
				return err
			}
			return eval(a0, a1).Evaluate(c, v, yield)
		})
}

// BindEvaluator3 produces an evaluator from the bound values.
func BindEvaluator3[T, U, V any, E Evaluator](eval func(T, U, V) E, t bind.Binder[T], u bind.Binder[U], v bind.Binder[V]) Evaluator {
	return newEvaluator(initializers(t, u, v),
		func(c context.Context, vany any, yield func(any) error) error {
			a0, a1, a2, err := bind3(c, t, u, v)
			if err != nil {
				return err
			}
			return eval(a0, a1, a2).Evaluate(c, vany, yield)
		})
}

func newEvaluator(initz cli.Action, evaluator evaluatorFunc) *evaluatorInit {
	return &evaluatorInit{
		Action:    cli.Pipeline(initz, SetEvaluator(evaluator)),
		Evaluator: evaluator,
	}
}

func initializers(binders ...any) cli.Action {
	return bind.Initializers(binders...)
}

func bind1[T any](c context.Context, t bind.Binder[T]) (T, error) {
	return t.Bind(c)
}

func bind2[T, U any](c context.Context, t bind.Binder[T], u bind.Binder[U]) (a0 T, a1 U, err error) {
	a0, err = t.Bind(c)
	if err != nil {
		return
	}
	a1, err = u.Bind(c)
	return
}

func bind3[T, U, V any](c context.Context, t bind.Binder[T], u bind.Binder[U], v bind.Binder[V]) (a0 T, a1 U, a2 V, err error) {
	a0, err = t.Bind(c)
	if err != nil {
		return
	}
	a1, err = u.Bind(c)
	if err != nil {
		return
	}

	a2, err = v.Bind(c)
	return
}
