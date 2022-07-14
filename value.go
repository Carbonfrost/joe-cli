package cli

import (
	"encoding"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Value provides the interface for custom handling of arg and flag values.  This is the
// same as flag.Value.  Values can implement additional methods by convention which are called
// on the first occurrence of a value being set
//
// * DisableSplitting()        called when the option has set the DisableSplitting option, which
//                             indicates that commas shouldn't be treated as list separators
//
// * Reset()                   called on first occurrence of setting a value.  This can be used to reset lists
//                             to empty when the Merge option has not been set
//
// * NewCounter() ArgCounter   if provided, this method is consulted to obtain the arg counter if NArg is unset
//
// * Initializer() Action      obtains an initialization action for the value which is called after initialization
//                             of the flag or arg
//
// * Value() interface{}       obtains the actual value to return from a lookup, useful when flag.Value is a wrapper
//
// * Synopsis() string         obtains the synopsis text
//
// * SetData(io.Reader)error   read from a reader to set the value
//
// * Completion() Completion   called to obtain the default completion for a value
//
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
	Value() interface{}
}

type valueProvidesSynopsis interface {
	Synopsis() string
}

type valueSetData interface {
	SetData(io.Reader) error
}

type valueCompleter interface {
	Completion() Completion
}

type generic struct {
	p interface{}
}

type valuePairCounter struct {
	count int
}

type valueContext struct {
	v      *valueTarget
	name   string
	lookup *set
}

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

// UInt creates an uint value.  This is for convenience to obtain the right pointer.
func UInt() *uint {
	return new(uint)
}

// UInt8 creates an uint8 value.  This is for convenience to obtain the right pointer.
func UInt8() *uint8 {
	return new(uint8)
}

// UInt16 creates an uint16 value.  This is for convenience to obtain the right pointer.
func UInt16() *uint16 {
	return new(uint16)
}

// UInt32 creates an uint32 value.  This is for convenience to obtain the right pointer.
func UInt32() *uint32 {
	return new(uint32)
}

// UInt64 creates an uint64 value.  This is for convenience to obtain the right pointer.
func UInt64() *uint64 {
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
func Set(dest interface{}, args ...string) error {
	for _, arg := range args {
		err := setCore(dest, true, arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func trySetOptional(dest interface{}, trySetOptional func() (interface{}, bool)) bool {
	switch p := dest.(type) {
	case *bool:
		if v, ok := trySetOptional(); ok {
			*p = v.(bool)
			return true
		}
		return false
	case *string:
		if v, ok := trySetOptional(); ok {
			*p = v.(string)
			return true
		}
		return false
	case *[]string:
		if v, ok := trySetOptional(); ok {
			*p = v.([]string)
			return true
		}
		return false
	case *map[string]string:
		if v, ok := trySetOptional(); ok {
			*p = v.(map[string]string)
			return true
		}
		return false
	case *int:
		if v, ok := trySetOptional(); ok {
			*p = v.(int)
			return true
		}
		return false
	case *int8:
		if v, ok := trySetOptional(); ok {
			*p = v.(int8)
			return true
		}
		return false
	case *int16:
		if v, ok := trySetOptional(); ok {
			*p = v.(int16)
			return true
		}
		return false
	case *int32:
		if v, ok := trySetOptional(); ok {
			*p = v.(int32)
			return true
		}
		return false
	case *int64:
		if v, ok := trySetOptional(); ok {
			*p = v.(int64)
			return true
		}
		return false
	case *uint:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint)
			return true
		}
		return false
	case *uint8:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint8)
			return true
		}
		return false
	case *uint16:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint16)
			return true
		}
		return false
	case *uint32:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint32)
			return true
		}
		return false
	case *uint64:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint64)
			return true
		}
		return false
	case *float32:
		if v, ok := trySetOptional(); ok {
			*p = v.(float32)
			return true
		}
		return false
	case *float64:
		if v, ok := trySetOptional(); ok {
			*p = v.(float64)
			return true
		}
		return false
	case *time.Duration:
		if v, ok := trySetOptional(); ok {
			*p = v.(time.Duration)
			return true
		}
		return false
	case **url.URL:
		if v, ok := trySetOptional(); ok {
			*p = v.(*url.URL)
			return true
		}
		return false

	case *net.IP:
		if v, ok := trySetOptional(); ok {
			*p = v.(net.IP)
			return true
		}
		return false

	case **regexp.Regexp:
		if v, ok := trySetOptional(); ok {
			*p = v.(*regexp.Regexp)
			return true
		}
		return false
	case **big.Int:
		if v, ok := trySetOptional(); ok {
			*p = v.(*big.Int)
			return true
		}
		return false
	case **big.Float:
		if v, ok := trySetOptional(); ok {
			*p = v.(*big.Float)
			return true
		}
		return false
	}
	return false
}

