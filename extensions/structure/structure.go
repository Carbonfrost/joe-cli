// Package structure provides a flag Value which can be used to initialize a structure.
// Under the covers, mergo is used.  The default configuration supports string-based conversion
// from all the types that joe-cli supports built-in.
package structure

import (
	"flag"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
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
	valueType    = reflect.TypeOf((*cli.Value)(nil)).Elem()
	durationType = reflect.TypeOf(time.Duration(0))
	bigIntType   = reflect.TypeOf(big.NewInt(0))
	bigFloatType = reflect.TypeOf(big.NewFloat(0))
	ipType       = reflect.TypeOf(net.IP{})
	listType     = reflect.TypeOf([]string{})
	mapType      = reflect.TypeOf(map[string]string{})
	regexpType   = reflect.TypeOf(&regexp.Regexp{})
	urlType      = reflect.TypeOf(&url.URL{})
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
	return ""
}

func viableOptions() []DecoderOption {
	return []DecoderOption{
		func(m *mapstructure.DecoderConfig) {
			m.WeaklyTypedInput = true
			m.DecodeHook = mapstructure.DecodeHookFuncValue(applyConversions)
		},
	}
}

func applyConversions(from, to reflect.Value) (interface{}, error) {
	typ := to.Type()
	value := from.String()

	// In the case of failure, let conversions fallthrough to the default
	// in the decoder
	switch typ.Kind() {
	case reflect.Int64:
		if typ == durationType {
			return time.ParseDuration(value)
		}
		return strconv.ParseInt(value, 0, 64)

	case reflect.Ptr:
		switch typ {
		case bigIntType:
			v := new(big.Int)
			if _, ok := v.SetString(value, 10); ok {
				return v, nil
			}

		case bigFloatType:
			v, _, err := big.ParseFloat(value, 10, 53, big.ToZero)
			return v, err

		case regexpType:
			return regexp.Compile(value)

		case urlType:
			return url.Parse(value)

		default:
			if typ.Implements(valueType) {
				to.Interface().(cli.Value).Set(value)
				return to.Interface(), nil
			}
		}

	case reflect.Map:
		if typ == mapType {
			m := support.ParseMap(cli.SplitList(value, ",", -1))
			if to.IsNil() {
				to = reflect.MakeMap(mapType)
			}
			for k, v := range m {
				to.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
			return to.Interface(), nil
		}

	case reflect.Slice:
		if typ == listType {
			return cli.SplitList(value, ",", -1), nil

		} else if typ == ipType {
			v := net.ParseIP(value)
			if v != nil {
				return v, nil
			}
		}
	}
	return from.Interface(), nil
}

var _ flag.Value = (*Value)(nil)
