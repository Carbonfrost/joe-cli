// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expr

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Carbonfrost/joe-cli"
)

// contextKind identifies which context parameter, if any, a reflected
// evaluator function declares.
type contextKind int

const (
	noContext contextKind = iota
	plainContext
	cliContext
)

// resultKind identifies the result, if any, of a reflected evaluator function.
type resultKind int

const (
	noResult resultKind = iota
	errorResult
	boolResult
)

// reflectSignature describes the parameters and results of a function which
// can be used as an evaluator.
type reflectSignature struct {
	context  contextKind
	value    reflect.Type // nil when the function takes no value
	yield    bool
	result   resultKind
	function reflect.Value
}

var (
	contextType    = reflect.TypeFor[context.Context]()
	cliContextType = reflect.TypeFor[*cli.Context]()
	yielderType    = reflect.TypeFor[func(any) error]()
	errorType      = reflect.TypeFor[error]()
)

// Do converts action to an evaluator
func Do(a cli.Action) Evaluator {
	return evaluatorFunc(func(ctx context.Context, v any, y func(any) error) error {
		err := cli.Do(ctx, a)
		if err != nil {
			return err
		}
		return y(v)
	})
}

// reflectEvaluatorOf creates an evaluator from a function using reflection,
// which makes it possible to use the signatures which EvaluatorOf accepts
// while naming types that are more specific than any.  For example,
// func(context.Context, string) bool is accepted, and each value from the
// expression pipeline is passed to it after being converted to a string.
// When a value can't be used with the parameter type of the function,
// evaluation returns an error.
//
// The function must have the shape
// func([context], [value], [yield]) [error or bool] where the context
// parameter, when present, is context.Context or *cli.Context; the value
// parameter, when present, can have any type; and the yield parameter, when
// present, must be exactly func(any) error.  No covariance is allowed for the
// yield function.  Results are treated as they are in EvaluatorOf: an error
// stops the evaluation, and false in the case of a bool filters out the value.
// Because a function which takes the yielder is responsible for yielding
// values itself, it can't also return bool.
func reflectEvaluatorOf(v any) Evaluator {
	sig, ok := newReflectSignature(v)
	if !ok {
		panic(fmt.Sprintf("unexpected type: %T", v))
	}
	if sig.function.IsNil() {
		// Consistent with the nil handling of EvaluatorFunc, do nothing
		return evaluatorFunc(nil)
	}
	return evaluatorFunc(sig.evaluate)
}

func newReflectSignature(v any) (*reflectSignature, bool) {
	fn := reflect.ValueOf(v)
	if !fn.IsValid() || fn.Kind() != reflect.Func {
		return nil, false
	}

	typ := fn.Type()
	if typ.IsVariadic() {
		return nil, false
	}

	sig := &reflectSignature{function: fn}
	var index int

	if index < typ.NumIn() {
		switch typ.In(index) {
		case contextType:
			sig.context = plainContext
			index++
		case cliContextType:
			sig.context = cliContext
			index++
		}
	}
	if index < typ.NumIn() && typ.In(index) != yielderType {
		sig.value = typ.In(index)
		index++
	}
	if index < typ.NumIn() && typ.In(index) == yielderType {
		sig.yield = true
		index++
	}
	if index != typ.NumIn() {
		return nil, false
	}

	switch typ.NumOut() {
	case 0:
	case 1:
		switch out := typ.Out(0); {
		case out == errorType:
			sig.result = errorResult
		case out.Kind() == reflect.Bool && !sig.yield:
			sig.result = boolResult
		default:
			return nil, false
		}
	default:
		return nil, false
	}
	return sig, true
}

func (s *reflectSignature) evaluate(c context.Context, v any, y func(any) error) error {
	args, err := s.args(c, v, y)
	if err != nil {
		return err
	}

	res := s.function.Call(args)
	switch s.result {
	case errorResult:
		if err, _ := reflect.TypeAssert[error](res[0]); err != nil {
			return err
		}
	case boolResult:
		if !res[0].Bool() {
			return nil
		}
	}

	if s.yield || y == nil {
		return nil
	}
	return y(v)
}

func (s *reflectSignature) args(c context.Context, v any, y func(any) error) ([]reflect.Value, error) {
	args := make([]reflect.Value, 0, s.function.Type().NumIn())

	switch s.context {
	case plainContext:
		args = append(args, valueOrZero(c, contextType))
	case cliContext:
		var actual *cli.Context
		if c != nil {
			actual = cli.FromContext(c)
		}
		args = append(args, valueOrZero(actual, cliContextType))
	}

	if s.value != nil {
		arg, err := valueArg(s.value, v)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	if s.yield {
		args = append(args, valueOrZero(y, yielderType))
	}
	return args, nil
}

// valueArg converts the value from the expression pipeline so that it can be
// used with a parameter of the given type.
func valueArg(t reflect.Type, v any) (reflect.Value, error) {
	if v == nil {
		return reflect.Zero(t), nil
	}
	if actual := reflect.ValueOf(v); actual.Type().AssignableTo(t) {
		return actual, nil
	}
	return reflect.Value{}, fmt.Errorf("unsupported value: %T", v)
}

// valueOrZero obtains the reflected value of v, which is the zero value of the
// given type when v is nil.  (reflect.ValueOf provides an invalid value for
// nil, which can't be used as an argument.)
func valueOrZero(v any, t reflect.Type) reflect.Value {
	res := reflect.ValueOf(v)
	if !res.IsValid() {
		return reflect.Zero(t)
	}
	return res
}
