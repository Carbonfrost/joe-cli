package provider

import (
	"context"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

// ValueBinder provides a binder which works with the *provider.Value
type ValueBinder struct {
	delegateBinder[*Value]
}

type binder[T any] struct {
	delegateBinder[*Value]
	name any
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

// Bind provides the binder the invokes the provider factory with
// its configured arguments
func Bind[T any](nameopt ...any) bind.Binder[T] {
	var name any = nil
	if len(nameopt) > 0 && nameopt[0] != "" {
		name = nameopt[0]
	}

	return &binder[T]{
		delegateBinder: delegate(bind.Value[*Value](nameopt...)),
		name:           name,
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
	target := v.findTarget(c)

	result, err := Services(c).New(target, value.Name, value.Args)
	if err != nil {
		return zero, err
	}
	return result.(T), nil
}

func (v *binder[_]) findTarget(c *cli.Context) any {
	if v.name != nil {
		// When present, the name is interpreted as the name of a flag
		// or arg
		flag, ok := c.LookupFlag(v.name)
		if ok {
			return flag
		}

		arg, ok := c.LookupArg(v.name)
		if !ok {
			return arg
		}
		return v.name
	}
	return c.Target()
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
	return bind.SeqContext(b, func(_ context.Context, t *Value) (U, error) {
		return fn(t), nil
	})
}

var _ bind.Binder[any] = (*binder[any])(nil)
