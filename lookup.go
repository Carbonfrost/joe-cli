package cli

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
)

// Lookup provides type conversion from the various built-in types supported by
// the framework. For each method, the name is either a string, rune, *Arg, or *Flag
// corresponding to the name of the argument or flag.
type Lookup interface {
	// Bool obtains the value and converts it to a bool
	Bool(name interface{}) bool
	// File obtains the value and converts it to a File
	File(name interface{}) *File
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
	// List obtains the value and converts it to a slice of strings
	List(name interface{}) []string
	// Map obtains the value and converts it to a map
	Map(name interface{}) map[string]string
	// String obtains the value and converts it to a string
	String(name interface{}) string
	// UInt obtains the value and converts it to a uInt
	UInt(name interface{}) uint
	// UInt16 obtains the value and converts it to a uInt16
	UInt16(name interface{}) uint16
	// UInt32 obtains the value and converts it to a uInt32
	UInt32(name interface{}) uint32
	// UInt64 obtains the value and converts it to a uInt64
	UInt64(name interface{}) uint64
	// UInt8 obtains the value and converts it to a uInt8
	UInt8(name interface{}) uint8
	// Value obtains the value and converts it to Value
	Value(name interface{}) interface{}
	// URL obtains the value and converts it to a URL
	URL(name interface{}) *url.URL
	// Regexp obtains the value and converts it to a Regexp
	Regexp(name interface{}) *regexp.Regexp
	// IP obtains the value and converts it to a IP
	IP(name interface{}) *net.IP
}

// LookupValues provides a Lookup backed by a map
type LookupValues map[string]interface{}

// Value obtains the value and converts it to Value
func (c LookupValues) Value(name interface{}) interface{} {
	if c == nil {
		return nil
	}
	switch v := name.(type) {
	case rune:
		return c[string(v)]
	case string:
		return c[v]
	case *Arg:
		return c[v.Name]
	case *Flag:
		return c[v.Name]
	}
	panic(fmt.Sprintf("unexpected type: %T", name))
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

// UInt obtains the uint for the specified name
func (c LookupValues) UInt(name interface{}) uint {
	return lookupUInt(c, name)
}

// UInt8 obtains the uint8 for the specified name
func (c LookupValues) UInt8(name interface{}) uint8 {
	return lookupUInt8(c, name)
}

// UInt16 obtains the uint16 for the specified name
func (c LookupValues) UInt16(name interface{}) uint16 {
	return lookupUInt16(c, name)
}

// UInt32 obtains the uint32 for the specified name
func (c LookupValues) UInt32(name interface{}) uint32 {
	return lookupUInt32(c, name)
}

// UInt64 obtains the uint64 for the specified name
func (c LookupValues) UInt64(name interface{}) uint64 {
	return lookupUInt64(c, name)
}

// Float32 obtains the float32 for the specified name
func (c LookupValues) Float32(name interface{}) float32 {
	return lookupFloat32(c, name)
}

// Float64 obtains the float64 for the specified name
func (c LookupValues) Float64(name interface{}) float64 {
	return lookupFloat64(c, name)
}

// File obtains the File for the specified name
func (c LookupValues) File(name interface{}) *File {
	return lookupFile(c, name)
}

// Map obtains the map for the specified name
func (c LookupValues) Map(name interface{}) map[string]string {
	return lookupMap(c, name)
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
func (c LookupValues) IP(name interface{}) *net.IP {
	return lookupIP(c, name)
}

func lookupBool(c Lookup, name interface{}) (res bool) {
	val := c.Value(name)
	if val != nil {
		res = val.(bool)
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
	if val != nil {
		res = val.(int)
	}
	return
}

func lookupInt8(c Lookup, name interface{}) (res int8) {
	val := c.Value(name)
	if val != nil {
		res = val.(int8)
	}
	return
}

func lookupInt16(c Lookup, name interface{}) (res int16) {
	val := c.Value(name)
	if val != nil {
		res = val.(int16)
	}
	return
}

func lookupInt32(c Lookup, name interface{}) (res int32) {
	val := c.Value(name)
	if val != nil {
		res = val.(int32)
	}
	return
}

func lookupInt64(c Lookup, name interface{}) (res int64) {
	val := c.Value(name)
	if val != nil {
		res = val.(int64)
	}
	return
}

func lookupUInt(c Lookup, name interface{}) (res uint) {
	val := c.Value(name)
	if val != nil {
		res = val.(uint)
	}
	return
}

func lookupUInt8(c Lookup, name interface{}) (res uint8) {
	val := c.Value(name)
	if val != nil {
		res = val.(uint8)
	}
	return
}

func lookupUInt16(c Lookup, name interface{}) (res uint16) {
	val := c.Value(name)
	if val != nil {
		res = val.(uint16)
	}
	return
}

func lookupUInt32(c Lookup, name interface{}) (res uint32) {
	val := c.Value(name)
	if val != nil {
		res = val.(uint32)
	}
	return
}

func lookupUInt64(c Lookup, name interface{}) (res uint64) {
	val := c.Value(name)
	if val != nil {
		res = val.(uint64)
	}
	return
}

func lookupFloat32(c Lookup, name interface{}) (res float32) {
	val := c.Value(name)
	if val != nil {
		res = val.(float32)
	}
	return
}

func lookupFloat64(c Lookup, name interface{}) (res float64) {
	val := c.Value(name)
	if val != nil {
		res = val.(float64)
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

func lookupMap(c Lookup, name interface{}) (res map[string]string) {
	val := c.Value(name)
	if val != nil {
		res = val.(map[string]string)
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

func lookupIP(c Lookup, name interface{}) (res *net.IP) {
	val := c.Value(name)
	if val != nil {
		res = val.(*net.IP)
	}
	return
}
