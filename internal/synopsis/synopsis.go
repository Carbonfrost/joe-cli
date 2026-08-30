// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package synopsis

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/Carbonfrost/joe-cli/internal/support"
	"github.com/juju/ansiterm"
)

type styleWriter interface {
	io.Writer
	io.StringWriter
	SetForeground(ansiterm.Color)
	SetStyle(ansiterm.Style)
	Reset()
	Styled(ansiterm.Style, ...any) (int, error)
}

type Stringer interface {
	WriteTo(styleWriter)
}

type argCounterUsage interface {
	Usage() (bool, bool)
}

type Command struct {
	Name         string
	Flags        map[OptionGroup][]*Flag
	Args         []*Arg
	RequiredArgs []*Arg
	OptionalArgs []*Arg
	RTL          bool
	Style        Style

	// CondenseCategories causes flags with a SynopsisCategory set to be condensed
	// into groups
	CondenseCategories bool
}

type Flag struct {
	Short          string
	Shorts         []rune
	Long           string
	Primary        string // Long if present, otherwise Short
	Separator      string
	Names          []string
	AlternateNames []string
	Value          *Value
	Group          OptionGroup
	Style          Style

	// Category names the synopsis category that the flag belongs to, which is
	// empty for most flags.  See Command.CondenseCategories.
	Category string

	OptionalValue bool
}

type Arg struct {
	Value    string
	Multi    bool
	Optional bool
}

type Expr struct {
	Long         string
	Short        string
	Usage        *Usage
	Names        []string
	Args         []*Arg
	RequiredArgs []*Arg
	OptionalArgs []*Arg
	Style        Style
}

type Value struct {
	Placeholder string
	helpText    string
	Usage       *Usage

	derived bool // true when inferred from the type of the value rather than authored by user
}

type OptionGroup int

const (
	OnlyShortNoValue         = OptionGroup(iota) // -v
	OnlyShortNoValueOptional                     // [-v]
	OtherOptional                                // [--long=value]
	Other                                        // --long=value
	ActionGroup                                  // { --help|--version}
	Hidden
)

const (
	Bold      = ansiterm.Bold
	Underline = ansiterm.Underline
)

const (
	ColorData    = "_synopsisColor"
	StyleData    = "_synopsisStyle"
	CategoryData = "_synopsisCategory"
)

func Choices(opts []string) string {
	if len(opts) <= 1 {
		return strings.Join(opts, "")
	}

	slices.Sort(opts)
	if len(opts) > 3 {
		opts = append(opts[:3], "...")
	}
	return fmt.Sprintf("{%v}", strings.Join(opts, "|"))
}

// Style controls how the name of a flag, command, or expression is rendered in
// the synopsis.  The zero value renders the name in Bold.  When a color is set,
// it is used in place of the default Bold style unless a style is also set
// explicitly, in which case both are applied.
type Style struct {
	color ansiterm.Color
	style ansiterm.Style
}

func StyleFromData(data map[string]any) Style {
	var s Style
	if data == nil {
		return s
	}
	if c, ok := data[ColorData].(ansiterm.Color); ok {
		s.color = c
	}
	if st, ok := data[StyleData].(ansiterm.Style); ok {
		s.style = st
	}
	return s
}

func (s Style) write(w styleWriter, text string) {
	if s.color == 0 {
		style := s.style
		if style == 0 {
			style = Bold
		}
		w.Styled(style, text)
		return
	}

	w.SetForeground(s.color)
	if s.style != 0 {
		w.SetStyle(s.style)
	}
	w.WriteString(text)
	w.Reset()
}

// CategoryFromData obtains the synopsis category which was set within the data of a
// flag, which is the empty string if the flag has no synopsis category.
func CategoryFromData(data map[string]any) string {
	if data == nil {
		return ""
	}
	name, _ := data[CategoryData].(string)
	return name
}

func (s Style) ApplyTo(str string) string {
	return support.FormatDefault(func(sw support.StyleWriter) {
		s.write(sw, str)
	})
}

func NewCommand(name string, flags []*Flag, args []*Arg, rtl bool) *Command {
	groups := map[OptionGroup][]*Flag{
		OnlyShortNoValue:         {},
		OnlyShortNoValueOptional: {},
		OtherOptional:            {},
		ActionGroup:              {},
		Other:                    {},
		Hidden:                   {},
	}

	for _, f := range flags {
		groups[f.Group] = append(groups[f.Group], f)
	}

	sortedByName(groups[OnlyShortNoValueOptional])
	sortedByName(groups[OnlyShortNoValue])

	var required []*Arg
	var optional []*Arg

	if rtl {
		var start int
		for i, p := range args {
			if p.Optional {
				start = i
				break
			}
		}
		required = args[0:start]
		optional = args[start:]
	} else {
		for _, p := range args {
			if p.Optional {
				optional = append(optional, p)
			} else {
				required = append(required, p)
			}
		}
	}

	return &Command{
		Name:               name,
		Flags:              groups,
		Args:               args,
		RequiredArgs:       required,
		OptionalArgs:       optional,
		RTL:                rtl,
		CondenseCategories: true,
	}
}

