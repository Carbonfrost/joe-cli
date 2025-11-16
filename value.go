// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cli

import (
	"bytes"
	"context"
	"encoding"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli/internal/support"
)

// Value provides the interface for custom handling of arg and flag values.  This is the
// same as flag.Value.  Values can implement additional methods by convention which are called
// on the first occurrence of a value being set
//
//   - DisableSplitting()        called when the option has set the DisableSplitting option, which
//     indicates that commas shouldn't be treated as list separators
//
//   - Reset()                   called on first occurrence of setting a value.  This can be used to reset lists
//     to empty when the Merge option has not been set
//
//   - Copy()                    when used in addition to Reset(), can be used to copy into a new value
//
//   - NewCounter() ArgCounter   if provided, this method is consulted to obtain the arg counter if NArg is unset
//
//   - Initializer() Action      obtains an initialization action for the value which is called after initialization
//     of the flag or arg
//
//   - Value() T                 obtains the actual value to return from a lookup, useful when flag.Value is a wrapper.
//     (the underlying value T is used)
//
//   - Synopsis() string         obtains the synopsis text
//
//   - SetData(io.Reader)error   read from a reader to set the value
//
//   - Completion() Completion   called to obtain the default completion for a value
type Value = flag.Value

//counterfeiter:generate . Value

// NameValue encapsulates a name-value pair.  This is a flag value specified
// using the syntax name=value.  When only the name is specified, this is interpreted
// as setting value to the constant true.  This allows disambiguating the syntax
// name= explicitly setting value to blank.
type NameValue struct {
	// Name in the name-value pair
	Name string
	// Value in the name-value pair
	Value string
	// AllowFileReference indicates whether the @file syntax is allowed for the value, which
	// is automatically loaded as a value.
	AllowFileReference bool
}

// Conventions for values

// ValueReader is a flag Value that can read from an input reader
type ValueReader interface {
	Value
	SetData(io.Reader) error
}

type valueDisableSplitting interface {
	DisableSplitting()
}

type valueResetOrMerge interface {
	Reset()
}

type valueProvidesCounter interface {
	NewCounter() ArgCounter
}

type valueInitializer interface {
	Initializer() Action
}

type valueDereference interface {
	Value() any
}

type valueCompleter interface {
	Completion() Completion
}

type valuePairCounter struct {
	count int
}

// BindingLookup can lookup values or their raw bindings
type BindingLookup interface {
	Lookup
	// Raw obtains values which were specified for a flag or arg
	// including the flag or arg name
	Raw(name string) []string

	// Raw obtains values which were specified for a flag or arg
	// but not including the flag or arg name
	RawOccurrences(name string) []string

	// Bindings obtains values which were specified for a flag or arg
	// including the flag or arg name and grouped into occurrences.
	Bindings(name string) [][]string

	// BindingNames obtains the names of the flags/args which are available.
	// Even if it is available, the empty string "" is not returned from this list.
	BindingNames() []string
}

type valueContext struct {
	v      *valueTarget
	lookup BindingLookup
}

var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9@#+\._\*:-]+$`)

// Bool creates a bool value.  This is for convenience to obtain the right pointer.
func Bool() *bool {
	return new(bool)
}

// String creates a string value.  This is for convenience to obtain the right pointer.
func String() *string {
	return new(string)
}

// List creates a string slice value.  This is for convenience to obtain the right pointer.
func List() *[]string {
	return new([]string)
}

// Int creates an int value.  This is for convenience to obtain the right pointer.
func Int() *int {
	return new(int)
}

// Int8 creates an int8 value.  This is for convenience to obtain the right pointer.
func Int8() *int8 {
	return new(int8)
}

// Int16 creates an int16 value.  This is for convenience to obtain the right pointer.
func Int16() *int16 {
	return new(int16)
}

// Int32 creates an int32 value.  This is for convenience to obtain the right pointer.
func Int32() *int32 {
	return new(int32)
}

// Int64 creates an int64 value.  This is for convenience to obtain the right pointer.
func Int64() *int64 {
	return new(int64)
}

// Uint creates an uint value.  This is for convenience to obtain the right pointer.
func Uint() *uint {
	return new(uint)
}

// Uint8 creates an uint8 value.  This is for convenience to obtain the right pointer.
func Uint8() *uint8 {
	return new(uint8)
}