// setCore sets the variable; no additional splitting is applied
func setCore(dest interface{}, disableSplitting bool, value string) error {
	strconvErr := func(err error) error {
		if e, ok := err.(*strconv.NumError); ok {
			switch e.Err {
			case strconv.ErrRange:
				err = fmt.Errorf("value out of range: %s", value)
			case strconv.ErrSyntax:
				err = fmt.Errorf("not a valid number: %s", value)
			}
		}
		return err
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
			s = s + " "
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
		var key, value string
		m := *p
		if m == nil {
			m = map[string]string{}
			*p = m
		}
		for _, kvp := range values() {
			k := SplitList(kvp, "=", 2)
			switch len(k) {
			case 2:
				key = k[0]
				value = k[1]
			case 1:
				// Implies comma was meant to be escaped
				// -m key=value,s,t  --> interpreted as key=value,s,t rather than s and t keys
				value = value + "," + k[0]
			}
			m[key] = value
		}

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

func (v *NameValue) Reset() {
	v.Name = ""
	v.Value = ""
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
func (n *NameValue) SetAllowFileReference(v bool) error {
	n.AllowFileReference = v
	return nil
}

func (n *NameValue) AllowFileReferencesFlag() Prototype {
	return Prototype{
		Name:     "allow-files",
		HelpText: "Allow a file to be specified with name=@file",
		Setup: Setup{
			Uses: Bind(n.SetAllowFileReference),
		},
	}
}

func (*NameValue) Initializer() Action {
	return AtTiming(ActionFunc(loadFileReference), BeforeTiming)
}

func loadFileReference(c *Context) error {
	if current, ok := c.Value("").(*NameValue); ok {
		s := current.Value
		if current.AllowFileReference && strings.HasPrefix(s, "@") {
			f, err := c.FS.Open(s[1:])
			if err != nil {
				return err
			}
			contents, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			current.Value = string(contents)

		} else {
			current.Value = s
		}
	}
	return nil
}

func (v *valuePairCounter) Done() error {
	switch v.count {
	case 0:
		return errors.New("missing name and value")
	}
	return nil
}

func (v *valuePairCounter) Take(arg string, possibleFlag bool) error {
	switch v.count {
	case 0:
		if _, _, hasValue := splitValuePair(arg); hasValue {
			v.count += 2
		} else {
			v.count += 1
		}
		return nil
	case 1:
		v.count += 1
		return nil
	case 2:
		v.count += 1
		return EndOfArguments
	}

	return errors.New("too many arguments to filter")
}

func (g *generic) Set(value string, opt *internalOption) error {
	if trySetOptional(g.p, func() (interface{}, bool) {
		return opt.optionalValue, (value == "" && opt.flags.optional())
	}) {
		return nil
	}

	return setCore(g.p, opt.flags.disableSplitting(), value)
}

func (g *generic) applyValueConventions(flags internalFlags, occurs int) {
	resetOnFirstOccur := !flags.merge()
	if occurs > 1 {
		// string will reset on every occurrence unless Merge is turned on
		if resetOnFirstOccur {
			switch p := g.p.(type) {
			case *string:
				*p = ""
			case valueResetOrMerge:
				p.Reset()
			}
		}
		return
	}

	if flags.disableSplitting() {
		if i, ok := g.p.(valueDisableSplitting); ok {
			i.DisableSplitting()
		}
	}

	if resetOnFirstOccur {
		switch p := g.p.(type) {
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

func (g *generic) smartOptionalDefault() interface{} {
	switch g.p.(type) {
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

func (v *valueContext) initialize(c *Context) error {
	return execute(c, v.v.uses().Initializers)
}

func (v *valueContext) executeBefore(ctx *Context) error {
	return execute(ctx, v.v.uses().Before)
}

func (v *valueContext) executeAfter(ctx *Context) error {
	return execute(ctx, v.v.uses().After)
}

func (v *valueContext) execute(ctx *Context) error {
	return execute(ctx, v.v.uses().Action)
}

func (v *valueContext) executeBeforeDescendent(ctx *Context) error { return nil }
func (v *valueContext) executeAfterDescendent(ctx *Context) error  { return nil }

func (v *valueContext) lookupBinding(name string, occurs bool) []string {
	if v.lookup == nil {
		return nil
	}
	return v.lookup.bindings.lookup(name, occurs)
}

func (v *valueContext) set() *set {
	return v.lookup
}

func (v *valueContext) setDidSubcommandExecute() {
}

func (v *valueContext) target() target {
	return v.v
}

func (v *valueContext) lookupValue(name string) (interface{}, bool) {
	if v.lookup == nil {
		return nil, false
	}
	return v.lookup.lookupValue(name)
}

func (v *valueContext) Name() string {
	return "<-" + v.name + ">"
}

func wrapGeneric(v interface{}) *generic {
	switch v.(type) {
	case Value:
		return &generic{v}
	case *bool:
		return &generic{v}
	case *string, *[]string:
		return &generic{v}
	case *int, *int8, *int16, *int32, *int64:
		return &generic{v}
	case *uint, *uint8, *uint16, *uint32, *uint64:
		return &generic{v}
	case *float32, *float64:
		return &generic{v}
	case *time.Duration:
		return &generic{v}
	case *map[string]string:
		return &generic{v}
	case *[]*NameValue:
		return &generic{v}
	case **url.URL:
		return &generic{v}
	case *net.IP:
		return &generic{v}
	case **regexp.Regexp:
		return &generic{v}
	case **big.Int:
		return &generic{v}
	case **big.Float:
		return &generic{v}
	case *[]byte:
		return &generic{v}
	case encoding.TextUnmarshaler:
		return &generic{v}
	default:
		panic(fmt.Sprintf("unsupported flag type: %T", v))
	}
}

// cloneZero creates a clone with the same type
func (g *generic) cloneZero() *generic {
	return wrapGeneric(func() interface{} {
		switch val := g.p.(type) {
		case *bool:
			return Bool()
		case *string:
			return String()
		case *[]string:
			return List()
		case *int:
			return Int()
		case *int8:
			return Int8()
		case *int16:
			return Int16()
		case *int32:
			return Int32()
		case *int64:
			return Int64()
		case *uint:
			return UInt()
		case *uint8:
			return UInt8()
		case *uint16:
			return UInt16()
		case *uint32:
			return UInt32()
		case *uint64:
			return UInt64()
		case *float32:
			return Float32()
		case *float64:
			return Float64()
		case *time.Duration:
			return Duration()
		case *map[string]string:
			return Map()
		case *[]*NameValue:
			return NameValues()
		case **url.URL:
			return URL()
		case *net.IP:
			return IP()
		case **regexp.Regexp:
			return Regexp()
		case **big.Int:
			return BigInt()
		case **big.Float:
			return BigFloat()
		case *[]byte:
			return Bytes()
		case valueResetOrMerge:
			return val
		}
		panic(fmt.Sprintf("unsupported flag type: %T", g.p))
	}())
}

func dereference(v interface{}) interface{} {
	if _, ok := v.(Value); ok {
		if d, ok := v.(valueDereference); ok {
			return d.Value()
		}
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
