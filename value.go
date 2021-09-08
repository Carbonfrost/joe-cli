package cli

import (
	"bytes"
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Value = flag.Value

type generic struct {
	p interface{}
}

func Bool() *bool {
	return new(bool)
}

func String() *string {
	return new(string)
}

func List() *[]string {
	return new([]string)
}

func Int() *int {
	return new(int)
}

func Int8() *int8 {
	return new(int8)
}

func Int16() *int16 {
	return new(int16)
}

func Int32() *int32 {
	return new(int32)
}

func Int64() *int64 {
	return new(int64)
}

func UInt() *uint {
	return new(uint)
}

func UInt8() *uint8 {
	return new(uint8)
}

func UInt16() *uint16 {
	return new(uint16)
}

func UInt32() *uint32 {
	return new(uint32)
}

func UInt64() *uint64 {
	return new(uint64)
}

func Float32() *float32 {
	return new(float32)
}

func Float64() *float64 {
	return new(float64)
}

func Duration() *time.Duration {
	return new(time.Duration)
}

func Map() *map[string]string {
	return new(map[string]string)
}

func (g *generic) Set(value string, opt *internalOption) error {
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
	switch p := g.p.(type) {
	case Value:
		return p.Set(value)
	case *bool:
		switch strings.ToLower(value) {
		case "", "1", "true", "on", "t":
			*p = true
		case "0", "false", "off", "f":
			*p = false
		default:
			return fmt.Errorf("invalid value for bool %q", value)
		}
		return nil
	case *string:
		*p = value
		return nil
	case *[]string:
		a := strings.Split(value, ",")
		// Reset on the first occurrence
		if opt.Count() <= 1 {
			*p = nil
		}
		*p = append(*p, a...)
		return nil
	case *map[string]string:
		if opt.Count() <= 1 {
			// Reset the map on the first occurrence
			*p = map[string]string{}
		}
		for _, kvp := range strings.Split(value, ",") {
			k := strings.SplitN(kvp, "=", 2)
			var key, value string
			switch len(k) {
			case 2:
				value = k[1]
				fallthrough
			case 1:
				key = k[0]
			}
			m := *p
			m[key] = value
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
	}
	panic("unreachable!")
}

func (g *generic) String() string {
	switch p := g.p.(type) {
	case *bool:
		return genericString(*p)
	case *string:
		return genericString(*p)
	case *[]string:
		return genericString(*p)
	case *int:
		return genericString(*p)
	case *int8:
		return genericString(*p)
	case *int16:
		return genericString(*p)
	case *int32:
		return genericString(*p)
	case *int64:
		return genericString(*p)
	case *uint:
		return genericString(*p)
	case *uint8:
		return genericString(*p)
	case *uint16:
		return genericString(*p)
	case *uint32:
		return genericString(*p)
	case *uint64:
		return genericString(*p)
	case *float32:
		return genericString(*p)
	case *float64:
		return genericString(*p)
	case *time.Duration:
		return genericString(*p)
	case *map[string]string:
		return genericString(*p)
	case Value:
		return genericString(p)
	}
	panic("unreachable!")
}

func genericString(v interface{}) string {
	switch p := v.(type) {
	case Value:
		return p.String()
	case bool:
		if p {
			return "true"
		}
		return "false"
	case string:
		return p
	case []string:
		return strings.Join([]string(p), ",")
	case int:
		return strconv.FormatInt(int64(p), 10)
	case int8:
		return strconv.FormatInt(int64(p), 10)
	case int16:
		return strconv.FormatInt(int64(p), 10)
	case int32:
		return strconv.FormatInt(int64(p), 10)
	case int64:
		return strconv.FormatInt(p, 10)
	case uint:
		return strconv.FormatUint(uint64(p), 10)
	case uint8:
		return strconv.FormatUint(uint64(p), 10)
	case uint16:
		return strconv.FormatUint(uint64(p), 10)
	case uint32:
		return strconv.FormatUint(uint64(p), 10)
	case uint64:
		return strconv.FormatUint(p, 10)
	case float32:
		return strconv.FormatFloat(float64(p), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(p, 'g', -1, 64)
	case time.Duration:
		return p.String()
	case map[string]string:
		return formatMap(p)
	}
	panic("unreachable!")
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
	default:
		panic(fmt.Sprintf("unsupported flag type: %T", v))
	}
}

// wrapFlagLong will wrap the simple flag.Value with the one
// required by getopt if necessary
func wrapFlagLong(v interface{}) interface{} {
	switch v.(type) {
	case Value:
		return wrapGeneric(v)
	default:
		return v
	}
}

func dereference(v interface{}) interface{} {
	if _, ok := v.(Value); ok {
		return v
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		return val.Elem().Interface()
	}
	return v
}

func formatMap(m map[string]string) string {
	var (
		b     bytes.Buffer
		comma bool
	)
	for k, v := range m {
		if comma {
			b.WriteString(",")
		}
		comma = true
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(v)
	}
	return b.String()
}
