// Package structure provides a flag Value which can be used to initialize a structure.
// Under the covers, mergo is used.  The default configuration supports string-based conversion
// from all the types that joe-cli supports built-in.
package structure

import (
	"encoding"
	"flag"
	"fmt"
	"math/big"
	"net/url"
	"reflect"
	"regexp"
	"time"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
	"github.com/mitchellh/mapstructure"
)

// Value is a value that uses structured initialization.  This allows arbitrary key value pairs
// to be used to initialize the inner value.  It uses the same semantics and syntax as cli.Map,
// including the short hand key-value pair syntax, a flag with multiple occurrences, and
// disabling splitting.
type Value struct {
	// V is the value that is actually initialized, a pointer to the struct.
	V interface{}

	// Options specifies the mapstructure options to use during the conversion.
	// If nil, some default options are specified that supports viable weakly typed
	// parsing.  To stop this, the options must be explicitly set to non-nil slice
	// (or some other custom decoder config)
	Options []DecoderOption

	disableSplitting bool
}

type DecoderOption func(*mapstructure.DecoderConfig)

var (
	valueType       = reflect.TypeOf((*cli.Value)(nil)).Elem()
	unmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	durationType    = reflect.TypeOf(time.Duration(0))
	bigIntType      = reflect.TypeOf(big.NewInt(0))
	bigFloatType    = reflect.TypeOf(big.NewFloat(0))
	listType        = reflect.TypeOf([]string{})
	mapType         = reflect.TypeOf(map[string]string{})
	regexpType      = reflect.TypeOf(&regexp.Regexp{})
	urlType         = reflect.TypeOf(&url.URL{})
)

// Of creates a Value which can be initialized using name-value pairs.  The argument v
// must be a pointer to a struct.   By default, a set of
// viable options provide conversions is also specified.  (To stop this,
// you must set or clear Options directly)
func Of(v interface{}) *Value {
	return &Value{
		V: v,
	}
}

// Decode will apply values to the given output
func Decode(input, output any, opts ...DecoderOption) error {
	config := &mapstructure.DecoderConfig{
		Result: output,
	}
	if opts == nil {
		opts = viableOptions()
	}
	for _, opt := range opts {
		opt(config)
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// WithOptions applies additional options.
func (v *Value) WithOptions(options ...DecoderOption) *Value {
	v.Options = append(v.Options, options...)
	return v
}

// Value obtains the inner value.
func (v *Value) Value() interface{} {
	return v.V
}

// DisableSplitting causes commas to be treated literally instead of as
// delimiters between values.
func (v *Value) DisableSplitting() {
	v.disableSplitting = true
}

// Set the text of the value.  Can be called successively to append.
func (v *Value) Set(arg string) error {
	text := arg
	var args []string

	if v.disableSplitting {
		args = []string{text}
	} else {
		args = cli.SplitList(text, ",", -1)
	}

	src := support.ParseMap(args)
	return Decode(src, v.V, v.Options...)
}

func (v *Value) String() string {
	switch val := v.V.(type) {
	case map[string]string:
		return support.FormatMap(val, ",")
	default:
		output := map[string]string{}
		_ = mapstructure.Decode(val, &output)
		return support.FormatMap(output, ",")
	}
}

func viableOptions() []DecoderOption {
	return []DecoderOption{
		func(m *mapstructure.DecoderConfig) {
			m.WeaklyTypedInput = true
			m.DecodeHook = mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToIPHookFunc(),
				mapstructure.StringToIPNetHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.DecodeHookFunc(bigIntHook),
				mapstructure.DecodeHookFunc(bigFloatHook),
				mapstructure.DecodeHookFunc(urlHook),
				mapstructure.DecodeHookFunc(regexpHook),
				mapstructure.DecodeHookFunc(valueHook),
				mapstructure.DecodeHookFunc(unmarshalerHook),
				mapstructure.RecursiveStructToMapHookFunc(),
			)
		},
	}
}

func bigIntHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String {
		return data, nil
	}
	if to != bigIntType {
		return data, nil
	}

	v := new(big.Int)
	if _, ok := v.SetString(data.(string), 10); !ok {
		return nil, fmt.Errorf("failed to parse big.Int")
	}
	return v, nil
}

func bigFloatHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to != bigFloatType {
		return data, nil
	}
	v, _, err := big.ParseFloat(data.(string), 10, 53, big.ToZero)
	return v, err
}

func urlHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to != urlType {
		return data, nil
	}
	return url.Parse(data.(string))
}

func regexpHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to != regexpType {
		return data, nil
	}
	return regexp.Compile(data.(string))
}

func valueHook(from, to reflect.Value) (any, error) {
	if from.Kind() != reflect.String {
		return from.Interface(), nil
	}

	if !to.Type().Implements(valueType) {
		return from.Interface(), nil
	}

	result := to.Interface()
	err := result.(cli.Value).Set(from.String())
	return result, err
}

func unmarshalerHook(from, to reflect.Value) (any, error) {
	if from.Kind() != reflect.String || to.Kind() != reflect.Struct {
		return from.Interface(), nil
	}

	// Generate a pointer to the value and check whether it is an unmarshaler
	indirect := reflect.New(to.Type())
	indirect.Elem().Set(to)

	if !indirect.Type().Implements(unmarshalerType) {
		return from.Interface(), nil
	}

	result := indirect.Interface()
	err := result.(encoding.TextUnmarshaler).UnmarshalText([]byte(from.String()))
	return result, err
}

var _ flag.Value = (*Value)(nil)
