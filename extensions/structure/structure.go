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
	Options []func(*mapstructure.DecoderConfig)

	disableSplitting bool
}

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

// WithOptions applies additional options.
func (v *Value) WithOptions(options ...func(*mapstructure.DecoderConfig)) *Value {
	if v.Options == nil {
		v.Options = make([]func(*mapstructure.DecoderConfig), 0)
	}
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

	src := parseMap(args)
	decoder, err := v.newDecoder()
	if err != nil {
		return err
	}
	return decoder.Decode(src)
}

func (v *Value) newDecoder() (*mapstructure.Decoder, error) {
	config := &mapstructure.DecoderConfig{
		Result: v.V,
	}
	opts := v.Options
	if opts == nil {
		opts = viableOptions()
	}
	for _, opt := range opts {
		opt(config)
	}
	return mapstructure.NewDecoder(config)
}

// parseMap is a clone of the logic in generic value
func parseMap(values []string) map[string]interface{} {
	res := map[string]interface{}{}

	var key, value string
	for _, kvp := range values {
		k := cli.SplitList(kvp, "=", 2)
		switch len(k) {
		case 2:
			key = k[0]
			value = k[1]
		case 1:
			// Implies comma was meant to be escaped
			// -m key=value,s,t  --> interpreted as key=value,s,t rather than s and t keys
			value = value + "," + k[0]
		}
		res[key] = value
	}
	return res
}

func (v *Value) String() string {
	return ""
}

func viableOptions() []func(*mapstructure.DecoderConfig) {
	return []func(*mapstructure.DecoderConfig){
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
		if typ == bigIntType {
			v := new(big.Int)
			if _, ok := v.SetString(value, 10); ok {
				return v, nil
			}

		} else if typ == bigFloatType {
			v, _, err := big.ParseFloat(value, 10, 53, big.ToZero)
			return v, err

		} else if typ == regexpType {
			return regexp.Compile(value)

		} else if typ == urlType {
			return url.Parse(value)

		} else if typ.Implements(valueType) {
			to.Interface().(cli.Value).Set(value)
			return to.Interface(), nil
		}

	case reflect.Map:
		if typ == mapType {
			m := parseMap(cli.SplitList(value, ",", -1))
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
