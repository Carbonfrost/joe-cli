package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
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

func URL() **url.URL {
	return new(*url.URL)
}

func Regexp() **regexp.Regexp {
	return new(*regexp.Regexp)
}

func IP() *net.IP {
	return new(net.IP)
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
	trySetOptional := func() (interface{}, bool) {
		return opt.optionalValue, (value == "" && opt.optional)
	}
	switch p := g.p.(type) {
	case Value:
		return p.Set(value)
	case *bool:
		if v, ok := trySetOptional(); ok {
			*p = v.(bool)
			return nil
		}
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
		if v, ok := trySetOptional(); ok {
			*p = v.(string)
			return nil
		}
		*p = value
		return nil
	case *[]string:
		if v, ok := trySetOptional(); ok {
			*p = v.([]string)
			return nil
		}
		a := splitWithEscapes(value, ",", -1)
		// Reset on the first occurrence
		if opt.Count() <= 1 {
			*p = nil
		}
		*p = append(*p, a...)
		return nil
	case *map[string]string:
		if v, ok := trySetOptional(); ok {
			*p = v.(map[string]string)
			return nil
		}
		if opt.Count() <= 1 {
			// Reset the map on the first occurrence
			*p = map[string]string{}
		}
		text := value
		var key, value string
		for _, kvp := range splitWithEscapes(text, ",", -1) {
			k := splitWithEscapes(kvp, "=", 2)
			switch len(k) {
			case 2:
				key = k[0]
				value = k[1]
			case 1:
				// Implies comma was meant to be escaped
				// -m key=value,s,t  --> interpreted as key=value,s,t rather than s and t keys
				value = value + "," + k[0]
			}
			m := *p
			m[key] = value
		}

		return nil
	case *int:
		if v, ok := trySetOptional(); ok {
			*p = v.(int)
			return nil
		}
		i64, err := strconv.ParseInt(value, 0, strconv.IntSize)
		if err == nil {
			*p = int(i64)
		}
		return strconvErr(err)
	case *int8:
		if v, ok := trySetOptional(); ok {
			*p = v.(int8)
			return nil
		}
		i64, err := strconv.ParseInt(value, 0, 8)
		if err == nil {
			*p = int8(i64)
		}
		return strconvErr(err)
	case *int16:
		if v, ok := trySetOptional(); ok {
			*p = v.(int16)
			return nil
		}
		i64, err := strconv.ParseInt(value, 0, 16)
		if err == nil {
			*p = int16(i64)
		}
		return strconvErr(err)
	case *int32:
		if v, ok := trySetOptional(); ok {
			*p = v.(int32)
			return nil
		}
		i64, err := strconv.ParseInt(value, 0, 32)
		if err == nil {
			*p = int32(i64)
		}
		return strconvErr(err)
	case *int64:
		if v, ok := trySetOptional(); ok {
			*p = v.(int64)
			return nil
		}
		i64, err := strconv.ParseInt(value, 0, 64)
		if err == nil {
			*p = i64
		}
		return strconvErr(err)
	case *uint:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint)
			return nil
		}
		u64, err := strconv.ParseUint(value, 0, strconv.IntSize)
		if err == nil {
			*p = uint(u64)
		}
		return strconvErr(err)
	case *uint8:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint8)
			return nil
		}
		u64, err := strconv.ParseUint(value, 0, 8)
		if err == nil {
			*p = uint8(u64)
		}
		return strconvErr(err)
	case *uint16:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint16)
			return nil
		}
		u64, err := strconv.ParseUint(value, 0, 16)
		if err == nil {
			*p = uint16(u64)
		}
		return strconvErr(err)
	case *uint32:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint32)
			return nil
		}
		u64, err := strconv.ParseUint(value, 0, 32)
		if err == nil {
			*p = uint32(u64)
		}
		return strconvErr(err)
	case *uint64:
		if v, ok := trySetOptional(); ok {
			*p = v.(uint64)
			return nil
		}
		u64, err := strconv.ParseUint(value, 0, 64)
		if err == nil {
			*p = u64
		}
		return strconvErr(err)
	case *float32:
		if v, ok := trySetOptional(); ok {
			*p = v.(float32)
			return nil
		}
		f64, err := strconv.ParseFloat(value, 32)
		if err == nil {
			*p = float32(f64)
		}
		return strconvErr(err)
	case *float64:
		if v, ok := trySetOptional(); ok {
			*p = v.(float64)
			return nil
		}
		f64, err := strconv.ParseFloat(value, 64)
		if err == nil {
			*p = f64
		}
		return strconvErr(err)
	case *time.Duration:
		if v, ok := trySetOptional(); ok {
			*p = v.(time.Duration)
			return nil
		}
		v, err := time.ParseDuration(value)
		if err == nil {
			*p = v
		}
		return err
	case **url.URL:
		if v, ok := trySetOptional(); ok {
			*p = v.(*url.URL)
			return nil
		}
		v, err := url.Parse(value)
		if err == nil {
			*p = v
		}
		return err

	case *net.IP:
		if v, ok := trySetOptional(); ok {
			*p = v.(net.IP)
			return nil
		}
		v := net.ParseIP(value)
		if v != nil {
			*p = v
			return nil
		}
		return errors.New("not a valid IP address")

	case **regexp.Regexp:
		if v, ok := trySetOptional(); ok {
			*p = v.(*regexp.Regexp)
			return nil
		}
		v, err := regexp.Compile(value)
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
	case **url.URL:
		return genericString(*p)
	case *net.IP:
		return genericString(*p)
	case **regexp.Regexp:
		return genericString(*p)
	case Value:
		return genericString(p)
	}
	panic("unreachable!")
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
	case *url.URL:
		return fmt.Sprint(p)
	case net.IP:
		return fmt.Sprint(p)
	case *regexp.Regexp:
		return fmt.Sprint(p)
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
	case **url.URL:
		return &generic{v}
	case *net.IP:
		return &generic{v}
	case **regexp.Regexp:
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

// splitWithEscapes considers escape sequences when splitting.  sep must not
// be empty string
func splitWithEscapes(s, sep string, n int) []string {
	if strings.Index(s, "\\") >= 0 {
		regex := regexp.MustCompile(`(^|[^\\])` + regexp.QuoteMeta(sep))
		matches := regex.FindAllStringSubmatchIndex(s, n)

		if len(matches) == 0 {
			return []string{s}
		}

		unquote := func(x string) string {
			return strings.ReplaceAll(x, "\\", "")
		}
		res := make([]string, 0)

		var last int
		for _, match := range matches {
			res = append(res, unquote(s[last:match[1]-1]))
			res = append(res, unquote(s[match[2]+1+1:]))
			last = match[2] + 1 + 1
		}
		return res
	}
	return strings.SplitN(s, sep, n)
}