// Uint16 creates an uint16 value.  This is for convenience to obtain the right pointer.
func Uint16() *uint16 {
	return new(uint16)
}

// Uint32 creates an uint32 value.  This is for convenience to obtain the right pointer.
func Uint32() *uint32 {
	return new(uint32)
}

// Uint64 creates an uint64 value.  This is for convenience to obtain the right pointer.
func Uint64() *uint64 {
	return new(uint64)
}

// Float32 creates a float32 value.  This is for convenience to obtain the right pointer.
func Float32() *float32 {
	return new(float32)
}

// Float64 creates a float64 value.  This is for convenience to obtain the right pointer.
func Float64() *float64 {
	return new(float64)
}

// Duration creates a time.Duration value.  This is for convenience to obtain the right pointer.
func Duration() *time.Duration {
	return new(time.Duration)
}

// Map creates a map value.  This is for convenience to obtain the right pointer.
func Map() *map[string]string {
	return new(map[string]string)
}

// URL creates a URL value.  This is for convenience to obtain the right pointer.
func URL() **url.URL {
	return new(*url.URL)
}

// Regexp creates a Regexp value.  This is for convenience to obtain the right pointer.
func Regexp() **regexp.Regexp {
	return new(*regexp.Regexp)
}

// IP creates an IP value.  This is for convenience to obtain the right pointer.
func IP() *net.IP {
	return new(net.IP)
}

// BigInt creates a big integer value.  This is for convenience to obtain the right pointer.
func BigInt() **big.Int {
	return new(*big.Int)
}

// BigFloat creates a big float value.  This is for convenience to obtain the right pointer.
func BigFloat() **big.Float {
	return new(*big.Float)
}

// Bytes creates a slice of bytes.  This is for convenience to obtain the right pointer.
func Bytes() *[]byte {
	return new([]byte)
}

// NameValues creates a list of name-value pairs, optionally specifying the values to
// set
func NameValues(namevalue ...string) *[]*NameValue {
	if len(namevalue)%2 != 0 {
		panic("unexpected number of arguments")
	}

	res := make([]*NameValue, 0, len(namevalue)/2)
	for i := 0; i < len(namevalue); i += 2 {
		res = append(res, &NameValue{
			Name:  namevalue[i],
			Value: namevalue[i+1],
		})
	}
	return &res
}

