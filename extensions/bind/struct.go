// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bind

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

type structBinder[T any] struct {
	fields []structField
}

type structField struct {
	flagName string

	// declared is the name of the field as it is declared, used in errors
	declared string

	index []int
	typ   reflect.Type
}

// Struct obtains a binder which considers the context to create an instance of the
// struct type T.  The name of each of the exported fields of T is inflected to the
// naming convention of flags in order to retrieve the corresponding value.  For
// example, the field ConfigFile is bound from the flag or arg named config-file.
// (Refer to the package overview for more about how names are inflected.)
// The fields of an embedded struct are bound as if they were declared by T itself,
// which is consistent with how Go promotes them.  Unexported fields are ignored,
// including the fields promoted by an embedded struct which is itself unexported
// (such fields can't be set).  A field for which no flag or arg is defined simply
// retains its zero value.
//
// When present in the Uses pipeline, this also sets up each of the corresponding
// flags or args with a reasonable default of the same type as the field it binds.
// The function panics if T is not a struct type.
func Struct[T any]() Binder[T] {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("expected struct type for T, got %v", typ))
	}
	return &structBinder[T]{fields: structFields(typ, nil)}
}

func structFields(typ reflect.Type, index []int) []structField {
	res := make([]structField, 0, typ.NumField())
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}

		nested := append(slices.Clone(index), i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			res = append(res, structFields(f.Type, nested)...)
			continue
		}

		res = append(res, structField{
			flagName: inflectName(f.Name),
			declared: f.Name,
			index:    nested,
			typ:      f.Type,
		})
	}
	return res
}

func (b *structBinder[T]) Bind(ctx context.Context) (T, error) {
	var res T

	c := cli.FromContext(ctx)
	target := reflect.ValueOf(&res).Elem()

	for _, f := range b.fields {
		value := c.Value(f.flagName)
		if value == nil {
			// Either no flag or arg was defined by this name or it has no
			// value, so the field keeps its zero value
			continue
		}

		actual, ok := convertToField(value, f.typ)
		if !ok {
			return res, conversionError(c, f, value)
		}
		target.FieldByIndex(f.index).Set(actual)
	}

	return res, nil
}

func (b *structBinder[_]) Initializer() cli.Action {
	uses := make([]any, 0, len(b.fields))
	for _, f := range b.fields {
		uses = append(uses, cli.WithContextOf(f.flagName, &cli.Prototype{
			Value: support.BindSupportedValue(reflect.New(f.typ).Interface()),
		}))
	}
	return cli.Pipeline(uses...)
}

func convertToField(value any, typ reflect.Type) (reflect.Value, bool) {
	val := reflect.ValueOf(value)
	switch {
	case val.Type().AssignableTo(typ):
		return val, true

	case val.Kind() == reflect.Pointer && val.Type().Elem().AssignableTo(typ):
		if val.IsNil() {
			return reflect.Zero(typ), true
		}
		return val.Elem(), true

	case sameKindConvertible(val.Type(), typ):
		return val.Convert(typ), true
	}

	return reflect.Value{}, false
}

func sameKindConvertible(from, to reflect.Type) bool {
	if !from.ConvertibleTo(to) {
		return false
	}
	return from.Kind() == to.Kind() || (isNumeric(from.Kind()) && isNumeric(to.Kind()))
}

func isNumeric(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func inflectName(name string) string {
	var buf strings.Builder

	runes := []rune(name)
	var last rune

	for i, r := range runes {
		if r == '_' {
			r = '-'
		}
		if r == '-' {
			if last == '-' || buf.Len() == 0 {
				continue
			}
		} else if last != 0 && last != '-' && startsWord(runes, i) {
			buf.WriteRune('-')
		}

		buf.WriteRune(unicode.ToLower(r))
		last = r
	}

	return buf.String()
}

func startsWord(runes []rune, i int) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	prev := runes[i-1]
	if unicode.IsLower(prev) || unicode.IsDigit(prev) {
		return true
	}
	return unicode.IsUpper(prev) && lowerRunLen(runes[i+1:]) > 1
}

func lowerRunLen(runes []rune) int {
	for i, r := range runes {
		if !unicode.IsLower(r) {
			return i
		}
	}
	return len(runes)
}

func conversionError(c *cli.Context, f structField, value any) error {
	return &cli.InternalError{
		Path:   c.Path(),
		Timing: c.Timing(),
		Err: fmt.Errorf("cannot bind %q to field %s: expected %v, got %T",
			f.flagName, f.declared, f.typ, value),
	}
}

var (
	_ Binder[struct{}] = (*structBinder[struct{}])(nil)
	_ binderInit       = (*structBinder[struct{}])(nil)
)
