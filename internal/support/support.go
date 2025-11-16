// Copyright 2023 The Joe-cli Authors. All rights reserved.
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

func ParseMap(values []string) map[string]string {
	res := map[string]string{}

	var key, value string
	for _, kvp := range values {
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
		res[key] = value
	}
	return res
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
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.Contains(s, "\\") {
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
