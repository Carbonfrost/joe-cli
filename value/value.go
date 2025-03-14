package value

import (
	"encoding"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
)

// Hex represents an integer that parses from the hex syntax
type Hex int

// Octal represents an integer that parses from the octal syntax
type Octal int

// ByteLength represents number of bytes
type ByteLength int

type jsonValue struct {
	V any
}

// JSON wraps a pointer to a value which will be marshalled from files as JSON.
// The value can't be used directly from the command line unless it also implements
// Value.Set or ValueReader.SetData, which the value must define.  Using JSON from
// the command line would be cumbersome.
func JSON(v any) flag.Getter {
	return &jsonValue{
		V: v,
	}
}

var (
	byteLengthPat = regexp.MustCompile(`([0-9.]+)\s*([kKMGTPEZYRQ]i?)?B`)
	magnitude     = map[string]int{
		"":  0,
		"k": 1,
		"K": 1,
		"M": 2,
		"G": 3,
		"T": 4,
		"P": 5,
		"E": 6,
		"Z": 7,
		"Y": 8,
		"R": 9,
		"Q": 10,
	}
)

// ParseByteLength from a string
func ParseByteLength(s string) (int, error) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "B") {
		return strconv.Atoi(s)
	}

	sub := byteLengthPat.FindSubmatch([]byte(s))
	if len(sub) == 0 {
		return -1, fmt.Errorf("invalid byte length")
	}
	if len(sub[2]) == 0 {
		return strconv.Atoi(string(sub[1]))
	}

	num, _ := strconv.ParseFloat(string(sub[1]), 64)
	var base float64 = 1000
	if strings.HasSuffix(string(sub[2]), "i") {
		base = 1024
	}
	magnitude := magnitude[string(sub[2][0])]
	return int(num * math.Pow(base, float64(magnitude))), nil
}

func (b *ByteLength) UnmarshalText(data []byte) error {
	val, err := ParseByteLength(string(data))
	if err != nil {
		return err
	}
	*b = ByteLength(val)
	return nil
}

func (h *Octal) UnmarshalText(d []byte) error {
	s, err := strconv.ParseInt(strings.TrimPrefix(string(d), "0o"), 8, 64)
	*h = Octal(s)
	return formatStrconvError(err, string(d))
}

func (h Octal) String() string {
	return fmt.Sprintf("0o%o", int(h))
}

func (h *Hex) UnmarshalText(d []byte) error {
	s, err := strconv.ParseInt(strings.TrimPrefix(string(d), "0x"), 16, 64)
	*h = Hex(s)
	return formatStrconvError(err, string(d))
}

func (h Hex) String() string {
	return fmt.Sprintf("0x%X", int(h))
}

func (j *jsonValue) Set(s string) error {
	if j.supportsIntrinsicSet() {
		return cli.Set(j.V, s)
	}
	return fmt.Errorf("can't set value directly; must read from file")
}

func (j *jsonValue) SetData(r io.Reader) error {
	return json.NewDecoder(r).Decode(j.V)
}

func (j *jsonValue) Get() any {
	return j.V
}

func (j *jsonValue) String() string {
	if j.supportsIntrinsicSet() {
		return cli.Quote(j.V)
	}
	return ""
}

func (j *jsonValue) supportsIntrinsicSet() bool {
	switch j.V.(type) {
	// Supported flag types
	case flag.Value,
		*bool, *string, *[]string, *[]byte, *map[string]string, *[]*cli.NameValue,
		*int, *int8, *int16, *int32, *int64, *uint, *uint8, *uint16, *uint32, *uint64,
		*float32, *float64,
		*time.Duration, **url.URL, *net.IP, **regexp.Regexp, **big.Int, **big.Float,
		encoding.TextUnmarshaler:
		return true
	}
	return false
}

func formatStrconvError(err error, value string) error {
	if e, ok := err.(*strconv.NumError); ok {
		switch e.Err {
		case strconv.ErrRange:
			err = fmt.Errorf("value out of range: %s", value)
		case strconv.ErrSyntax:
			if value == "" {
				err = fmt.Errorf("empty string is not a valid number")
			} else {
				err = fmt.Errorf("not a valid number: %s", value)
			}
		}
	}
	return err
}

var (
	_ encoding.TextUnmarshaler = (*Octal)(nil)
	_ encoding.TextUnmarshaler = (*Hex)(nil)
	_ cli.ValueReader          = (*jsonValue)(nil)
)
