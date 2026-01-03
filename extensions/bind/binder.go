// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Package bind provides functions for creating late-bound actions and evaluators.
// It's common to produce actions or evaluators that delegate to other
// actions or evaluators, to values in the context, or to models where the values
// that are needed have been set.  This package provides idioms that can simplify
// this code and encourage the pattern of separating the concerns.
//
// For example, compare the equivalent flags:
//
//	&cli.App{
//	    Name: "a",
//	    Flags: []*cli.Flag{
//	        {
//	            Name: "cumbersome",
//	            Action: func(c *cli.Context) error {
//	                value := c.Int("")
//	                return logic(value)
//	            },
//	            Value: new(int),
//	        },
//	        {
//	            Name: "clean",
//	            Uses: bind.Call(logic, bind.Int()),
//	        },
//	    },
//	}
//
// With the clean flag, you benefit from the implicit declaration of the type
// of the flag's Value and not having to map the value manually. In addition, it
// encourages you to factor out the logic as its own function with the
// signature func(int)error, which is decoupled from Joe's types and probably easier
// to test.
//
// # Evaluators
//
// Say that you have an expression that has an operand "-name TEXT",
// and you have a function SetName(string)cli.Evaluator that can produce the
// evaluator that is actually used. You can use the following to simplify the binding
// of the string argument.
//
//	&cli.Arg {
//	    Name: "expression",
//	    Value: cli.Expression{
//	        Exprs: []*cli.Expr{
//	            {
//	                Name: "name",
//	                Args: cli.Args(cli.String()),
//	                Evaluator: bind.Evaluator(SetName, bind.String()),
//	            }
//	        }
//	    }
//	}
//
// Notice that the bind.String() call doesn't require you to name the argument from
// which to obtain the value. When unspecified, it uses the first argument in the
// argument list for the Expr.
package bind

import (
	"context"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

// Binder provides a strategy for obtaining a value from the context
type Binder[T any] interface {
	// Bind obtains the value from the context
	Bind(context.Context) (T, error)
}

// Func provides a function that binds
type Func[T any] func(c *cli.Context) (T, error)

func (f Func[T]) Bind(c context.Context) (T, error) {
	return f(cli.FromContext(c))
}

type bindFunc[T any] func(context.Context) (T, error)

func (f bindFunc[T]) Bind(c context.Context) (T, error) {
	return f(c)
}

type exactBinder[V any] struct {
	binderSupport[V]
	v V
}

func (b *exactBinder[V]) Bind(_ context.Context) (V, error) {
	return b.v, nil
}

func (b *exactBinder[V]) Initializer() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		ctx := c.ContextOf(b.name())
		if ctx == nil {
			return nil
		}
		return ctx.Do(&cli.Prototype{Value: new(bool)})
	})
}

// binderSupport facilitates the implied naming/initializer logic that
// ensures that any flag or arg referenced by a binder gets a corresponding
// default value
type binderSupport[V any] struct {
	impliedName any
}

type binderInit interface {
	Initializer() cli.Action
}

type binderImpliedName interface {
	SetName(any)
}

type binderSupportInterface[V any] interface {
	binderInit
	binderImpliedName
	Binder[V]
}

type binder[V any] struct {
	binderSupport[V]
	lookupValue func(*cli.Context, any) V
}

func (b *binderSupport[_]) SetName(name any) {
	if b.impliedName == nil {
		b.impliedName = name
	}
}

func (b *binderSupport[V]) Initializer() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		ctx := c.ContextOf(b.name())
		if ctx == nil {
			return nil
		}
		return ctx.Do(&cli.Prototype{Value: support.BindSupportedValue(new(V))})
	})
}

func (b *binderSupport[_]) name() any {
	if b.impliedName == nil {
		return ""
	}
	return b.impliedName
}

func (b *binder[V]) Bind(c context.Context) (V, error) {
	return b.lookupValue(cli.FromContext(c), b.binderSupport.name()), nil
}

// Exact takes either the exact value that is specified
// or will take the value from the flag or arg.
func Exact[T any](valopt ...T) Binder[T] {
	if len(valopt) == 0 {
		return Value[T]()
	}
	if len(valopt) > 1 {
		panic("expected 0 or 1 args for valopt")
	}
	return wrapWithComposite(&exactBinder[T]{v: valopt[0]}).(Binder[T])
}

