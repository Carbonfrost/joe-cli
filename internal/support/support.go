// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package support

import (
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

type valueResetOrMerge interface {
	Reset()
}

func FlattenValues(m map[string][]string) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		result[k] = strings.Join(v, ",")
	}
	return result
}

// ParseMap applies parsing of maps.
//
// The map syntax is comma-delimited list of key-value pairs. For key-value
// pairs without an equal sign, the first occurrence of a properly formed KVP
// implies the rest are keys without values. As a special case, the initial keys
// without values are added to "".
//
//	NoKey,A=1,B,C   --> ""=["NoKey"], "B"=[], and "C"=[]
//
// Another special case is when the first (and only) instance is a key
//
//	A=1,2,3         --> "A"=["1","2","3"]  (instead of "A"=["1"], "2"=[], "3"=[])
func ParseMap(text string) map[string][]string {
	if len(text) == 0 {
		return nil
	}

	result := stringListMap{}
	prefix, kvps, suffix := splitMapRegions(text)

	for _, field := range prefix {
		result.Add("", unescapeAndUnquote(field))
	}

	for _, field := range kvps {
		key, value, _ := splitKeyValue(field)
		if key == "" {
			result.Add("", unescapeAndUnquote(field))
		} else {
			result.Add(key, value)
		}
	}

	for _, field := range suffix {
		result.Add(field, "")
	}

	return result
}

type stringListMap map[string][]string

func (m stringListMap) Add(key, value string) {
	m[key] = append(m[key], value)
}

func splitFields(s string, sep rune, n int) []string {
	if len(s) == 0 {
		return nil
	}

	var fields []string
	var buf strings.Builder
	var quote rune
	escaped := false

	// Quoting and escaping is tracked in order to correctly delimit fields;
	// however, actual unquoting and unescaping happens when parsing key-value pairs
	for i, r := range s {
		switch {
		case escaped:
			buf.WriteRune(r)
			escaped = false

		case r == '\\':
			// Preserve escape characters
			buf.WriteRune(r)
			escaped = true

		case quote != 0:
			if r == quote {
				quote = 0
			}

			// Preserve quote characters
			buf.WriteRune(r)

		case r == '\'' || r == '"':
			quote = r

			// Preserve quote characters
			buf.WriteRune(r)

		case r == sep:
			str := buf.String()
			if len(fields) == n {
				str = s[i:]
				break
			}

			fields = append(fields, str)
			buf.Reset()

		case r == '=':
			buf.WriteRune(r)

		default:
			buf.WriteRune(r)
		}
	}

	fields = append(fields, buf.String())
	return fields
}

// splitMapRegions splits the map text into its regions and their list of fields.
//
//	Prefix1,Prefix2,KVP=1,KVP=2,Suffix1,Suffix2
func splitMapRegions(s string) (prefix, kvps, suffix []string) {
	if len(s) == 0 {
		return
	}

	tmpKVPs := splitFields(s, ',', -1)

	prefix = tmpKVPs
	var prefixIndex int
	for prefixIndex = 0; prefixIndex < len(tmpKVPs); prefixIndex++ {
		// TODO Need additional check for escaping
		if strings.Contains(tmpKVPs[prefixIndex], "=") {
			prefix = tmpKVPs[0:prefixIndex]
			tmpKVPs = tmpKVPs[prefixIndex:]
			break
		}
	}

	// Split tmpKVPs into kvps and suffix by looking for equals
	if len(tmpKVPs) == 0 {
		return
	}

	var suffixIndex int
	for suffixIndex = len(tmpKVPs) - 1; suffixIndex >= 0; suffixIndex-- {
		// TODO Need additional check for escaping
		if strings.Contains(tmpKVPs[suffixIndex], "=") {
			suffix = tmpKVPs[suffixIndex+1:]
			break
		}
	}

	// Special case with only one KVP
	if len(prefix) == 0 && suffixIndex == 0 {
		kvps = []string{strings.Join(tmpKVPs, ",")}
		suffix = nil
		return
	}

	kvps = make([]string, 0, len(tmpKVPs))
	for _, field := range tmpKVPs[:suffixIndex+1] {
		if strings.Contains(field, "=") {
			kvps = append(kvps, field)
		} else {
			kvps[len(kvps)-1] += ("," + field)
		}
	}
	return
}

