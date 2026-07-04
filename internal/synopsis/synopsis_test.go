// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package synopsis_test

import (
	"fmt"
	"strings"

	"github.com/Carbonfrost/joe-cli/internal/synopsis"
	"github.com/juju/ansiterm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("String", func() {

	DescribeTable("examples", func(s synopsis.Stringer, expected types.GomegaMatcher) {
		w := &Writer{}
		s.WriteTo(w)
		Expect(w.String()).To(expected)
	},

		Entry(
			"flag",
			&synopsis.Flag{Names: []string{"-a"}},
			Equal("**-a**"),
		),
		Entry(
			"flag multi",
			&synopsis.Flag{Names: []string{"-a", "-b"}, Separator: "=", Value: basicValue},
			Equal("**-a, -b**=_STRING_"),
		),
		Entry(
			"flag with style override",
			&synopsis.Flag{
				Names: []string{"-a"},
				Style: synopsis.StyleFromData(map[string]any{synopsis.StyleData: ansiterm.Underline}),
			},
			Equal("_-a_"),
		),
		Entry(
			"flag with color override",
			&synopsis.Flag{
				Names: []string{"-a"},
				Style: synopsis.StyleFromData(map[string]any{synopsis.ColorData: ansiterm.Red}),
			},
			Equal("{red}-a{reset}"),
		),
		Entry(
			"flag with color and style override",
			&synopsis.Flag{
				Names: []string{"-a"},
				Style: synopsis.StyleFromData(map[string]any{
					synopsis.ColorData: ansiterm.Red,
					synopsis.StyleData: ansiterm.Underline,
				}),
			},
			Equal("{red}<underline>-a{reset}"),
		),
		Entry(
			"command name with color override",
			&synopsis.Command{
				Name:  "c",
				Style: synopsis.StyleFromData(map[string]any{synopsis.ColorData: ansiterm.Red}),
			},
			ContainSubstring("{red}c{reset}"),
		),
		Entry("action flags",
			synopsis.NewCommand("c",
				[]*synopsis.Flag{
					synopsis.NewFlag("help", nil, "", "", "", synopsis.ActionGroup),
					synopsis.NewFlag("version", nil, "", "", "", synopsis.ActionGroup),
				}, nil, false),
			ContainSubstring("{**--help**=_VALUE_ | **--version**=_VALUE_}")),

		Entry("inline optional flags",
			synopsis.NewCommand("c",
				[]*synopsis.Flag{
					synopsis.NewFlag("a", nil, "", "", "", synopsis.OnlyShortNoValueOptional),
					synopsis.NewFlag("b", nil, "", "", "", synopsis.OnlyShortNoValueOptional),
				}, nil, false),
			ContainSubstring("[-ab]")),

		Entry("other flags",
			synopsis.NewCommand("c",
				[]*synopsis.Flag{
					synopsis.NewFlag("normal", nil, "", "", "", synopsis.Other),
				}, nil, false),
			ContainSubstring("--normal")),

		Entry(
			"arg",
			&synopsis.Arg{Value: "STRING"},
			Equal("STRING"),
		),
		Entry(
			"arg multi",
			&synopsis.Arg{Value: "STRING", Multi: true},
			Equal("STRING..."),
		),
		Entry(
			"arg optioanl",
			&synopsis.Arg{Value: "STRING"},
			Equal("STRING"),
		),

		Entry("required and optional args",
			synopsis.NewCommand("c",
				nil,
				[]*synopsis.Arg{
					{
						Value:    "<required>",
						Optional: false,
					},
					{
						Value:    "<optional>",
						Optional: true,
					},
				}, false),
			ContainSubstring("<required> [<optional>]")),
	)
})

type Writer struct {
	strings.Builder
}

var basicValue = &synopsis.Value{Placeholder: "STRING"}

func (w *Writer) SetForeground(c ansiterm.Color) {
	fmt.Fprintf(w, "{%s}", c)
}

func (w *Writer) SetStyle(s ansiterm.Style) {
	fmt.Fprintf(w, "<%s>", s)
}

func (w *Writer) Reset() {
	w.WriteString("{reset}")
}

func (w *Writer) Styled(s ansiterm.Style, v ...any) (int, error) {
	var c string
	if s == synopsis.Underline {
		c = "_"
	} else if s == synopsis.Bold {
		c = "**"
	}
	w.WriteString(c)
	fmt.Fprint(w, v...)
	return w.WriteString(c)
}
