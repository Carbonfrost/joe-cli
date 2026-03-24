// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package color

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
)

type templateContext struct {
	funcs map[string]any

	colorEnabledThunk func() bool
}

type sprinter = func(...any) (string, error)

var (
	styleNames = map[string]cli.Style{
		"Bold":          cli.Bold,
		"Faint":         cli.Faint,
		"Italic":        cli.Italic,
		"Underline":     cli.Underline,
		"Blink":         cli.Blink,
		"Reverse":       cli.Reverse,
		"Strikethrough": cli.Strikethrough,
		"Conceal":       cli.Conceal,
	}

	colorNames = map[string]cli.Color{
		"Default":       cli.Default,
		"Black":         cli.Black,
		"Red":           cli.Red,
		"Green":         cli.Green,
		"Yellow":        cli.Yellow,
		"Blue":          cli.Blue,
		"Magenta":       cli.Magenta,
		"Cyan":          cli.Cyan,
		"Gray":          cli.Gray,
		"DarkGray":      cli.DarkGray,
		"BrightRed":     cli.BrightRed,
		"BrightGreen":   cli.BrightGreen,
		"BrightYellow":  cli.BrightYellow,
		"BrightBlue":    cli.BrightBlue,
		"BrightMagenta": cli.BrightMagenta,
		"BrightCyan":    cli.BrightCyan,
		"White":         cli.White,
	}
)

// NewTemplateFuncs gets the template funcs that support the
// given mode
func NewTemplateFuncs(modeopt ...Mode) map[string]any {
	result := func() bool {
		switch {
		case len(modeopt) == 0 || modeopt[0] == Auto:
			return support.ColorEnabled(os.Stdout)
		case modeopt[0] == Never:
			return false
		case modeopt[0] == Always:
		}
		return true
	}()
	return templateFuncs(func() bool {
		return result
	})
}

func templateFuncs(colorEnabled func() bool) map[string]any {
	t := &templateContext{colorEnabledThunk: colorEnabled}
	bold := t.sprintStyle(cli.Bold)
	underline := t.sprintStyle(cli.Underline)

	t.funcs = map[string]any{
		"Black":         t.sprintForegroundColor(cli.Black),
		"Red":           t.sprintForegroundColor(cli.Red),
		"Green":         t.sprintForegroundColor(cli.Green),
		"Yellow":        t.sprintForegroundColor(cli.Yellow),
		"Blue":          t.sprintForegroundColor(cli.Blue),
		"Magenta":       t.sprintForegroundColor(cli.Magenta),
		"Cyan":          t.sprintForegroundColor(cli.Cyan),
		"Gray":          t.sprintForegroundColor(cli.Gray),
		"DarkGray":      t.sprintForegroundColor(cli.DarkGray),
		"BrightRed":     t.sprintForegroundColor(cli.BrightRed),
		"BrightGreen":   t.sprintForegroundColor(cli.BrightGreen),
		"BrightYellow":  t.sprintForegroundColor(cli.BrightYellow),
		"BrightBlue":    t.sprintForegroundColor(cli.BrightBlue),
		"BrightMagenta": t.sprintForegroundColor(cli.BrightMagenta),
		"BrightCyan":    t.sprintForegroundColor(cli.BrightCyan),
		"White":         t.sprintForegroundColor(cli.White),
		"ResetColor":    t.resetColor(),
		"Color":         t.setColor,
		"Background":    t.setBackgroundColor,

		"Bold":          bold,
		"Faint":         t.sprintStyle(cli.Faint),
		"Italic":        t.sprintStyle(cli.Italic),
		"Underline":     underline,
		"Blink":         t.sprintStyle(cli.Blink),
		"Reverse":       t.sprintStyle(cli.Reverse),
		"Strikethrough": t.sprintStyle(cli.Strikethrough),
		"Conceal":       t.sprintStyle(cli.Conceal),
		"Reset":         t.reset(),
		"Style":         t.setStyle,

		"Emoji": t.emoji,

		"BoldFirst": func(s []string) []string {
			if len(s) == 0 {
				return s
			}
			first, _ := bold(s[0])
			return append([]string{first}, s[1:]...)
		},
	}
	return t.funcs
}

func (t *templateContext) sprintStyle(s cli.Style) sprinter {
	return func(a ...any) (string, error) {
		return t.format(
			func(w cli.Writer) { w.SetStyle(s) },
			func(w cli.Writer) { w.Reset() },
			a,
		)
	}
}

func (t *templateContext) reset() func() string {
	return func() string {
		return support.Format(t.colorEnabled(), (support.StyleWriter).Reset)
	}
}

func (t *templateContext) sprintForegroundColor(f cli.Color) sprinter {
	return func(a ...any) (string, error) {
		return t.format(
			func(w cli.Writer) { w.SetForeground(f) },
			func(w cli.Writer) { w.SetForeground(cli.Default) },
			a,
		)
	}
}

func (t *templateContext) resetColor() func() string {
	return func() string {
		return support.Format(t.colorEnabled(), func(sw support.StyleWriter) {
			sw.SetForeground(cli.Default)
		})
	}
}

func (t *templateContext) setColor(color string, a ...any) (string, error) {
	if _, ok := colorNames[color]; ok {
		return t.funcs[color].(sprinter)(a...)
	}
	return "", fmt.Errorf("not valid color: %q", color)
}

func (t *templateContext) setBackgroundColor(color string, a ...any) (string, error) {
	if f, ok := colorNames[color]; ok {
		return t.format(
			func(w cli.Writer) { w.SetBackground(f) },
			func(w cli.Writer) { w.SetBackground(cli.Default) },
			a,
		)
	}
	return "", fmt.Errorf("not valid color: %q", color)
}

func (t *templateContext) setStyle(styles string, a ...any) (string, error) {
	s := strings.Fields(styles)
	switch len(s) {
	case 0:
		return fmt.Sprint(a...), nil
	case 1:
		if _, ok := styleNames[s[0]]; ok {
			return t.funcs[s[0]].(sprinter)(a...)
		}
	default:
		all := make([]cli.Style, len(s))
		var ok bool
		for i, style := range s {
			if all[i], ok = styleNames[style]; !ok {
				return "", fmt.Errorf("not valid style: %q", styles)
			}
		}
		return t.format(
			func(w cli.Writer) {
				for _, style := range all {
					w.SetStyle(style)
				}
			},
			func(w cli.Writer) { w.Reset() },
			a,
		)
	}

	return "", fmt.Errorf("not valid style: %q", styles)
}

func (t *templateContext) format(on, off func(cli.Writer), a []any) (string, error) {
	buf := bytes.NewBuffer(nil)
	res := support.NewWriter(buf)
	res.SetColorCapable(t.colorEnabled())

	text := fmt.Sprint(a...)
	if len(a) > 0 && len(text) == 0 {
		return "", nil
	}

	on(res)
	if len(a) > 0 {
		fmt.Fprint(res, text)
		off(res)
	}
	return buf.String(), nil
}

func (t *templateContext) emoji(name string) (string, error) {
	if len(name) > 0 && t.colorEnabled() {
		res, ok := emojiByName[name]
		if !ok {
			return "", fmt.Errorf("not valid emoji: %q", name)
		}
		return string(res), nil
	}
	return "", nil
}

func (t *templateContext) colorEnabled() bool {
	return t.colorEnabledThunk()
}