// Value obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Value[T any](nameopt ...any) Binder[T] {
	return wrapWithComposite(byName(contextValue[T], nameopt)).(Binder[T])
}

func contextValue[T any](c *cli.Context, name any) T {
	v := c.Value(name)
	return v.(T)
}

// FromContext locates a value within the context.
// A common value for the argument is cli.FromContext to obtain the cli.Context
// pointer. Indeed, the function [Context] provides this behavior.
func FromContext[T any](fn func(context.Context) T) Binder[T] {
	return bindFunc[T](func(c context.Context) (T, error) {
		return fn(c), nil
	})
}

// ContextValue locates a value within the context.
func ContextValue[T any](key any) Binder[T] {
	return FromContext(func(c context.Context) T {
		return c.Value(key).(T)
	})
}

// Context binds the context as a parameter.
func Context() Binder[*cli.Context] {
	return FromContext(cli.FromContext)
}

// Bool obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Bool(nameopt ...any) Binder[bool] {
	return byName((*cli.Context).Bool, nameopt)
}

// String obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func String(nameopt ...any) Binder[string] {
	return byName((*cli.Context).String, nameopt)
}

// List obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func List(nameopt ...any) Binder[[]string] {
	return byName((*cli.Context).List, nameopt)
}

// Int obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Int(nameopt ...any) Binder[int] {
	return byName((*cli.Context).Int, nameopt)
}

// Int8 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Int8(nameopt ...any) Binder[int8] {
	return byName((*cli.Context).Int8, nameopt)
}

// Int16 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Int16(nameopt ...any) Binder[int16] {
	return byName((*cli.Context).Int16, nameopt)
}

// Int32 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Int32(nameopt ...any) Binder[int32] {
	return byName((*cli.Context).Int32, nameopt)
}

// Int64 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Int64(nameopt ...any) Binder[int64] {
	return byName((*cli.Context).Int64, nameopt)
}

// Uint obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Uint(nameopt ...any) Binder[uint] {
	return byName((*cli.Context).Uint, nameopt)
}

// Uint8 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Uint8(nameopt ...any) Binder[uint8] {
	return byName((*cli.Context).Uint8, nameopt)
}

// Uint16 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Uint16(nameopt ...any) Binder[uint16] {
	return byName((*cli.Context).Uint16, nameopt)
}

// Uint32 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Uint32(nameopt ...any) Binder[uint32] {
	return byName((*cli.Context).Uint32, nameopt)
}

// Uint64 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Uint64(nameopt ...any) Binder[uint64] {
	return byName((*cli.Context).Uint64, nameopt)
}

// Float32 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Float32(nameopt ...any) Binder[float32] {
	return byName((*cli.Context).Float32, nameopt)
}

// Float64 obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Float64(nameopt ...any) Binder[float64] {
	return byName((*cli.Context).Float64, nameopt)
}

// Duration obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Duration(nameopt ...any) Binder[time.Duration] {
	return byName((*cli.Context).Duration, nameopt)
}

// File obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func File(nameopt ...any) *FileBinder {
	return wrapWithComposite(byName((*cli.Context).File, nameopt)).(*FileBinder)
}

// FileSet obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func FileSet(nameopt ...any) Binder[*cli.FileSet] {
	return byName((*cli.Context).FileSet, nameopt)
}

// Map obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Map(nameopt ...any) Binder[map[string]string] {
	return byName((*cli.Context).Map, nameopt)
}

// NameValue obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func NameValue(nameopt ...any) *NameValueBinder {
	return wrapWithComposite(byName((*cli.Context).NameValue, nameopt)).(*NameValueBinder)
}

// NameValues obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func NameValues(nameopt ...any) Binder[[]*cli.NameValue] {
	return byName((*cli.Context).NameValues, nameopt)
}

// URL obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func URL(nameopt ...any) Binder[*url.URL] {
	return byName((*cli.Context).URL, nameopt)
}

// Regexp obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Regexp(nameopt ...any) Binder[*regexp.Regexp] {
	return byName((*cli.Context).Regexp, nameopt)
}

