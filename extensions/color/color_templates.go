package color

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
)

type templateContext struct {
	c *cli.Context
}

func templateFuncs(c *cli.Context) map[string]interface{} {
	t := &templateContext{c}
	bold := t.sprintStyle(cli.Bold)
	underline := t.sprintStyle(cli.Underline)
	return map[string]interface{}{
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

		"Bold":          bold,
		"Faint":         t.sprintStyle(cli.Faint),
		"Italic":        t.sprintStyle(cli.Italic),
		"Underline":     underline,
		"Blink":         t.sprintStyle(cli.Blink),
		"Reverse":       t.sprintStyle(cli.Reverse),
		"Strikethrough": t.sprintStyle(cli.Strikethrough),
		"Conceal":       t.sprintStyle(cli.Conceal),
		"Reset":         t.reset(),
	}
}

func (t *templateContext) newBuffer() *buffer {
	res := new(bytes.Buffer)
	return &buffer{
		Writer: t.c.Stdout,
		res:    res,
	}
}
func (t *templateContext) sprintStyle(s cli.Style) func(...interface{}) string {
	return func(a ...interface{}) string {
		res := t.newBuffer()
		text := fmt.Sprint(a...)
		if len(a) > 0 && len(text) == 0 {
			return ""
		}

		res.SetStyle(s)
		if len(a) > 0 {
			fmt.Fprint(res, text)
			res.Reset()
		}
		return res.String()
	}
}

func (t *templateContext) reset() func() string {
	return func() string {
		res := t.newBuffer()
		res.Reset()
		return res.String()
	}
}

func (t *templateContext) sprintForegroundColor(f cli.Color) func(...interface{}) string {
	return func(a ...interface{}) string {
		res := t.newBuffer()
		text := fmt.Sprint(a...)
		if len(a) > 0 && len(text) == 0 {
			return ""
		}

		res.SetForeground(f)
		if len(a) > 0 {
			fmt.Fprint(res, text)
			res.SetForeground(cli.Default)
		}
		return res.String()
	}
}

func (t *templateContext) resetColor() func() string {
	return func() string {
		res := t.newBuffer()
		res.SetForeground(cli.Default)
		return res.String()
	}
}