func NewArg(usage string, narg any) *Arg {
	opt, mul := ArgCounter(narg)
	return &Arg{
		Value:    usage,
		Multi:    mul,
		Optional: opt,
	}
}

func NewFlag(primary string, aliases []string, helpText, usageString string, v any, group OptionGroup) *Flag {
	sep := "="

	long, short := canonicalNames(primary, aliases)

	if len(long) == 0 {
		sep = " "
	}
	value := getValueSynopsis(helpText, usageString, v)
	if len(value.Placeholder) == 0 {
		sep = ""
	}

	return (&Flag{
		Separator: sep,
		Value:     value,
		Group:     group,
	}).withLongAndShort(long, short)
}

func NewExpr(primary string, aliases []string, usage *Usage, args []*Arg) *Expr {
	long, short := canonicalNames(primary, aliases)
	names := func() []string {
		if len(long) == 0 {
			return []string{fmt.Sprintf("-%s", string(short[0]))}
		}
		if len(short) == 0 {
			return []string{fmt.Sprintf("-%s", long[0])}
		}
		return []string{fmt.Sprintf("-%s", string(short[0])), fmt.Sprintf("-%s", string(long[0]))}
	}

	return &Expr{
		Long:         longName(long),
		Short:        shortName(short),
		Usage:        usage,
		Args:         args,
		RequiredArgs: args,
		Names:        names(),
	}
}

func ArgCounter(narg any) (optional, multi bool) {
	switch c := narg.(type) {
	case int:
		return c == 0, c < 0 || c > 1
	case nil:
		return true, false
	case argCounterUsage:
		return c.Usage()
	}
	return false, false
}

func (v *Value) WriteTo(w styleWriter) {
	if v == nil {
		return
	}
	w.Styled(Underline, v.Placeholder)
}

func (c *Command) WriteTo(sb styleWriter) {
	c.Style.write(sb, c.Name)

	groups, categories := c.flagGroups()

	if flags := groups[ActionGroup]; len(flags) > 0 {
		sb.WriteString(" {")
		for i, f := range flags {
			if i > 0 {
				sb.WriteString(" | ")
			}
			f.primaryWriteTo(sb)
		}
		sb.WriteString("}")
	}

	if flags := groups[OnlyShortNoValue]; len(flags) > 0 {
		sb.WriteString(" -")
		for _, f := range flags {
			sb.WriteString(f.Short)
		}
	}

	if flags := groups[OnlyShortNoValueOptional]; len(flags) > 0 {
		sb.WriteString(" [-")
		for _, f := range flags {
			sb.WriteString(f.Short)
		}
		sb.WriteString("]")
	}

	if flags := groups[OtherOptional]; len(flags) > 0 {
		for _, f := range flags {
			sb.WriteString(" [")
			f.primaryWriteTo(sb)
			sb.WriteString("]")
		}
	}

	if flags := groups[Other]; len(flags) > 0 {
		for _, f := range flags {
			sb.WriteString(" ")
			f.primaryWriteTo(sb)
		}
	}

	for _, category := range categories {
		sb.WriteString(" { ")
		sb.Styled(Underline, category+"-flags")
		sb.WriteString(" }")
	}

	writeArgList(sb, c.RTL, c.RequiredArgs, c.OptionalArgs)
}

// flagGroups obtains the flags to display within the synopsis along with the names of
// the synopsis categories, sorted by name, which stand in for the flags they contain.
// Unless CondenseCategories is set, the flags are used as-is and no categories are
// identified.
func (c *Command) flagGroups() (map[OptionGroup][]*Flag, []string) {
	if !c.CondenseCategories {
		return c.Flags, nil
	}

	var (
		groups     = make(map[OptionGroup][]*Flag, len(c.Flags))
		seen       = map[string]bool{}
		categories []string
	)

	for group, flags := range c.Flags {
		if group == Hidden {
			groups[group] = flags
			continue
		}

		res := make([]*Flag, 0, len(flags))
		for _, f := range flags {
			if f.Category == "" {
				res = append(res, f)
				continue
			}
			if !seen[f.Category] {
				seen[f.Category] = true
				categories = append(categories, f.Category)
			}
		}
		groups[group] = res
	}

	sort.Strings(categories)
	return groups, categories
}

func (a *Arg) WriteTo(sb styleWriter) {
	if a.Multi {
		sb.WriteString(a.Value + "...")
	} else {
		sb.WriteString(a.Value)
	}
}

func (a *Arg) WithUsage(text string) *Arg {
	a.Value = text
	return a
}

func (f *Flag) WriteTo(sb styleWriter) {
	f.Style.write(sb, strings.Join(f.Names, ", "))
	f.valueWriteTo(sb)
}

