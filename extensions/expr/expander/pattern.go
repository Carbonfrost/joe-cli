// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package expander

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"
	"time"
)

var (
	space   = nopExpr{"space"}
	tab     = nopExpr{"tab"}
	newline = nopExpr{"newline"}
	empty   = nopExpr{"empty"}
)

type Syntax int

const (
	// SyntaxDefault indicates that the expression syntax uses the
	// form <name>[':' <format>]. That is, an optional format string is allowed
	// to control the output.
	SyntaxDefault Syntax = iota

	// SyntaxRecursive causes recursive expression evaluation to
	// be allowed. For example, the expression ${VISUAL:${EDITOR}}
	// would evaluate both environment variables, falling back to EDITOR.
	SyntaxRecursive
)

type Pattern struct {
	exprs []expr

	start []byte
	end   []byte
}

type expr interface {
	Format(expand Interface) string
}

type formatExpr struct {
	name        string
	format      string
	trailingOpt string // optional whitespace iff the expr evaluates non-empty
}

// nopExpr is reserved for whitespace expressions that are reserved names
type nopExpr struct {
	name string
}

type fallbackExpr struct {
	name        string
	fallback    expr
	trailingOpt string
}

type literalExpr struct {
	text     string
	trailing string // whitespace after literal (produced by ws expressions)
}

// Renderer is a specialized writer that understands writing to multiple
// files and the corresponding variable support
type Renderer struct {
	io.Writer
	out, err io.Writer
}

func (r *Renderer) expandFiles(k string) any {
	switch k {
	case "stderr":
		r.Writer = r.err
		return ""

	case "stdout":
		r.Writer = r.out
		return ""
	}
	return nil
}

func NewRenderer(stdout, stderr io.Writer) *Renderer {
	return &Renderer{
		Writer: stdout,
		out:    stdout,
		err:    stderr,
	}
}

func Fprint(w io.Writer, pattern *Pattern, e Interface) (count int, err error) {
	// Implicitly upgrade w to *Renderer
	if r, ok := w.(*Renderer); ok {
		e = Compose(Func(r.expandFiles), e)
	}
	for _, item := range pattern.exprs {
		count, err = fmt.Fprint(w, item.Format(e))
		if err != nil {
			break
		}
	}
	return
}

func Compile(pattern string) *Pattern {
	return SyntaxDefault.Compile(pattern)
}

func CompilePattern(pattern, start, end string) *Pattern {
	return SyntaxDefault.CompilePattern(pattern, start, end)
}

func (s Syntax) Compile(pattern string) *Pattern {
	return s.CompilePattern(pattern, "%(", ")")
}

func (s Syntax) CompilePattern(pattern, start, end string) *Pattern {
	endBytes := []byte(end)
	if len(endBytes) > 1 {
		panic("end sequence must be one byte")
	}

	newExpr := defaultNewExpr
	if s == SyntaxRecursive {
		newExpr = recursiveNewExpr(start, end)
	}

	return compilePatternCore([]byte(pattern), []byte(start), endBytes, newExpr)
}

func (l literalExpr) Format(_ Interface) string {
	return l.text + l.trailing
}

func (nopExpr) Format(Interface) string {
	return ""
}

func (n nopExpr) Space() string {
	switch n.name {
	case "space":
		return " "
	case "newline":
		return "\n"
	case "tab":
		return "\t"
	case "empty":
	}
	return ""
}

func fprintReprWS(ws string, w io.Writer, start, end []byte) {
	for _, s := range []byte(ws) {
		w.Write(start)
		fmt.Fprint(w, nameOfWSToken(s).name)
		w.Write(end)
	}
}

func nameOfWSToken(s byte) nopExpr {
	switch s {
	case byte(' '):
		return space

	case byte('\n'):
		return newline

	case byte('\t'):
		return tab
	}
	return empty
}

func (f formatExpr) Format(expand Interface) string {
	var res string
	value := expand.Expand(f.name)
	format := f.format
	switch t := value.(type) {
	case time.Time:
		if format == "" {
			format = time.RFC3339
		}
		res = t.Format(format)
	case error:
		res = fmt.Sprintf("%%!(%s)", t)
	default:
		if format == "" {
			format = "v"
		}
		res = fmt.Sprintf("%"+format, value)
	}

	if res == "" {
		return res
	}
	return res + f.trailingOpt
}

func (f fallbackExpr) Format(expand Interface) string {
	value := expand.Expand(f.name)
	if value == nil {
		return f.fallback.Format(expand)
	}
	res := fmt.Sprint(value)
	if res == "" {
		return res
	}
	return res + f.trailingOpt
}

func (p *Pattern) Expand(expand Interface) string {
	var b strings.Builder
	Fprint(&b, p, expand)
	return b.String()
}

func (p *Pattern) String() string {
	var sb strings.Builder
	for _, e := range p.exprs {
		fprintRepr(e, &sb, p.start, p.end)
	}
	return sb.String()
}

func (p *Pattern) WithMeta(name string, pattern *Pattern) *Pattern {
	newExprs := make([]expr, 0, len(p.exprs))
	for _, e := range p.exprs {
		if f, ok := e.(*formatExpr); ok && f.name == name {
			newExprs = append(newExprs, pattern.exprs...)
			continue
		}
		newExprs = append(newExprs, e)
	}
	return &Pattern{exprs: newExprs, start: p.start, end: p.end}
}

func (p *Pattern) debugExprs() string {
	var sb strings.Builder
	for _, e := range p.exprs {
		fprintDebugExpr(&sb, e)
	}
	return sb.String()
}

