package cli

import (
	"fmt"
)

type Lookup interface {
	Bool(name interface{}) bool
	File(name interface{}) *File
	Float32(name interface{}) float32
	Float64(name interface{}) float64
	Int(name interface{}) int
	Int16(name interface{}) int16
	Int32(name interface{}) int32
	Int64(name interface{}) int64
	Int8(name interface{}) int8
	List(name interface{}) []string
	Map(name interface{}) map[string]string
	String(name interface{}) string
	UInt(name interface{}) uint
	UInt16(name interface{}) uint16
	UInt32(name interface{}) uint32
	UInt64(name interface{}) uint64
	UInt8(name interface{}) uint8
	Value(name interface{}) interface{}
}

type LookupValues map[string]interface{}

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

func (c LookupValues) Bool(name interface{}) bool {
	return lookupBool(c, name)
}

func (c LookupValues) String(name interface{}) string {
	return lookupString(c, name)
}

func (c LookupValues) List(name interface{}) []string {
	return lookupList(c, name)
}

func (c LookupValues) Int(name interface{}) int {
	return lookupInt(c, name)
}

func (c LookupValues) Int8(name interface{}) int8 {
	return lookupInt8(c, name)
}

func (c LookupValues) Int16(name interface{}) int16 {
	return lookupInt16(c, name)
}

func (c LookupValues) Int32(name interface{}) int32 {
	return lookupInt32(c, name)
}

func (c LookupValues) Int64(name interface{}) int64 {
	return lookupInt64(c, name)
}

func (c LookupValues) UInt(name interface{}) uint {
	return lookupUInt(c, name)
}

func (c LookupValues) UInt8(name interface{}) uint8 {
	return lookupUInt8(c, name)
}

func (c LookupValues) UInt16(name interface{}) uint16 {
	return lookupUInt16(c, name)
}

func (c LookupValues) UInt32(name interface{}) uint32 {
	return lookupUInt32(c, name)
}

func (c LookupValues) UInt64(name interface{}) uint64 {
	return lookupUInt64(c, name)
}

func (c LookupValues) Float32(name interface{}) float32 {
	return lookupFloat32(c, name)
}

func (c LookupValues) Float64(name interface{}) float64 {
	return lookupFloat64(c, name)
}

// File obtains the file for the specified flag or argument.
func (c LookupValues) File(name interface{}) *File {
	return lookupFile(c, name)
}

func (c LookupValues) Map(name interface{}) map[string]string {
	return lookupMap(c, name)
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
