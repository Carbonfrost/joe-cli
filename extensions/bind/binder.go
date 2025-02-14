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
	"fmt"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
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

type binderInit interface {
	Initializer() cli.Action
}

type binderImpliedName interface {
	SetName(any)
}

type binder[V any] struct {
	impliedName any
	lookupValue func(*cli.Context, any) V
}

var (
	valueType = reflect.TypeFor[cli.Value]()
)

func (b *binder[V]) SetName(name any) {
	if b.impliedName == nil {
		b.impliedName = name
	}
}

func (b *binder[V]) Initializer() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		ctx := c.ContextOf(b.name())
		if ctx == nil {
			return nil
		}
		return ctx.Do(&cli.Prototype{Value: bindSupportedValue(new(V))})
	})
}

func (b *binder[V]) name() any {
	if b.impliedName == nil {
		return ""
	}
	return b.impliedName
}

func (b *binder[V]) Bind(c context.Context) (V, error) {
	return b.lookupValue(cli.FromContext(c), b.name()), nil
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
func File(nameopt ...any) Binder[*cli.File] {
	return byName((*cli.Context).File, nameopt)
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
func NameValue(nameopt ...any) Binder[*cli.NameValue] {
	return byName((*cli.Context).NameValue, nameopt)
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

// For provides the binder of the specified type
func For[T any](nameopt ...any) Binder[T] {
	var t T
	return func() any {
		switch any(t).(type) {
		case bool:
			return Bool(nameopt...)
		case string:
			return String(nameopt...)
		case []string:
			return List(nameopt...)
		case int:
			return Int(nameopt...)
		case int8:
			return Int8(nameopt...)
		case int16:
			return Int16(nameopt...)
		case int32:
			return Int32(nameopt...)
		case int64:
			return Int64(nameopt...)
		case uint:
			return Uint(nameopt...)
		case uint8:
			return Uint8(nameopt...)
		case uint16:
			return Uint16(nameopt...)
		case uint32:
			return Uint32(nameopt...)
		case uint64:
			return Uint64(nameopt...)
		case float32:
			return Float32(nameopt...)
		case float64:
			return Float64(nameopt...)
		case time.Duration:
			return Duration(nameopt...)
		case *cli.File:
			return File(nameopt...)
		case *cli.FileSet:
			return FileSet(nameopt...)
		case map[string]string:
			return Map(nameopt...)
		case *cli.NameValue:
			return NameValue(nameopt...)
		case []*cli.NameValue:
			return NameValues(nameopt...)
		case *url.URL:
			return URL(nameopt...)
		case *regexp.Regexp:
			return Regexp(nameopt...)
		case *net.IP:
			return IP(nameopt...)
		case *big.Int:
			return BigInt(nameopt...)
		case *big.Float:
			return BigFloat(nameopt...)
		case []byte:
			return Bytes(nameopt...)
		case any:
			return Interface(nameopt...)
		default:
			panic(fmt.Sprintf("unexpected target type %T", t))
		}
	}().(Binder[T])
}

func byName[T any](f func(*cli.Context, any) T, nameopt []any) Binder[T] {
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
		impliedName: name,
		lookupValue: f,
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

func bindSupportedValue(v interface{}) interface{} {
	// Bind functions will either use *V or V depending upon what
	// supports the built-in convention values or implements Value.
	// Any built-in primitive will work as is.  However, if v is actually
	// *V but V is **W and W is a Value implementation, then unwrap this
	// so we end up with *W.  For example, instead of **FileSet, just use *FileSet.
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Ptr {
		pointsToValue := val.Elem().Type()
		if pointsToValue.Implements(valueType) {
			return reflect.New(pointsToValue.Elem()).Interface()
		}
	}

	// Primitives and other values
	return v
}

var _ binderInit = (*binder[any])(nil)
var _ binderImpliedName = (*binder[any])(nil)