// IP obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func IP(nameopt ...any) Binder[net.IP] {
	return byName((*cli.Context).IP, nameopt)
}

// BigInt obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func BigInt(nameopt ...any) Binder[*big.Int] {
	return byName((*cli.Context).BigInt, nameopt)
}

// BigFloat obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func BigFloat(nameopt ...any) Binder[*big.Float] {
	return byName((*cli.Context).BigFloat, nameopt)
}

// Bytes obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Bytes(nameopt ...any) Binder[[]byte] {
	return byName((*cli.Context).Bytes, nameopt)
}

// Interface obtains a binder that obtains a value from the context. If the name is
// not specified, then either the current flag or arg is used or the corresponding
// argument by index.
// When present in the Uses pipeline, this also sets up the corresponding flag or
// arg with a reasonable default of the same type.
func Interface(nameopt ...any) Binder[any] {
	return byName((*cli.Context).Value, nameopt)
}

func byName[T any](f func(*cli.Context, any) T, nameopt []any) *binder[T] {
	var name any
	switch len(nameopt) {
	case 0:
		break
	case 1:
		name = nameopt[0]
	default:
		panic("expected 0 or 1 args for nameopt")
	}
	return &binder[T]{
		binderSupport[T]{
			impliedName: name,
		},
		f,
	}
}

func bind[T any](c context.Context, t Binder[T]) (T, error) {
	return t.Bind(c)
}

func bind2[T, U any](c context.Context, t Binder[T], u Binder[U]) (a0 T, a1 U, err error) {
	a0, err = t.Bind(c)
	if err != nil {
		return
	}
	a1, err = u.Bind(c)
	return
}

func bind3[T, U, V any](c context.Context, t Binder[T], u Binder[U], v Binder[V]) (a0 T, a1 U, a2 V, err error) {
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

var (
	_ binderSupportInterface[any] = (*binder[any])(nil)
	_ binderSupportInterface[any] = (*exactBinder[any])(nil)
)

// FileBinder provides a binder for [cli.File]
type FileBinder struct {
	binderSupportInterface[*cli.File]
}

func (f *FileBinder) Bind(c context.Context) (*cli.File, error) {
	return f.binderSupportInterface.Bind(c)
}

// Name obtains the name for the file
func (f *FileBinder) Name() Binder[string] {
	return then(f, func(f *cli.File) string {
		return f.Name
	})
}

// Dir obtains the name for the file
func (f *FileBinder) Dir() Binder[string] {
	return then(f, (*cli.File).Dir)
}

// Exists obtains the name for the file
func (f *FileBinder) Exists() Binder[bool] {
	return then(f, (*cli.File).Exists)
}

// Ext obtains the name for the file
func (f *FileBinder) Ext() Binder[string] {
	return then(f, (*cli.File).Ext)
}

// Base obtains the name for the file
func (f *FileBinder) Base() Binder[string] {
	return then(f, (*cli.File).Base)
}

// NameValueBinder provides a binder for [cli.NameValue]
type NameValueBinder struct {
	binderSupportInterface[*cli.NameValue]
}

// Name provides a delegate binder which obtains the name part
func (f *NameValueBinder) Name() Binder[string] {
	return then(f, func(f *cli.NameValue) string {
		return f.Name
	})
}

// Value provides a delegate binder which obtains the value part
func (f *NameValueBinder) Value() Binder[string] {
	return then(f, func(f *cli.NameValue) string {
		return f.Value
	})
}

func then[T, U any](b Binder[T], fn func(T) U) bindFunc[U] {
	return func(c context.Context) (U, error) {
		t, err := b.Bind(c)
		if err != nil {
			var zero U
			return zero, err
		}
		return fn(t), nil
	}
}

func wrapWithComposite[V any](in binderSupportInterface[V]) any {
	var zero V
	switch any(zero).(type) {
	case *cli.File:
		return &FileBinder{in.(binderSupportInterface[*cli.File])}
	case *cli.NameValue:
		return &NameValueBinder{in.(binderSupportInterface[*cli.NameValue])}
	}
	return in
}

var _ Binder[*cli.File] = (*FileBinder)(nil)
