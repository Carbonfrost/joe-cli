// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bind

import (
	"context"

	"github.com/Carbonfrost/joe-cli"
)

// Action obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Action[T any, Action cli.Action](fn func(T) Action, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		actioner(cli.ActionTiming, fn, t),
	)
}

// Action2 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Action2[T, U any, Action cli.Action](fn func(T, U) Action, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		actioner2(cli.ActionTiming, fn, t, u),
	)
}

// Action3 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Action3[T, U, V any, Action cli.Action](fn func(T, U, V) Action, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		actioner3(cli.ActionTiming, fn, t, u, v),
	)
}

// Call obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Call[T any](call func(T) error, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		caller(cli.ActionTiming, call, t),
	)
}

// Call2 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Call2[T, U any](call func(T, U) error, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		caller2(cli.ActionTiming, call, t, u),
	)
}

// Call3 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Action timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Call3[T, U, V any](call func(T, U, V) error, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		caller3(cli.ActionTiming, call, t, u, v),
	)
}

// BeforeCall obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Before timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func BeforeCall[T any](call func(T) error, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		caller(cli.BeforeTiming, call, t),
	)
}

// BeforeCall2 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Before timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func BeforeCall2[T, U any](call func(T, U) error, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		caller2(cli.BeforeTiming, call, t, u),
	)
}

// BeforeCall3 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Before timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func BeforeCall3[T, U, V any](call func(T, U, V) error, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		caller3(cli.BeforeTiming, call, t, u, v),
	)
}

// Before obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Before timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Before[T any, Action cli.Action](fn func(T) Action, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		actioner(cli.BeforeTiming, fn, t),
	)
}

// Before2 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Before timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Before2[T, U any, Action cli.Action](fn func(T, U) Action, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		actioner2(cli.BeforeTiming, fn, t, u),
	)
}

// Before3 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// Before timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func Before3[T, U, V any, Action cli.Action](fn func(T, U, V) Action, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		actioner3(cli.BeforeTiming, fn, t, u, v),
	)
}

// AfterCall obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// After timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func AfterCall[T any](call func(T) error, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		caller(cli.AfterTiming, call, t),
	)
}

// AfterCall2 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// After timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func AfterCall2[T, U any](call func(T, U) error, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		caller2(cli.AfterTiming, call, t, u),
	)
}

// AfterCall3 obtains an action invokes the function, binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// After timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func AfterCall3[T, U, V any](call func(T, U, V) error, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		caller3(cli.AfterTiming, call, t, u, v),
	)
}

// After obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// After timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func After[T any, Action cli.Action](fn func(T) Action, t Binder[T]) cli.Action {
	return cli.Pipeline(
		initializers(t),
		actioner(cli.AfterTiming, fn, t),
	)
}

// After2 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// After timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func After2[T, U any, Action cli.Action](fn func(T, U) Action, t Binder[T], u Binder[U]) cli.Action {
	return cli.Pipeline(
		initializers(t, u),
		actioner2(cli.AfterTiming, fn, t, u),
	)
}

// After3 obtains an action invokes the function to derive another action
// whilst binding the parameters.
// If this is added to the Uses timing, it will actually be run in the
// After timing and the binders can also provide initializers if they
// have the method Initializer() Action (as the binders in this package do).
func After3[T, U, V any, Action cli.Action](fn func(T, U, V) Action, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.Pipeline(
		initializers(t, u, v),
		actioner3(cli.AfterTiming, fn, t, u, v),
	)
}

func actioner[T any, Action cli.Action](time cli.Timing, fn func(T) Action, t Binder[T]) cli.Action {
	return cli.At(time, cli.ActionOf(func(c context.Context) error {
		a0, err := bind(c, t)
		if err != nil {
			return err
		}
		return cli.Do(c, fn(a0))
	}))
}

func actioner2[T, U any, Action cli.Action](time cli.Timing, fn func(T, U) Action, t Binder[T], u Binder[U]) cli.Action {
	return cli.At(time, cli.ActionOf(func(c context.Context) error {
		a0, a1, err := bind2(c, t, u)
		if err != nil {
			return err
		}
		return cli.Do(c, fn(a0, a1))
	}))
}

func actioner3[T, U, V any, Action cli.Action](time cli.Timing, fn func(T, U, V) Action, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.At(time, cli.ActionOf(func(c context.Context) error {
		a0, a1, a2, err := bind3(c, t, u, v)
		if err != nil {
			return err
		}
		return cli.Do(c, fn(a0, a1, a2))
	}))
}

func caller[T any](time cli.Timing, call func(T) error, t Binder[T]) cli.Action {
	return cli.At(time, cli.ActionOf(func(c context.Context) error {
		a0, err := bind(c, t)
		if err != nil {
			return err
		}
		return call(a0)
	}))
}

func caller2[T, U any](time cli.Timing, call func(T, U) error, t Binder[T], u Binder[U]) cli.Action {
	return cli.At(time, cli.ActionOf(func(c context.Context) error {
		a0, a1, err := bind2(c, t, u)
		if err != nil {
			return err
		}
		return call(a0, a1)
	}))
}

func caller3[T, U, V any](time cli.Timing, call func(T, U, V) error, t Binder[T], u Binder[U], v Binder[V]) cli.Action {
	return cli.At(time, cli.ActionOf(func(c context.Context) error {
		a0, a1, a2, err := bind3(c, t, u, v)
		if err != nil {
			return err
		}
		return call(a0, a1, a2)
	}))
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

// SetPointer sets a pointer as the binding action
func SetPointer[V any](v *V, binder Binder[V]) cli.Action {
	return Call(func(in V) (_ error) {
		*v = in
		return
	}, binder)
}

// Initializers obtains the initializers for a sequence of binders.
// For a binder that has a method Initializer() Action, such method will be called
// to retrieve the initializer. For a binder that has a method SetName(any), the
// method will be called to set the implicit name that can be used when a binder is
// used within a function. Refer to the package overview for information about implicit
// naming.
func Initializers(binders ...any) cli.Action {
	return initializers(binders...)
}

func initializers(binders ...any) cli.Action {
	var uses, setters []cli.Action
	for index, binder := range binders {
		if b, ok := binder.(binderImpliedName); ok {
			setters = append(setters, willSetImpliedName(b, index))
		}
		if b, ok := binder.(binderInit); ok {
			uses = append(uses, b.Initializer())
		}
	}

	return cli.ActionPipeline(setters).Append(cli.Setup{
		Optional: true,
		Uses:     cli.ActionPipeline(uses),
	})
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
