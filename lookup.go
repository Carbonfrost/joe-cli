package cli

import (
	"fmt"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Lookup provides type conversion from the various built-in types supported by
// the framework. For each method, the name is either a string, rune, *Arg, or *Flag
// corresponding to the name of the argument or flag.
type Lookup interface {
	// Bool obtains the value and converts it to a bool
	Bool(name interface{}) bool
	// File obtains the value and converts it to a File
	File(name interface{}) *File
	// FileSet obtains the value and converts it to a FileSet
	FileSet(name interface{}) *FileSet
	// Float32 obtains the value and converts it to a float32
	Float32(name interface{}) float32
	// Float64 obtains the value and converts it to a float64
	Float64(name interface{}) float64
	// Int obtains the value and converts it to a int
	Int(name interface{}) int
	// Int16 obtains the value and converts it to a int16
	Int16(name interface{}) int16
	// Int32 obtains the value and converts it to a int32
	Int32(name interface{}) int32
	// Int64 obtains the value and converts it to a int64
	Int64(name interface{}) int64
	// Int8 obtains the value and converts it to a int8
	Int8(name interface{}) int8
	// Duration obtains the value and converts it to a Duration
	Duration(name interface{}) time.Duration
	// List obtains the value and converts it to a slice of strings
	List(name interface{}) []string
	// Map obtains the value and converts it to a map
	Map(name interface{}) map[string]string
	// NameValue obtains the value and converts it to a name-value pair
	NameValue(name interface{}) *NameValue
	// NameValues obtains the value and converts it to a list of name-value pairs
	NameValues(name interface{}) []*NameValue
	// String obtains the value and converts it to a string
	String(name interface{}) string
	// Uint obtains the value and converts it to a uint
	Uint(name interface{}) uint
	// Uint16 obtains the value and converts it to a uint16
	Uint16(name interface{}) uint16
	// Uint32 obtains the value and converts it to a uint32
	Uint32(name interface{}) uint32
	// Uint64 obtains the value and converts it to a uint64
	Uint64(name interface{}) uint64
	// Uint8 obtains the value and converts it to a uint8
	Uint8(name interface{}) uint8
	// Value obtains the value and converts it to Value
	Value(name interface{}) interface{}
	// URL obtains the value and converts it to a URL
	URL(name interface{}) *url.URL
	// Regexp obtains the value and converts it to a Regexp
	Regexp(name interface{}) *regexp.Regexp
	// IP obtains the value and converts it to a IP
	IP(name interface{}) net.IP
	// BigInt obtains the value and converts it to a BigInt
	BigInt(name interface{}) *big.Int
	// BigFloat obtains the value and converts it to a BigFloat
	BigFloat(name interface{}) *big.Float
	// Bytes obtains the value and converts it to a slice of bytes
	Bytes(name interface{}) []byte
	// Interface returns the raw value without dereferencing and whether it exists
	Interface(name interface{}) (interface{}, bool)
}

// LookupValues provides a Lookup backed by a map
type LookupValues map[string]interface{}

// LookupFunc provides a Lookup that converts from a name
// to a value.
type LookupFunc func(string) (any, bool)

type lookupCore interface {
	// lookupValue, no need to dereference
	lookupValue(name string) (interface{}, bool)
}

// Value obtains the value and converts it to Value
func (c LookupValues) Value(name interface{}) interface{} {
	r, _ := c.try(name, true)
	return r
}

func (c LookupValues) try(name interface{}, deref bool) (interface{}, bool) {
	if c == nil {
		return nil, false
	}

	actual, ok := c[nameToString(name)]
	if !ok {
		return nil, false
	}

	if deref {
		return dereference(actual), true
	}
	return actual, true
}

// Bool obtains the value and converts it to a bool
func (c LookupValues) Bool(name interface{}) bool {
	return lookupBool(c, name)
}

// String obtains the value and converts it to a string
func (c LookupValues) String(name interface{}) string {
	return lookupString(c, name)
}

// List obtains the value and converts it to a string slice
func (c LookupValues) List(name interface{}) []string {
	return lookupList(c, name)
}

// Int obtains the int for the specified name
func (c LookupValues) Int(name interface{}) int {
	return lookupInt(c, name)
}

// Int8 obtains the int8 for the specified name
func (c LookupValues) Int8(name interface{}) int8 {
	return lookupInt8(c, name)
}

// Int16 obtains the int16 for the specified name
func (c LookupValues) Int16(name interface{}) int16 {
	return lookupInt16(c, name)
}

// Int32 obtains the int32 for the specified name
func (c LookupValues) Int32(name interface{}) int32 {
	return lookupInt32(c, name)
}

// Int64 obtains the int64 for the specified name
func (c LookupValues) Int64(name interface{}) int64 {
	return lookupInt64(c, name)
}

// Uint obtains the uint for the specified name
func (c LookupValues) Uint(name interface{}) uint {
	return lookupUint(c, name)
}

// Uint8 obtains the uint8 for the specified name
func (c LookupValues) Uint8(name interface{}) uint8 {
	return lookupUint8(c, name)
}

// Uint16 obtains the uint16 for the specified name
func (c LookupValues) Uint16(name interface{}) uint16 {
	return lookupUint16(c, name)
}

// Uint32 obtains the uint32 for the specified name
func (c LookupValues) Uint32(name interface{}) uint32 {
	return lookupUint32(c, name)
}

// Uint64 obtains the uint64 for the specified name
func (c LookupValues) Uint64(name interface{}) uint64 {
	return lookupUint64(c, name)
}

// Float32 obtains the float32 for the specified name
func (c LookupValues) Float32(name interface{}) float32 {
	return lookupFloat32(c, name)
}

// Float64 obtains the float64 for the specified name
func (c LookupValues) Float64(name interface{}) float64 {
	return lookupFloat64(c, name)
}

// Duration obtains the Duration for the specified name
func (c LookupValues) Duration(name interface{}) time.Duration {
	return lookupDuration(c, name)
}

// File obtains the File for the specified name
func (c LookupValues) File(name interface{}) *File {
	return lookupFile(c, name)
}

// FileSet obtains the FileSet for the specified name
func (c LookupValues) FileSet(name interface{}) *FileSet {
	return lookupFileSet(c, name)
}

// Map obtains the map for the specified name
func (c LookupValues) Map(name interface{}) map[string]string {
	return lookupMap(c, name)
}

// NameValue obtains the value and converts it to a name-value pair
func (c LookupValues) NameValue(name interface{}) *NameValue {
	return lookupNameValue(c, name)
}

// NameValues obtains the value and converts it to a list of name-value pairs
func (c LookupValues) NameValues(name interface{}) []*NameValue {
	return lookupNameValues(c, name)
}

// URL obtains the URL for the specified name
func (c LookupValues) URL(name interface{}) *url.URL {
	return lookupURL(c, name)
}

// Regexp obtains the Regexp for the specified name
func (c LookupValues) Regexp(name interface{}) *regexp.Regexp {
	return lookupRegexp(c, name)
}

// IP obtains the IP for the specified name
func (c LookupValues) IP(name interface{}) net.IP {
	return lookupIP(c, name)
}

// BigInt obtains the BigInt for the specified name
func (c LookupValues) BigInt(name interface{}) *big.Int {
	return lookupBigInt(c, name)
}

// BigFloat obtains the BigFloat for the specified name
func (c LookupValues) BigFloat(name interface{}) *big.Float {
	return lookupBigFloat(c, name)
}

// Bytes obtains the bytes for the specified name
func (c LookupValues) Bytes(name interface{}) []byte {
	return lookupBytes(c, name)
}

// Interface obtains the raw value without dereferencing
func (c LookupValues) Interface(name interface{}) (interface{}, bool) {
	return c.try(name, false)
}

func (c LookupFunc) Bool(name any) bool {
	return lookupBool(c, name)
}

func (c LookupFunc) String(name any) string {
	return lookupString(c, name)
}

func (c LookupFunc) List(name any) []string {
	return lookupList(c, name)
}

func (c LookupFunc) Int(name any) int {
	return lookupInt(c, name)
}

func (c LookupFunc) Int8(name any) int8 {
	return lookupInt8(c, name)
}

func (c LookupFunc) Int16(name any) int16 {
	return lookupInt16(c, name)
}

func (c LookupFunc) Int32(name any) int32 {
	return lookupInt32(c, name)
}

func (c LookupFunc) Int64(name any) int64 {
	return lookupInt64(c, name)
}

func (c LookupFunc) Uint(name any) uint {
	return lookupUint(c, name)
}

func (c LookupFunc) Uint8(name any) uint8 {
	return lookupUint8(c, name)
}

func (c LookupFunc) Uint16(name any) uint16 {
	return lookupUint16(c, name)
}

func (c LookupFunc) Uint32(name any) uint32 {
	return lookupUint32(c, name)
}

func (c LookupFunc) Uint64(name any) uint64 {
	return lookupUint64(c, name)
}

func (c LookupFunc) Float32(name any) float32 {
	return lookupFloat32(c, name)
}

func (c LookupFunc) Float64(name any) float64 {
	return lookupFloat64(c, name)
}

func (c LookupFunc) Duration(name any) time.Duration {
	return lookupDuration(c, name)
}

func (c LookupFunc) File(name any) *File {
	return lookupFile(c, name)
}

func (c LookupFunc) FileSet(name any) *FileSet {
	return lookupFileSet(c, name)
}

func (c LookupFunc) Map(name any) map[string]string {
	return lookupMap(c, name)
}

func (c LookupFunc) NameValue(name any) *NameValue {
	return lookupNameValue(c, name)
}

func (c LookupFunc) NameValues(name any) []*NameValue {
	return lookupNameValues(c, name)
}

func (c LookupFunc) URL(name any) *url.URL {
	return lookupURL(c, name)
}

func (c LookupFunc) Regexp(name any) *regexp.Regexp {
	return lookupRegexp(c, name)
}

func (c LookupFunc) IP(name any) net.IP {
	return lookupIP(c, name)
}

func (c LookupFunc) BigInt(name any) *big.Int {
	return lookupBigInt(c, name)
}

func (c LookupFunc) BigFloat(name any) *big.Float {
	return lookupBigFloat(c, name)
}

func (c LookupFunc) Bytes(name any) []byte {
	return lookupBytes(c, name)
}

func (c LookupFunc) Interface(name any) (any, bool) {
	return c.try(name, false)
}

func (c LookupFunc) Value(name any) any {
	r, _ := c.try(name, true)
	return r
}

func (c LookupFunc) try(n any, deref bool) (any, bool) {
	return tryLookup(c, n, deref)
}

func tryLookup(c lookupCore, n any, deref bool) (any, bool) {
	if c == nil {
		return nil, false
	}
	name := nameToString(n)

	// Strip possible decorators --flag, <arg>
	name = withoutDecorators(name)
	if v, ok := c.lookupValue(name); ok {
		if deref {
			return dereference(v), true
		}
		return v, true
	}
	return nil, false
}

func (p *parentLookup) lookupValue(name string) (interface{}, bool) {
	if v, ok := p.lookupCore.lookupValue(name); ok {
		return v, true
	}

	return p.parent.lookupValue(name)
}

func (c LookupFunc) lookupValue(name string) (any, bool) {
	if c == nil {
		return nil, false
	}
	return c(name)
}

func nameToString(name interface{}) string {
	switch v := name.(type) {
	case rune:
		return string(v)
	case string:
		return v
	case nil:
		return ""
	case *Arg:
		return v.Name
	case *Flag:
		return v.Name
	default:
		panic(fmt.Sprintf("unexpected type: %T", name))
	}
}

func withoutDecorators(name string) string {
	return strings.Trim(name, "-<>")
}

func lookupBool(c Lookup, name interface{}) (res bool) {
	val := c.Value(name)
	if i, ok := val.(bool); ok {
		res = i
	} else {
		res = reflect.ValueOf(val).Bool()
	}
	return
}

func lookupString(c Lookup, name interface{}) (res string) {
	val := c.Value(name)
	if val != nil {
		res = val.(string)
	}
	return
}

func lookupList(c Lookup, name interface{}) (res []string) {
	val := c.Value(name)
	if val != nil {
		res = val.([]string)
	}
	return
}

func lookupInt(c Lookup, name interface{}) (res int) {
	val := c.Value(name)
	if i, ok := val.(int); ok {
		res = i
	} else {
		res = int(reflect.ValueOf(val).Int())
	}
	return
}

func lookupInt8(c Lookup, name interface{}) (res int8) {
	val := c.Value(name)
	if i, ok := val.(int8); ok {
		res = i
	} else {
		res = int8(reflect.ValueOf(val).Int())
	}
	return
}

func lookupInt16(c Lookup, name interface{}) (res int16) {
	val := c.Value(name)
	if i, ok := val.(int16); ok {
		res = i
	} else {
		res = int16(reflect.ValueOf(val).Int())
	}
	return
}

func lookupInt32(c Lookup, name interface{}) (res int32) {
	val := c.Value(name)
	if i, ok := val.(int32); ok {
		res = i
	} else {
		res = int32(reflect.ValueOf(val).Int())
	}
	return
}

func lookupInt64(c Lookup, name interface{}) (res int64) {
	val := c.Value(name)
	if i, ok := val.(int64); ok {
		res = i
	} else {
		res = int64(reflect.ValueOf(val).Int())
	}
	return
}

func lookupUint(c Lookup, name interface{}) (res uint) {
	val := c.Value(name)
	if i, ok := val.(uint); ok {
		res = i
	} else {
		res = uint(reflect.ValueOf(val).Uint())
	}
	return
}

func lookupUint8(c Lookup, name interface{}) (res uint8) {
	val := c.Value(name)
	if i, ok := val.(uint8); ok {
		res = i
	} else {
		res = uint8(reflect.ValueOf(val).Uint())
	}
	return
}

func lookupUint16(c Lookup, name interface{}) (res uint16) {
	val := c.Value(name)
	if i, ok := val.(uint16); ok {
		res = i
	} else {
		res = uint16(reflect.ValueOf(val).Uint())
	}
	return
}

func lookupUint32(c Lookup, name interface{}) (res uint32) {
	val := c.Value(name)
	if i, ok := val.(uint32); ok {
		res = i
	} else {
		res = uint32(reflect.ValueOf(val).Uint())
	}
	return
}

func lookupUint64(c Lookup, name interface{}) (res uint64) {
	val := c.Value(name)
	if i, ok := val.(uint64); ok {
		res = i
	} else {
		res = uint64(reflect.ValueOf(val).Uint())
	}
	return
}

func lookupFloat32(c Lookup, name interface{}) (res float32) {
	val := c.Value(name)
	if i, ok := val.(float32); ok {
		res = i
	} else {
		res = float32(reflect.ValueOf(val).Float())
	}
	return
}

func lookupFloat64(c Lookup, name interface{}) (res float64) {
	val := c.Value(name)
	if i, ok := val.(float64); ok {
		res = i
	} else {
		res = float64(reflect.ValueOf(val).Float())
	}
	return
}

func lookupDuration(c Lookup, name interface{}) (res time.Duration) {
	val := c.Value(name)
	if val != nil {
		res = val.(time.Duration)
	}
	return
}

func lookupFile(c Lookup, name interface{}) (res *File) {
	val := c.Value(name)
	if val != nil {
		res = val.(*File)
	}
	return
}

func lookupFileSet(c Lookup, name interface{}) (res *FileSet) {
	val := c.Value(name)
	if val != nil {
		res = val.(*FileSet)
	}
	return
}

func lookupMap(c Lookup, name interface{}) (res map[string]string) {
	val := c.Value(name)
	if val != nil {
		res = val.(map[string]string)
	}
	return
}

func lookupNameValue(c Lookup, name interface{}) (res *NameValue) {
	val := c.Value(name)
	if val != nil {
		res = val.(*NameValue)
	}
	return
}

func lookupNameValues(c Lookup, name interface{}) (res []*NameValue) {
	val := c.Value(name)
	if val != nil {
		res = val.([]*NameValue)
	}
	return
}

func lookupURL(c Lookup, name interface{}) (res *url.URL) {
	val := c.Value(name)
	if val != nil {
		res = val.(*url.URL)
	}
	return
}

func lookupRegexp(c Lookup, name interface{}) (res *regexp.Regexp) {
	val := c.Value(name)
	if val != nil {
		res = val.(*regexp.Regexp)
	}
	return
}

func lookupIP(c Lookup, name interface{}) (res net.IP) {
	val := c.Value(name)
	if val != nil {
		res = val.(net.IP)
	}
	return
}

func lookupBigInt(c Lookup, name interface{}) (res *big.Int) {
	val := c.Value(name)
	if val != nil {
		res = val.(*big.Int)
	}
	return
}

func lookupBigFloat(c Lookup, name interface{}) (res *big.Float) {
	val := c.Value(name)
	if val != nil {
		res = val.(*big.Float)
	}
	return
}

func lookupBytes(c Lookup, name interface{}) (res []byte) {
	val := c.Value(name)
	if val != nil {
		res = val.([]byte)
	}
	return
}

var _ Lookup = (LookupFunc)(nil)
