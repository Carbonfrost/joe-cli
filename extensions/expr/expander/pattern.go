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

// Syntax selects the expression syntax used when compiling a pattern.
// A Syntax value is itself an Option.
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

// Pattern is a compiled expandable expression produced by Compile. A Pattern
// may be expanded repeatedly with different expanders.
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
// files and the corresponding variable support.
// When used as a writer to Fprint, two control expressions
// are exposed, stdout and stderr, which can be used to redirect
// to the underlying writers. For example, "%(stderr)debug: %(v:#v)%(newline)%(stdout)%(v)"
// would print debug text to whatever writer was set for stderr.
type Renderer struct {
	io.Writer
	out, err io.Writer
}

// Expander gets the expander for Renderer, which exposes control expressions, stdout and stderr, which switch the writer to use
func (r *Renderer) Expander() Interface {
	return Func(r.expandFiles)
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

// NewRenderer creates a Renderer that writes to stdout by default and exposes
// the stdout and stderr control expressions for switching the active writer.
func NewRenderer(stdout, stderr io.Writer) *Renderer {
	return &Renderer{
		Writer: stdout,
		out:    stdout,
		err:    stderr,
	}
}

// Expands expands the pattern using the given expander and produces a string
func Expand(pattern string, e Interface, opts ...Option) string {
	return Compile(pattern, opts...).Expand(e)
}

// Fprint expands the pattern using the given expander and writes to the specified writer.
func Fprint(pattern string, w io.Writer, e Interface, opts ...Option) (count int, err error) {
	return Compile(pattern, opts...).Fprint(w, e)
}

// Option configures how a pattern is compiled by Compile. The Syntax
// values are themselves options
type Option interface {
	apply(*options)
}

type options struct {
	syntax Syntax
	start  string
	end    string
	metas  []metaOption
}

type metaOption struct {
	name    string
	pattern *Pattern
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) { f(o) }

func (s Syntax) apply(o *options) { o.syntax = s }

// WithDelimiters overrides the start and end delimiters used to recognize
// expressions within a pattern. The end delimiter must be a single byte.
// The default delimiters are "%(" and ")".
func WithDelimiters(start, end string) Option {
	return optionFunc(func(o *options) {
		o.start = start
		o.end = end
	})
}

// WithMeta substitutes every expression named name with the expressions
// from the given pattern. It is applied after the pattern is compiled and
// may be specified more than once.
func WithMeta(name string, pattern *Pattern) Option {
	return optionFunc(func(o *options) {
		o.metas = append(o.metas, metaOption{name: name, pattern: pattern})
	})
}

// Compile compiles pattern into a Pattern, applying any options. By default
// the SyntaxDefault syntax and the delimiters "%(" and ")" are used.
func Compile(pattern string, opts ...Option) *Pattern {
	o := options{
		start: "%(",
		end:   ")",
	}
	for _, opt := range opts {
		opt.apply(&o)
	}

	endBytes := []byte(o.end)
	if len(endBytes) > 1 {
		panic("end sequence must be one byte")
	}

	newExpr := defaultNewExpr
	if o.syntax == SyntaxRecursive {
		newExpr = recursiveNewExpr(o.start, o.end)
	}

	p := compilePatternCore([]byte(pattern), []byte(o.start), endBytes, newExpr)
	for _, m := range o.metas {
		p = p.WithMeta(m.name, m.pattern)
	}
	return p
}

// CompilePattern compiles pattern using the given delimiters.
//
// Deprecated: Use Compile with WithDelimiters instead.
func CompilePattern(pattern, start, end string) *Pattern {
	return Compile(pattern, WithDelimiters(start, end))
}

// Compile compiles pattern using the receiver's syntax.
//
// Deprecated: Use Compile with the Syntax value as an option instead.
func (s Syntax) Compile(pattern string) *Pattern {
	return Compile(pattern, s)
}

// CompilePattern compiles pattern using the receiver's syntax and the given
// delimiters.
//
// Deprecated: Use Compile with the Syntax value and WithDelimiters instead.
func (s Syntax) CompilePattern(pattern, start, end string) *Pattern {
	return Compile(pattern, s, WithDelimiters(start, end))
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
	value := expand.Expand(f.name)
	return formatStr(value, f.format, f.trailingOpt)
}

func formatStr(value any, format, trailingOpt string) string {
	var res string
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
	return res + trailingOpt
}

func (f fallbackExpr) Format(expand Interface) string {
	value := expand.Expand(f.name)
	if value == nil {
		return f.fallback.Format(expand)
	}

	// The "fallback" is treated as a format string
	var format string
	if lit, ok := f.fallback.(literalExpr); ok {
		if f, ok := strings.CutPrefix(lit.text, "%"); ok {
			format = f
		}
	}

	res := formatStr(value, format, f.trailingOpt)
	if res == "" {
		return res
	}
	return res + f.trailingOpt
}

// AppendText implements [encoding.TextAppender]. The output
// matches that of calling the [Pattern.String] method.
func (p *Pattern) AppendText(b []byte) ([]byte, error) {
	return append(b, p.String()...), nil
}

// MarshalText implements [encoding.TextMarshaler]. The output
// matches that of calling the [Pattern.AppendText] method.
func (p *Pattern) MarshalText() ([]byte, error) {
	return p.AppendText(nil)
}

// UnmarshalText implements [encoding.TextUnmarshaler] by calling
// [Compile] on the encoded value using the default delimiters
func (p *Pattern) UnmarshalText(text []byte) error {
	newPattern := Compile(string(text))
	*p = *newPattern
	return nil
}

// Fprint expands the pattern using the given expander and writes to the specified writer.
// As a special case, if w has a method Expander() Interface, this
// method will be called to obtain an expander which composes with
// the expander e. The main use of this convention is to allow writers
// to supply control expressions. For an example, see [Renderer].
func (p *Pattern) Fprint(w io.Writer, e Interface) (count int, err error) {
	// Implicitly upgrade w to *Renderer, etc.
	if r, ok := w.(interface{ Expander() Interface }); ok {
		e = Compose(r.Expander(), e)
	}
	for _, item := range p.exprs {
		count, err = fmt.Fprint(w, item.Format(e))
		if err != nil {
			break
		}
	}
	return
}

// Expand expands the pattern with the given replacements
func (p *Pattern) Expand(expand Interface) string {
	var b strings.Builder
	p.Fprint(&b, expand)
	return b.String()
}

// ExpandAny expands the pattern, returning the underlying type of the expansion
// when the pattern consists of a single expression with no format specifier applied..
// When the pattern has multiple expressions, a literal fallback, or a format specifier,
// it falls back to string interpolation like Expand.
func (p *Pattern) ExpandAny(expand Interface) any {
	if len(p.exprs) != 1 {
		return p.Expand(expand)
	}
	return exprExpandAny(p.exprs[0], expand)
}

func exprExpandAny(e expr, expand Interface) any {
	switch exp := e.(type) {
	case *formatExpr:
		if exp.format != "" || exp.trailingOpt != "" {
			return exp.Format(expand)
		}
		return expand.Expand(exp.name)

	case *fallbackExpr:
		if _, ok := exp.fallback.(literalExpr); ok {
			return exp.Format(expand)
		}
		v := expand.Expand(exp.name)
		if v != nil {
			if exp.trailingOpt != "" {
				return exp.Format(expand)
			}
			return v
		}
		return exprExpandAny(exp.fallback, expand)

	default:
		return e.Format(expand)
	}
}

// String implements fmt.Stringer, reconstructing the pattern's source
// representation using its delimiters.
func (p *Pattern) String() string {
	var sb strings.Builder
	for _, e := range p.exprs {
		fprintRepr(e, &sb, p.start, p.end)
	}
	return sb.String()
}

// WithMeta returns a copy of the pattern with every expression named name
// replaced by the expressions from pattern. See the WithMeta Option for the
// equivalent applied at compile time.
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