func ParseKeyValue(field string) (string, string, bool) {
	return splitKeyValue(field)
}

// splitKeyValue splits on the first unescaped '=' outside quotes.
func splitKeyValue(field string) (string, string, bool) {
	var buf strings.Builder
	var quote rune
	escaped := false

	for i, r := range field {
		switch {
		case escaped:
			buf.WriteRune(r)
			escaped = false

		case r == '\\':
			escaped = true

		case quote != 0:
			if r == quote {
				quote = 0
			}
			buf.WriteRune(r)

		case r == '\'' || r == '"':
			quote = r
			buf.WriteRune(r)

		case r == '=':
			key := unescapeAndUnquote(buf.String())
			value := unescapeAndUnquote(field[i+1:])
			return key, value, true

		default:
			buf.WriteRune(r)
		}
	}

	return field, "", false
}

func Unescape(s []string) []string {
	for i := range s {
		s[i] = unescapeAndUnquote(s[i])
	}
	return s
}

// unescapeAndUnquote removes surrounding quotes and processes backslash escapes.
func unescapeAndUnquote(s string) string {
	if s1, err := strconv.Unquote(s); err == nil {
		s = s1
	}

	var buf strings.Builder
	escaped := false

	for _, r := range s {
		if escaped {
			buf.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		buf.WriteRune(r)
	}

	return buf.String()
}

func FormatMap(m map[string]string, delim string) string {
	items := make([]string, len(m))
	var i int
	for k, v := range m {
		items[i] = k + "=" + v
		i++
	}
	sort.Strings(items)
	return strings.Join(items, delim)
}

func SplitList(s, sep string, n int) []string {
	if sep == "" || len([]rune(sep)) > 1 {
		panic("sep cannot be empty or more than 1 rune")
	}

	s = strings.TrimSpace(s)
	return splitFields(s, []rune(sep)[0], n)
}

func MustValueReset(p any) {
	res, ok := p.(valueResetOrMerge)
	if !ok {
		panic(errUnsupportedForCopying(p))
	}
	res.Reset()
}

func MustValueCloneZero(p any) any {
	// TODO This does not support []*NameValue
	switch val := p.(type) {
	case *bool:
		return new(bool)
	case *string:
		return new(string)
	case *[]string:
		return new([]string)
	case *int:
		return new(int)
	case *int8:
		return new(int8)
	case *int16:
		return new(int16)
	case *int32:
		return new(int32)
	case *int64:
		return new(int64)
	case *uint:
		return new(uint)
	case *uint8:
		return new(uint8)
	case *uint16:
		return new(uint16)
	case *uint32:
		return new(uint32)
	case *uint64:
		return new(uint64)
	case *float32:
		return new(float32)
	case *float64:
		return new(float64)
	case *time.Duration:
		return new(time.Duration)
	case *map[string]string:
		return new(map[string]string)
	case **url.URL:
		return new(*url.URL)
	case *net.IP:
		return new(net.IP)
	case **regexp.Regexp:
		return new(*regexp.Regexp)
	case **big.Int:
		return new(*big.Int)
	case **big.Float:
		return new(*big.Float)
	case *[]byte:
		return new([]byte)
	case valueResetOrMerge:
		r := reflect.ValueOf(val).MethodByName("Copy")
		if r.IsValid() {
			res := r.Call(nil)[0].Interface()
			res.(valueResetOrMerge).Reset()
			return res
		}
	}

	panic(errUnsupportedForCopying(p))
}

func errUnsupportedForCopying(p any) error {
	return fmt.Errorf("unsupported flag type for copying or resetting: %T", p)
}

func GuessWidth() int {
	cols := os.Getenv("COLUMNS")
	if cols != "" {
		width, err := strconv.Atoi(cols)
		if err == nil && width > 12 && width < 80 {
			return width
		}
	}

	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		width, _, err := term.GetSize(fd)
		if err == nil && width > 12 && width < 80 {
			return width
		}
	}
	return 80
}
