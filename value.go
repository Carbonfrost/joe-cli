package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/getopt/v2"
)

type Value interface {
	Set(string) error
	String() string
}

type generic struct {
	p interface{}
}

func (g *generic) Set(value string, opt getopt.Option) error {
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
		// If this is the first time we are seen then nil out the
		// default value.
		if opt.Count() <= 1 {
			*p = nil
		}
		*p = append(*p, a...)
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
		if *p {
			return "true"
		}
		return "false"
	case *string:
		return *p
	case *[]string:
		return strings.Join([]string(*p), ",")
	case *int:
		return strconv.FormatInt(int64(*p), 10)
	case *int8:
		return strconv.FormatInt(int64(*p), 10)
	case *int16:
		return strconv.FormatInt(int64(*p), 10)
	case *int32:
		return strconv.FormatInt(int64(*p), 10)
	case *int64:
		return strconv.FormatInt(*p, 10)
	case *uint:
		return strconv.FormatUint(uint64(*p), 10)
	case *uint8:
		return strconv.FormatUint(uint64(*p), 10)
	case *uint16:
		return strconv.FormatUint(uint64(*p), 10)
	case *uint32:
		return strconv.FormatUint(uint64(*p), 10)
	case *uint64:
		return strconv.FormatUint(*p, 10)
	case *float32:
		return strconv.FormatFloat(float64(*p), 'g', -1, 32)
	case *float64:
		return strconv.FormatFloat(*p, 'g', -1, 64)
	case *time.Duration:
		return p.String()
	}
	panic("unreachable!")
}

func wrapGeneric(v interface{}) *generic {
	switch v.(type) {
	case Value:
		panic("not implemented: Value wrap generic")
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
	default:
		panic(fmt.Sprintf("unsupported flag type: %T", v))
	}
}