func fprintRepr(exp expr, w io.Writer, start, end []byte) {
	switch e := exp.(type) {
	case *fallbackExpr:
		w.Write(start)
		fmt.Fprintf(w, "%s:", e.name)
		fprintRepr(e.fallback, w, start, end)
		w.Write(end)
		fprintReprWS(e.trailingOpt, w, start, end)

	case *literalExpr:
		fmt.Fprint(w, e.text)
		fprintReprWS(e.trailing, w, start, end)

	case *formatExpr:
		w.Write(start)
		if e.format == "" {
			fmt.Fprintf(w, "%s", e.name)
		} else {
			fmt.Fprintf(w, "%s:%s", e.name, e.format)
		}
		w.Write(end)
		fprintReprWS(e.trailingOpt, w, start, end)
	}
}

func fprintDebugExpr(w io.Writer, e expr) {
	switch exp := e.(type) {
	case *fallbackExpr:
		fmt.Fprintf(w, "<fallback %s, to=", exp.name)
		fprintDebugExpr(w, exp.fallback)
		fmt.Fprint(w, ">")
	case *literalExpr:
		fmt.Fprintf(w, "<literal %s> ", exp.text)
	case *formatExpr:
		fmt.Fprintf(w, "<format %s, format=%s> ", exp.name, exp.format)
	}
}

func compilePatternCore(content, start, end []byte, newExpr func([]byte) expr) *Pattern {
	allIndexes := findAllSubmatchIndex(content, start, end[0])
	result := []expr{}

	var index int
	for loc := range allIndexes {
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

	return &Pattern{
		exprs: convertWSExprs(result),
		start: start,
		end:   end,
	}
}

// findAllSubmatchIndex provides the behavior of regexp FindAllSubmatchIndex
// except with simplifying assumptions but also detecting nested patterns. Only
// considers ASCII sequences, only allows a single byte end character.
func findAllSubmatchIndex(content, start []byte, end byte) iter.Seq[[4]int] {
	return func(yield func([4]int) bool) {
		var (
			nested     int
			lenStart   = len(start)
			lenContent = len(content)
			submatch   = func(i, j int) [4]int {
				return [4]int{i, j + 1, i + lenStart, j}
			}
		)

	OUTER:
		for i := 0; i < lenContent; i++ {
			c := content[i]
			// Submatch indexes - same as what regexp.Regexp.FindAllSubmatchIndex returns
			// 0 2    1,3
			// %(hello)
			if c != start[0] {
				continue
			}

			if bytes.Equal(content[i:i+lenStart], start) {
				for j := i + lenStart; j < lenContent; j++ {
					if content[j] == end {
						if nested == 0 {
							sub := submatch(i, j)
							if sub[2] != sub[3] {
								if !yield(sub) {
									return
								}
							}
							i = j
							continue OUTER

						} else {
							nested--
						}
					}

					// Detect nested occurrences
					if j < lenContent-lenStart && bytes.Equal(content[j:j+lenStart], start) {
						nested++
					}
				}
			}
		}
	}
}

// convertWSExprs sets up trailing whitespace in format expressions
// by looking for successive whitespace expansions:
//
//	%(space)
//	%(tab)
//	%(newline)
func convertWSExprs(exprs []expr) []expr {
	var res []expr
	for i := range exprs {
		if wse, ok := exprs[i].(nopExpr); ok {
			if len(res) == 0 {
				res = append(res, &literalExpr{text: "", trailing: wse.Space()})
				continue
			}
			last := res[len(res)-1]

			switch prev := last.(type) {
			case *formatExpr:
				prev.trailingOpt += wse.Space()
			case *fallbackExpr:
				prev.trailingOpt += wse.Space()
			case *literalExpr:
				prev.trailing += wse.Space()
			}
		} else {
			res = append(res, exprs[i])
		}
	}
	return res
}

func newLiteral(token []byte) expr {
	t := string(token)
	// Handle escape sequences
	if s, err := strconv.Unquote(`"` + t + `"`); err == nil {
		t = s
	}
	return &literalExpr{t, ""}
}

func defaultNewExpr(token []byte) expr {
	name, format, ok := strings.Cut(string(token), ":")
	if expr, ok := tryNopExpr(name); ok {
		return expr
	}

	if !ok {
		return &formatExpr{name: name, format: ""}
	}

	return &formatExpr{name: name, format: format}
}

func recursiveNewExpr(start, end string) func([]byte) expr {
	return func(token []byte) expr {
		var result, current expr

		// Splitting on recursive tokens should give artifacts that
		// indicate which ones (except for the first one which is always
		// interpretted as var)
		//
		// %(token:%(fallback_token:literal))
		//		-> token, '%(fallback_token', 'literal)'
		//
		for i, tt := range strings.Split(string(token), ":") {
			name, isExpr := strings.CutPrefix(tt, start)
			name = strings.TrimRight(name, end)

			var newCurrent expr
			if isExpr || i == 0 {
				newCurrent = &fallbackExpr{name: name, fallback: empty}
			} else {
				newCurrent = literalExpr{text: name}
			}

			if i == 0 {
				result = newCurrent
			} else {
				current.(*fallbackExpr).fallback = newCurrent
			}

			current = newCurrent

			if _, isFallback := newCurrent.(*fallbackExpr); !isFallback {
				break
			}

		}

		return result
	}
}

func tryNopExpr(name string) (expr, bool) {
	switch name {
	case "space":
		return space, true
	case "newline":
		return newline, true
	case "tab":
		return tab, true
	case "empty":
		return empty, true
	}
	return nil, false
}