// Set will set the destination value if supported.  If the destination value is not supported,
// this panics.  See the overview for Value for which destination types are supported.
// No additional splitting is performed on arguments.
func Set(dest any, args ...string) error {
	for _, arg := range args {
		err := setCore(dest, true, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetData sets the value of a flag Value using the semantics
// of SetData(io.Reader), which is a convention that can be implemented
// by a value (see the summary on Value for information about conventions).
// In particular, in argument can be string, []byte, or io.Reader.
// If the method convention is not implemented, then ordinary Set(string)
// method on Value is called on the input.
func SetData(dest any, arg any) error {
	if s, ok := dest.(ValueReader); ok {
		switch val := arg.(type) {
		case string:
			return s.SetData(strings.NewReader(val))
		case io.Reader:
			return s.SetData(val)
		case []byte:
			return s.SetData(bytes.NewReader(val))
		}
	}

	if s, ok := dest.(*[]byte); ok {
		switch val := arg.(type) {
		case io.Reader:
			buf := bytes.NewBuffer(*s)
			if _, err := io.Copy(buf, val); err != nil {
				return err
			}
			*s = buf.Bytes()
			return nil

		case []byte:
			buf := bytes.NewBuffer(*s)
			buf.Write(val)
			*s = buf.Bytes()
			return nil
		}
	}

	switch val := arg.(type) {
	case string:
		return Set(dest, val)
	case io.Reader:
		bb, err := io.ReadAll(val)
		if err != nil {
			return err
		}
		return Set(dest, string(bb))
	case []byte:
		return Set(dest, string(val))
	}

	panic(fmt.Sprintf("unexpected argument type %T", arg))
}

func trySetOptional(dest any, trySetOptional func() (any, bool)) bool {
	if _, ok := dest.(Value); ok {
		return false
	}
	if v, ok := trySetOptional(); ok {
		setDirect(dest, v)
		return true
	}
	return false
}

// setCore sets the variable; no additional splitting is applied
func setCore(dest any, disableSplitting bool, value string) error {
	strconvErr := func(err error) error {
		return formatStrconvError(err, value)
	}
	values := func() []string {
		if disableSplitting {
			return []string{value}
		}
		return SplitList(value, ",", -1)
	}
	switch p := dest.(type) {
	case Value:
		return p.Set(value)
	case *bool:
		var err error
		*p, err = parseBool(value)
		if err != nil {
			return err
		}
		return nil
	case *string:
		s := *p
		if len(s) > 0 {
			s += " "
		}
		*p = s + value
		return nil
	case *[]string:
		*p = append(*p, values()...)
		return nil
	case *[]byte:
		bb, err := hex.DecodeString(value)
		if err != nil {
			return fmt.Errorf("invalid bytes: %s", err)
		}
		*p = append(*p, bb...)
		return nil
	case *map[string]string:
		m := *p
		if m == nil {
			m = map[string]string{}
			*p = m
		}
		maps.Copy(m, support.ParseMap(values()))

		return nil
	case *[]*NameValue:
		for _, kvp := range values() {
			nvp := new(NameValue)
			var hasValue bool

			nvp.Name, nvp.Value, hasValue = splitValuePair(kvp)
			if !hasValue {
				return fmt.Errorf("value required for %s", nvp.Name)
			}
			*p = append(*p, nvp)
		}
		return nil

	case *int:
		i64, err := strconv.ParseInt(value, 0, strconv.IntSize)
		if err == nil {
			*p = int(i64)
		}
		return strconvErr(err)
	case *int8:
		i64, err := strconv.ParseInt(value, 0, 8)
		if err == nil {
			*p = int8(i64)
		}
		return strconvErr(err)
	case *int16:
		i64, err := strconv.ParseInt(value, 0, 16)
		if err == nil {
			*p = int16(i64)
		}
		return strconvErr(err)
	case *int32:
		i64, err := strconv.ParseInt(value, 0, 32)
		if err == nil {
			*p = int32(i64)
		}
		return strconvErr(err)
	case *int64:
		i64, err := strconv.ParseInt(value, 0, 64)
		if err == nil {
			*p = i64
		}
		return strconvErr(err)
	case *uint:
		u64, err := strconv.ParseUint(value, 0, strconv.IntSize)
		if err == nil {
			*p = uint(u64)
		}
		return strconvErr(err)
	case *uint8:
		u64, err := strconv.ParseUint(value, 0, 8)
		if err == nil {
			*p = uint8(u64)
		}
		return strconvErr(err)
	case *uint16:
		u64, err := strconv.ParseUint(value, 0, 16)
		if err == nil {
			*p = uint16(u64)
		}
		return strconvErr(err)
	case *uint32:
		u64, err := strconv.ParseUint(value, 0, 32)
		if err == nil {
			*p = uint32(u64)
		}
		return strconvErr(err)
	case *uint64:
		u64, err := strconv.ParseUint(value, 0, 64)
		if err == nil {
			*p = u64
		}
		return strconvErr(err)
	case *float32:
		f64, err := strconv.ParseFloat(value, 32)
		if err == nil {
			*p = float32(f64)
		}
		return strconvErr(err)
	case *float64:
		f64, err := strconv.ParseFloat(value, 64)
		if err == nil {
			*p = f64
		}
		return strconvErr(err)
	case *time.Duration:
		v, err := time.ParseDuration(value)
		if err == nil {
			*p = v
		}
		return err
	case **url.URL:
		v, err := url.Parse(value)
		if err == nil {
			*p = v
		}
		return err

	case *net.IP:
		v := net.ParseIP(value)
		if v != nil {
			*p = v
			return nil
		}
		return errors.New("not a valid IP address")

	case **regexp.Regexp:
		v, err := regexp.Compile(value)
		if err == nil {
			*p = v
		}
		return err
	case **big.Int:
		v := new(big.Int)
		if _, ok := v.SetString(value, 10); ok {
			*p = v
			return nil
		}
		return strconvErr(errors.New("conversion failed"))
	case **big.Float:
		v, _, err := big.ParseFloat(value, 10, 53, big.ToZero)
		if err == nil {
			*p = v
		}
		return strconvErr(err)
	case encoding.TextUnmarshaler:
		return p.UnmarshalText([]byte(value))
	}
	panic(fmt.Sprintf("unsupported flag type: %T", dest))
}

func setDirect(dest any, v any) error {
	switch p := dest.(type) {
	case *bool:
		*p = v.(bool)
	case *string:
		*p = v.(string)
	case *[]string:
		*p = v.([]string)
	case *map[string]string:
		*p = v.(map[string]string)
	case *int:
		*p = v.(int)
	case *int8:
		*p = v.(int8)
	case *int16:
		*p = v.(int16)
	case *int32:
		*p = v.(int32)
	case *int64:
		*p = v.(int64)
	case *uint:
		*p = v.(uint)
	case *uint8:
		*p = v.(uint8)
	case *uint16:
		*p = v.(uint16)
	case *uint32:
		*p = v.(uint32)
	case *uint64:
		*p = v.(uint64)
	case *float32:
		*p = v.(float32)
	case *float64:
		*p = v.(float64)
	case *time.Duration:
		*p = v.(time.Duration)
	case **url.URL:
		*p = v.(*url.URL)
	case *net.IP:
		*p = v.(net.IP)
	case **regexp.Regexp:
		*p = v.(*regexp.Regexp)
	case **big.Int:
		*p = v.(*big.Int)
	case **big.Float:
		*p = v.(*big.Float)
	case *[]byte:
		*p = v.([]byte)
	default:
		panic(fmt.Sprintf("cannot set value directly: %T %v", dest, v))
	}
	return nil
}

func (v *NameValue) Reset() {
	// Don't reset AllowFileReference because it is configuration
	v.Name = ""
	v.Value = ""
}

func (v *NameValue) Copy() *NameValue {
	res := *v
	return &res
}

func (v *NameValue) Set(arg string) error {
	if v.Name == "" {
		v.Name, v.Value, _ = splitValuePair(arg)
	} else {
		v.Value = arg
	}
	return nil
}

// String obtains the string representation of the name-value pair
func (v *NameValue) String() string {
	return Quote(v.Name + "=" + v.Value)
}

func (v *NameValue) NewCounter() ArgCounter {
	return &valuePairCounter{}
}

// SetAllowFileReference sets whether file references are allowed.  This function is for
// bindings
func (v *NameValue) SetAllowFileReference(val bool) error {
	v.AllowFileReference = val
	return nil
}

// AllowFileReferencesFlag obtains a flag configuration for setting the
// AllowFileReferences value
func (v *NameValue) AllowFileReferencesFlag() Prototype {
	return Prototype{
		Name:     "allow-files",
		HelpText: "Allow a file to be specified with name=@file",
		Setup: Setup{
			Uses: Bind(v.SetAllowFileReference),
		},
	}
}

func (v *NameValue) Initializer() Action {
	if v.AllowFileReference {
		return actionFunc(func(c context.Context) error {
			return Do(c, ValueTransform(TransformOptionalFileReference(FromContext(c).actualFS())))
		})
	}
	return nil
}

func (v *valuePairCounter) Done() error {
	if v.count == 0 {
		return errors.New("missing name and value")
	}
	return nil
}

func (v *valuePairCounter) Take(arg string, possibleFlag bool) error {
	if possibleFlag && strings.HasPrefix(arg, "-") {
		return EndOfArguments
	}
	switch v.count {
	case 0:
		if _, _, hasValue := splitValuePair(arg); hasValue {
			v.count += 2
		} else {
			v.count++
		}
		return nil
	case 1:
		v.count++
		return nil
	case 2:
		v.count++
		return EndOfArguments
	}

	return errors.New("too many arguments to filter")
}

func (v *valueContext) Raw(name string) []string {
	if v.lookup == nil {
		return nil
	}
	return v.lookup.Raw(name)
}

func (v *valueContext) RawOccurrences(name string) []string {
	if v.lookup == nil {
		return nil
	}
	return v.lookup.RawOccurrences(name)
}

func (v *valueContext) Bindings(name string) [][]string {
	if v.lookup == nil {
		return nil
	}
	return v.lookup.Bindings(name)
}

func (v *valueContext) BindingNames() []string {
	if v.lookup == nil {
		return nil
	}
	return v.lookup.BindingNames()
}

func (v *valueContext) lookupValue(name string) (any, bool) {
	if v.lookup == nil {
		return nil, false
	}
	return v.lookup.Interface(name)
}

func checkValidIdentifier(name string) error {
	if !validIdentifierPattern.MatchString(name) {
		return fmt.Errorf("not a valid name")
	}
	return nil
}

func checkSupportedFlagType(v any) error {
	switch v.(type) {
	case Value:
		return nil
	case *bool:
		return nil
	case *string, *[]string:
		return nil
	case *int, *int8, *int16, *int32, *int64:
		return nil
	case *uint, *uint8, *uint16, *uint32, *uint64:
		return nil
	case *float32, *float64:
		return nil
	case *time.Duration:
		return nil
	case *map[string]string:
		return nil
	case *[]*NameValue:
		return nil
	case **url.URL:
		return nil
	case *net.IP:
		return nil
	case **regexp.Regexp:
		return nil
	case **big.Int:
		return nil
	case **big.Float:
		return nil
	case *[]byte:
		return nil
	case encoding.TextUnmarshaler:
		return nil
	}
	return fmt.Errorf("unsupported flag type: %T", v)
}

func dereference(v any) any {
	if _, ok := v.(Value); ok {
		if d, ok := v.(valueDereference); ok {
			return d.Value()
		}
		if d, ok := v.(flag.Getter); ok {
			return d.Get()
		}

		// Reflection to detect typed value dereference
		selector := reflect.ValueOf(v)
		if valueMethod, ok := selector.Type().MethodByName("Value"); ok {
			if valueMethod.Type.NumIn() == 1 && valueMethod.Type.NumOut() == 1 {
				out := valueMethod.Func.Call([]reflect.Value{selector})
				return out[0].Interface()
			}
		}
		return v
	}
	// Don't dereference built-in types twice
	switch v.(type) {
	case *url.URL, *regexp.Regexp, *big.Int, *big.Float:
		return v
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		return val.Elem().Interface()
	}
	return v
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {

	case "", "1", "true", "on", "t":
		return true, nil
	case "0", "false", "off", "f":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value for bool %q", value)
	}
}

func splitValuePair(arg string) (k, v string, hasValue bool) {
	a := SplitList(arg, "=", 2)
	if len(a) == 1 {
		return a[0], "true", false
	}
	return a[0], a[1], true
}

func optionReset(p any, flags internalFlags) {
	// Unless merge was explicitly requested, resetting the option does not apply merge rules
	if !flags.mergeExplicitlyRequested() {
		flags &^= internalFlagMerge
	}
	optionApplyValueConventions(p, flags, true)
}

func optionApplyValueConventions(p any, flags internalFlags, firstOccur bool) {
	resetOnFirstOccur := !flags.merge()
	if !firstOccur {
		// string will reset on every occurrence unless Merge is turned on
		if resetOnFirstOccur {
			switch p := p.(type) {
			case *string:
				*p = ""
			case valueResetOrMerge:
				p.Reset()
			}
		}
		return
	}

	if flags.disableSplitting() {
		if i, ok := p.(valueDisableSplitting); ok {
			i.DisableSplitting()
		}
	}

	if resetOnFirstOccur {
		switch p := p.(type) {
		case valueResetOrMerge:
			p.Reset()

		case *string:
			*p = ""

		case *[]string:
			*p = nil

		case *[]*NameValue:
			*p = []*NameValue{}

		case *map[string]string:
			*p = map[string]string{}
		}
	}
}

func optionSetOccurrence(o option, values ...string) error {
	o.nextOccur()
	for _, arg := range values {
		err := o.Set(arg)
		if err != nil {
			return err
		}
	}
	return nil
}

// valueCloneZero creates a clone with the same type
func valueCloneZero(p any) any {
	switch p.(type) {
	case *[]*NameValue:
		return NameValues()
	}
	return support.MustValueCloneZero(p)
}

func valueSmartOptionalDefault(v any) any {
	switch v.(type) {
	case *bool:
		return true
	case *int:
		return int(1)
	case *int8:
		return int8(1)
	case *int16:
		return int16(1)
	case *int32:
		return int32(1)
	case *int64:
		return int64(1)
	case *uint:
		return uint(1)
	case *uint8:
		return uint8(1)
	case *uint16:
		return uint16(1)
	case *uint32:
		return uint32(1)
	case *uint64:
		return uint64(1)
	case *float32:
		return float32(1)
	case *float64:
		return float64(1)
	case *time.Duration:
		return time.Second
	case *net.IP:
		return net.ParseIP("127.0.0.1")
	}
	return nil
}

var (
	_ internalContext = (*valueContext)(nil)
)
