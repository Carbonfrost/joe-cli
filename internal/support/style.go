// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package support

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/juju/ansiterm"
	"golang.org/x/term"
)

type StyleWriter interface {
	io.Writer
	io.StringWriter

	ClearStyle(ansiterm.Style)
	Reset()
	SetColorCapable(bool)
	ResetColorCapable()
	ColorCapable() bool
	SetBackground(ansiterm.Color)
	SetForeground(ansiterm.Color)
	SetStyle(ansiterm.Style)
	Underline(...any) (int, error)
	Bold(...any) (int, error)
	Styled(ansiterm.Style, ...any) (int, error)
}

type ColorCapableWriter interface {
	ColorCapable() bool
	SetColorCapable(bool)
}

type stringHelper struct {
	*ansiterm.Writer
	enabled bool
}

func (w *stringHelper) WriteString(s string) (int, error) {
	return w.Writer.Write([]byte(s))
}

func (w *stringHelper) ResetColorCapable() {
	w.SetColorCapable(ColorEnabled(w.Writer))
}

func (w *stringHelper) ColorCapable() bool {
	return w.enabled
}

func (w *stringHelper) SetColorCapable(value bool) {
	w.enabled = value
	w.Writer.SetColorCapable(value)
}

func (w *stringHelper) Styled(style ansiterm.Style, v ...any) (int, error) {
	w.SetStyle(style)
	n, err := fmt.Fprint(w, v...)
	w.Reset()
	return n, err
}

func (w *stringHelper) Underline(v ...any) (int, error) {
	return w.Styled(ansiterm.Underline, v...)
}

func (w *stringHelper) Bold(v ...any) (int, error) {
	return w.Styled(ansiterm.Bold, v...)
}

func NewWriter(w io.Writer) StyleWriter {
	return &stringHelper{
		Writer:  ansiterm.NewWriter(w),
		enabled: ColorEnabled(w),
	}
}

func FormatDefault(fn func(StyleWriter)) string {
	return Format(ColorEnabled(os.Stdout), fn)
}

func Format(colorEnabled bool, fn func(StyleWriter)) string {
	res := bytes.NewBuffer(nil)
	w := NewWriter(res)
	w.SetColorCapable(colorEnabled)
	fn(w)
	return res.String()
}

func ColorEnabled(w io.Writer) bool {
	if s, ok := w.(ColorCapableWriter); ok {
		return s.ColorCapable()
	}

	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	// https://no-color.org/, which requires any value to be treated as true
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return os.Getenv("TERM") != "dumb" && term.IsTerminal(int(f.Fd()))
}
