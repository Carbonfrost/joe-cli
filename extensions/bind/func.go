package bind

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
)

type evaluatorFunc func(_ context.Context, v any, yield func(any) error) error

type evaluatorInit struct {
	cli.Evaluator
	cli.Action
}

// Action obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Action[T any](fn func(T) cli.Action, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		bindTiming(func(c context.Context) error {
			a0, err := bind(c, t)
			if err != nil {
				return err
			}
			return cli.Do(c, fn(a0))
		}),
	)
}

// Action2 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Action2[T, U any](fn func(T, U) cli.Action, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		bindTiming(func(c context.Context) error {
			a0, a1, err := bind2(c, t, u)
			if err != nil {
				return err
			}
			return cli.Do(c, fn(a0, a1))
		}),
	)
}

// Action3 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Action3[T, U, V any](fn func(T, U, V) cli.Action, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		bindTiming(func(c context.Context) error {
			a0, a1, a2, err := bind3(c, t, u, v)
			if err != nil {
				return err
			}
			return cli.Do(c, fn(a0, a1, a2))
		}),
	)
}

// Call obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Call[T any](call func(T) error, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		bindTiming(func(c context.Context) error {
			a0, err := bind(c, t)
			if err != nil {
				return err
			}
			return call(a0)
		}),
	)
}

// Call2 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Call2[T, U any](call func(T, U) error, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		bindTiming(func(c context.Context) error {
			a0, a1, err := bind2(c, t, u)
			if err != nil {
				return err
			}
			return call(a0, a1)
		}),
	)
}

// Call3 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Call3[T, U, V any](call func(T, U, V) error, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		bindTiming(func(c context.Context) error {
			a0, a1, a2, err := bind3(c, t, u, v)
			if err != nil {
				return err
			}
			return call(a0, a1, a2)
		}),
	)
}

// Evaluator produces an evaluator from the bound values.
func Evaluator[T any](factory func(T) cli.Evaluator, t Binder[T]) cli.Evaluator {
	return newEvaluator(initializers(t),
		func(c context.Context, v any, yield func(any) error) error {
			a0, err := bind(c, t)
			if err != nil {
				return err
			}
			return factory(a0).Evaluate(c, v, yield)
		})
}

// Evaluator2 produces an evaluator from the bound values.
func Evaluator2[T, U any](eval func(T, U) cli.Evaluator, t Binder[T], u Binder[U]) cli.Evaluator {
	return newEvaluator(initializers(t, u),
		func(c context.Context, v any, yield func(any) error) error {
			a0, a1, err := bind2(c, t, u)
			if err != nil {
				return err
			}
			return eval(a0, a1).Evaluate(c, v, yield)
		})
}

// Evaluator3 produces an evaluator from the bound values.
func Evaluator3[T, U, V any](eval func(T, U, V) cli.Evaluator, t Binder[T], u Binder[U], v Binder[V]) cli.Evaluator {
	return newEvaluator(initializers(t, u, v),
		func(c context.Context, vany any, yield func(any) error) error {
			a0, a1, a2, err := bind3(c, t, u, v)
			if err != nil {
				return err
			}
			return eval(a0, a1, a2).Evaluate(c, vany, yield)
		})
}

// Indirect binds a value to the specified option indirectly.
// For example, it is common to define a FileSet arg and a Boolean flag that
// controls whether or not the file set is enumerated recursively.  You can use
// Indirect to update the arg indirectly by naming it and the bind function:
//
//	&cli.Arg{
//	    Name: "files",
//	    Value: new(cli.FileSet),
//	}
//	&cli.Flag{
//	    Name: "recursive",
//	    HelpText: "Whether files is recursively searched",
//	    Action: bind.Indirect("files", (*cli.FileSet).SetRecursive),
//	}
//
// The name parameter specifies the name of the flag or arg that is affected.  The
// bind function is the function to set the value, and valopt is optional, and if specified,
// indicates the value to set; otherwise, the value is read from the flag.
func Indirect[T, V any](name any, call func(T, V) error, valopt ...V) cli.Action {
	return Call2(call, Value[T](name), Exact(valopt...))
}

// Redirect binds a value to the specified option.
// A common use case for this action is to manually create aliases for
// other flags. For example, say you have a flag --proto= and a flag --tls1.2.
// You could use Redirect to support it.
//
//	&cli.Flag{
//	    Name: "proto",
//	}
//	&cli.Flag{
//	    Name: "tls1.2",
//	    HelpText: "Use TLS 1.2 connections",
//	    Action: bind.Redirect("proto", "tls1.2"),
//	}
//	&cli.Flag{
//	    Name: "tls1.3",
//	    HelpText: "Use TLS 1.3 connections",
//	    Action: bind.Redirect("proto", "tls1.3"),
//	}
//
// The name parameter specifies the name of the flag or arg that is affected.  The
// valopt is optional, and if specified, indicates the value to set; otherwise,
// the value is read from the flag.
func Redirect[V any](name any, valopt ...V) cli.Action {
	call := func(c *cli.Context, val V, _ V) error {
		return c.ContextOf(name).SetValue(val)
	}

	switch len(valopt) {
	case 0:
		return Call3(call, Context(), Value[V](""), Value[V](name))

	case 1:
		return Call3(call, Context(), Exact(valopt[0]), Value[V](name))

	default:
		panic("expected 0 or 1 args for valopt")
	}
}

func newEvaluator(initz cli.Action, evaluator evaluatorFunc) *evaluatorInit {
	return &evaluatorInit{
		Action:    cli.Pipeline(initz, willSetEvaluator(evaluator)),
		Evaluator: evaluator,
	}
}

func (e evaluatorFunc) Evaluate(c context.Context, v any, yield func(any) error) error {
	return e(c, v, yield)
}

func initializers(binders ...any) cli.Action {
	var result []cli.Action
	for index, binder := range binders {
		if b, ok := binder.(binderImpliedName); ok {
			result = append(result, willSetImpliedName(b, index))
		}
		if b, ok := binder.(binderInit); ok {
			result = append(result, b.Initializer())
		}
	}
	return cli.Setup{
		Optional: true,
		Uses:     cli.ActionPipeline(result),
	}
}

func willSetImpliedName(b binderImpliedName, index int) cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.IsFlag() || c.IsArg() {
			b.SetName("")
		} else {
			b.SetName(index)
		}
		return nil
	}
}

func willSetEvaluator(eval cli.Evaluator) cli.ActionFunc {
	return func(c *cli.Context) error {
		c.Target().(*cli.Expr).Evaluate = eval
		return nil
	}
}

func bindTiming(a func(context.Context) error) cli.Action {
	return cli.At(cli.ActionTiming, cli.ActionOf(a))
}
