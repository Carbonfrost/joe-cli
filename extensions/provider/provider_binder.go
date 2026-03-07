package provider

import (
	"context"
	"fmt"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

// ValueBinder provides a binder which works with the *provider.Value
type ValueBinder struct {
	delegateBinder[*Value]
}

type binder[T any] struct {
	delegateBinder[*Value]
}

type delegateBinder[T any] struct {
	bind.Binder[T]
}

func delegate[T any](b bind.Binder[T]) delegateBinder[T] {
	return delegateBinder[T]{b}
}

type binderInit interface {
	Initializer() cli.Action
}

// BindValue provides the binder the invokes the provider factory with
// its configured arguments
func Bind[T any](nameopt ...any) bind.Binder[T] {
	return &binder[T]{
		delegate(bind.Value[*Value](nameopt...)),
	}
}

// BindValue provides the binder the obtains the provider *Value
func BindValue(nameopt ...any) *ValueBinder {
	return &ValueBinder{
		delegate(bind.Value[*Value](nameopt...)),
	}
}

func (v *binder[T]) Bind(ctx context.Context) (T, error) {
	value, err := v.delegateBinder.Bind(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	c := cli.FromContext(ctx)
	reg, ok := Services(c).LookupRegistry(c.Target())
	if !ok {
		panic(fmt.Sprintf("registry not found %q", registryName(c.Target())))
	}
	result, err := reg.New(value.Name, value.rawArgs)
	if err != nil {
		return zero, err
	}
	return result.(T), nil
}

func (v delegateBinder[_]) Initializer() cli.Action {
	return v.Binder.(binderInit).Initializer()
}

func (v *ValueBinder) Bind(ctx context.Context) (*Value, error) {
	return v.delegateBinder.Bind(ctx)
}

func (v *ValueBinder) Args() bind.Binder[any] {
	return then(v, func(f *Value) any {
		return f.Args
	})
}

func (v *ValueBinder) Name() bind.Binder[string] {
	return then(v, func(f *Value) string {
		return f.Name
	})
}

func then[U any](b bind.Binder[*Value], fn func(*Value) U) bind.Binder[U] {
	return &thenBinder[U]{
		delegateBinder: delegate(b),
		thunk:          fn,
	}
}

type thenBinder[U any] struct {
	delegateBinder[*Value]
	thunk func(*Value) U
}

func (b *thenBinder[U]) Bind(c context.Context) (U, error) {
	t, err := b.delegateBinder.Bind(c)
	if err != nil {
		var zero U
		return zero, err
	}
	return b.thunk(t), nil
}

type bindFunc[T any] func(context.Context) (T, error)

func (f bindFunc[T]) Bind(c context.Context) (T, error) {
	return f(c)
}

var _ bind.Binder[any] = (*binder[any])(nil)