func (f *Flag) primaryWriteTo(sb styleWriter) {
	f.Style.write(sb, f.Primary)
	f.valueWriteTo(sb)
}

func (f *Flag) valueWriteTo(sb styleWriter) {
	if !f.OptionalValue {
		sb.WriteString(f.Separator)
		f.Value.WriteTo(sb)
		return
	}

	sb.WriteString("[")

	// Trimming from separator applies to short flags so that the value runs in (-s[TLS1.2])
	sb.WriteString(strings.TrimSpace(f.Separator))
	f.Value.WriteTo(sb)
	sb.WriteString("]")
}

func (f *Flag) WithOptionalValue(text string) {
	if f.Value == nil || f.Value.Placeholder == "" {
		return
	}

	f.OptionalValue = true
	if text != "" && f.Value.derived {
		f.Value.Placeholder = text
	}
}

func (f *Flag) WithNo() {
	_ = f.withLongAndShort(
		[]string{"[no-]" + f.Long},
		f.Shorts,
	)
}

func (f *Flag) withLongAndShort(long []string, short []rune) *Flag {
	var primary string

	if len(long) == 0 {
		primary = optionName(string(short[0]))
	} else {
		primary = optionName(long[0])
	}

	var (
		shorts = func() []string {
			res := make([]string, len(short))
			for i := range short {
				res[i] = "-" + string(short[i])
			}
			return res
		}()
		longs = func() []string {
			res := make([]string, len(long))
			for i := range long {
				res[i] = optionName(long[i])
			}
			return res
		}()

		names, alternateNames = func() ([]string, []string) {
			if len(longs) == 0 {
				return []string{shorts[0]}, shorts[1:]
			}
			if len(short) == 0 {
				return []string{longs[0]}, longs[1:]
			}
			return []string{shorts[0], longs[0]}, append(shorts[1:], longs[1:]...)
		}()
	)
	f.Short = shortName(short)
	f.Shorts = short
	f.Long = longName(long)
	f.Primary = primary
	f.Names = names
	f.AlternateNames = alternateNames
	return f
}

func (e *Expr) WriteTo(sb styleWriter) {
	styleFirst(sb, e.Style, e.Names)
	writeArgList(sb, false, e.RequiredArgs, e.OptionalArgs)
}

func styleFirst(sb styleWriter, s Style, n []string) {
	if len(n) > 0 {
		s.write(sb, n[0])
		for _, name := range n[1:] {
			sb.WriteString(", ")
			sb.WriteString(name)
		}
	}
}

func writeArgList(sb styleWriter, rtl bool, req, opt []*Arg) {
	for _, arg := range req {
		sb.WriteString(" ")
		arg.WriteTo(sb)
	}

	if len(opt) > 0 {
		sb.WriteString(" ")
		for i, arg := range opt {
			if rtl {
				if i == 0 {
					sb.WriteString(strings.Repeat("[", len(opt)))
				} else {
					sb.WriteString(" ")
				}
			} else {
				sb.WriteString("[")
			}
			arg.WriteTo(sb)
			sb.WriteString("]")
		}
	}
}

func optionName(name any) string {
	switch n := name.(type) {
	case rune:
		if n == '-' {
			return "-"
		}
		return "-" + string(n)
	case string:
		if len(n) == 1 {
			return "-" + string(n)
		}
		return "--" + n
	}
	panic("unreachable!")
}

func longName(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func shortName(s []rune) string {
	if len(s) == 0 {
		return ""
	}
	return string(s[0])
}

func getValueSynopsis(helpText, usageString string, value any) *Value {
	usage := ParseUsage(helpText)
	placeholders := strings.Join(usage.Placeholders(), " ")

	if usageString != "" {
		return &Value{
			Placeholder: usageString,
			helpText:    usage.WithoutPlaceholders(),
			Usage:       usage,
		}
	}

	if len(placeholders) > 0 {
		return &Value{
			Placeholder: placeholders,
			Usage:       usage,
			helpText:    usage.WithoutPlaceholders(),
		}
	}
	_, provided := value.(valueProvidesSynopsis)
	return &Value{
		Placeholder: Placeholder(value),
		helpText:    usage.WithoutPlaceholders(),
		Usage:       usage,
		derived:     !provided,
	}
}

func sortedByName(flags []*Flag) {
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Short < flags[j].Short
	})
}

func canonicalNames(name string, aliases []string) (long []string, short []rune) {
	long = make([]string, 0, len(aliases))
	short = make([]rune, 0, len(aliases))
	names := append([]string{name}, aliases...)

	for _, nom := range names {
		if len(nom) == 1 {
			short = append(short, ([]rune(nom))[0])
		} else {
			long = append(long, nom)
		}
	}
	return
}

var (
	_ Stringer = (*Value)(nil)
	_ Stringer = (*Flag)(nil)
	_ Stringer = (*Arg)(nil)
	_ Stringer = (*Expr)(nil)
)
