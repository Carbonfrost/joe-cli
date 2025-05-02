// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package synopsis

import (
	"bytes"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	usagePattern = regexp.MustCompile(`{(.+?)}`)
)

type placeholdersByPos []*placeholderExpr

type valueProvidesSynopsis interface {
	Synopsis() string
}

type Usage struct {
	exprs []expr
}

type expr interface {
	exprSigil()
}

type placeholderExpr struct {
	name string
	pos  int
}

type literal struct {
	text string
}

func Placeholder(v any) string {
	switch m := v.(type) {
	case *bool:
		return ""
	case interface{ IsBoolFlag() bool }:
		if m.IsBoolFlag() {
			return ""
		}

	case *int, *int8, *int16, *int32, *int64:
		return "NUMBER"
	case *uint, *uint8, *uint16, *uint32, *uint64:
		return "NUMBER"
	case *float32, *float64:
		return "NUMBER"
	case *string:
		return "STRING"
	case *[]string:
		return "VALUES"
	case *time.Duration:
		return "DURATION"
	case *map[string]string:
		return "NAME=VALUE"
	case **url.URL:
		return "URL"
	case *net.IP:
		return "IP"
	case **regexp.Regexp:
		return "PATTERN"
	case valueProvidesSynopsis:
		return m.Synopsis()
	default:
	}
	return "VALUE"
}

func ParseUsage(text string) *Usage {
	content := []byte(text)
	allIndexes := usagePattern.FindAllSubmatchIndex(content, -1)
	result := []expr{}

	var index int
	for _, loc := range allIndexes {
		if index < loc[0] {
			result = append(result, newLiteral(content[index:loc[0]]))
		}
		key := content[loc[2]:loc[3]]
		result = append(result, newExpr(key))
		index = loc[1]
	}
	if index < len(content) {
		result = append(result, newLiteral(content[index:]))
	}

	return &Usage{
		result,
	}
}

func newLiteral(token []byte) expr {
	return &literal{string(token)}
}

func newExpr(token []byte) expr {
	positionAndName := strings.SplitN(string(token), ":", 2)
	if len(positionAndName) == 1 {
		return &placeholderExpr{name: positionAndName[0], pos: -1}
	}

	pos, _ := strconv.Atoi(positionAndName[0])
	name := positionAndName[1]
	return &placeholderExpr{name: name, pos: pos}
}

func (u *Usage) Placeholders() []string {
	res := make([]string, 0)
	for _, e := range u.placeholders() {
		res = append(res, e.name)
	}
	return res
}

func (u *Usage) placeholders() []*placeholderExpr {
	res := make(placeholdersByPos, 0, len(u.exprs))
	seen := map[string]bool{}
	for _, item := range u.exprs {
		if e, ok := item.(*placeholderExpr); ok {
			if !seen[e.name] {
				res = append(res, e)
				seen[e.name] = true
			}
		}
	}
	sort.Sort(res)
	return res
}

func (u *Usage) WithoutPlaceholders() string {
	var b bytes.Buffer
	for _, e := range u.exprs {
		switch item := e.(type) {
		case *placeholderExpr:
			b.WriteString(item.name)
		case *literal:
			b.WriteString(item.text)
		}
	}
	return b.String()
}

func (u *Usage) HelpText(w styleWriter) {
	for _, e := range u.exprs {
		switch item := e.(type) {
		case *placeholderExpr:
			w.Styled(Underline, item.name)
		case *literal:
			w.WriteString(item.text)
		}
	}
}

func (p placeholdersByPos) Less(i, j int) bool {
	return p[i].pos < p[j].pos
}

func (p placeholdersByPos) Len() int {
	return len(p)
}

func (p placeholdersByPos) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (*placeholderExpr) exprSigil() {}
func (*literal) exprSigil()         {}
