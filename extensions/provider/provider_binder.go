package provider

import (
	"context"
	"fmt"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

// ValueBinder provides a binder which works with the *provider.Value
type ValueBinder struct {
	b bind.Binder[*Value]
}

type binder[T any] struct {
	b bind.Binder[*Value]
}

// BindValue provides the binder the invokes the provider factory with
// its configured arguments
func Bind[T any](nameopt ...any) bind.Binder[T] {
	return &binder[T]{
		b: bind.Value[*Value](nameopt...),
	}
}

// BindValue provides the binder the obtains the provider *Value
func BindValue(nameopt ...any) *ValueBinder {
	return &ValueBinder{
		bind.Value[*Value](nameopt...),
	}
}

func (v *binder[T]) Bind(ctx context.Context) (T, error) {
	value, err := v.b.Bind(ctx)
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

func (v *ValueBinder) Bind(ctx context.Context) (*Value, error) {
	return v.b.Bind(ctx)
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

func then[T, U any](b bind.Binder[T], fn func(T) U) bindFunc[U] {
	return func(c context.Context) (U, error) {
		t, err := b.Bind(c)
		if err != nil {
			var zero U
			return zero, err
		}
		return fn(t), nil
	}
}

type bindFunc[T any] func(context.Context) (T, error)

func (f bindFunc[T]) Bind(c context.Context) (T, error) {
	return f(c)
}

var _ bind.Binder[any] = (*binder[any])(nil)
