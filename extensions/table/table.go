// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Package table implements utilities for rendering tables and managing table data.
//
// The template funcs registered by this package are:
//
//   - Table [ string | *Format ]
//   - EndTable
//   - Headers <[]string>
//   - Footers <[]string>
//   - Row
//   - Cell <string>
//
// The Table function takes an optional argument which is the name of a format
// or a value to use for formatting the table. To conclude the table, use EndTable.
// Headers/Footers appears optionally to denote
// the names of the headers.  Row must be used to group together cells.
// The content of each cell is passed to the Cell function.
//
// "Default" and "Unformatted" are the built-in Formats
//
// A viable table looks like:
//
//	{{- Table "Unformatted" -}}
//	{{- Headers "First" "Last" }}
//	{{- Row -}}
//	{{- Cell "George" -}}
//	{{- Cell "Burdell" -}}
//	{{- EndTable -}}`
//
// Additional formats can be registered using the RegisterTableFormat action.
//
// When you want to display tabular data in the CLI, you can use the RegisterTable
// function to give the table a name and specify
package table

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/internal/support"
	"github.com/olekukonko/tablewriter"
)

// Options is the configuration for the table extension and provides the initializer for the
// app initialization pipeline
type Options struct {
	// Features enumerates which features to use
	Features Feature
}

// Feature provides a name for each feature in the extension
type Feature int

// Alignment defines how table cell contents are aligned
type Alignment int

// Format defines the optional format for tables
type Format struct {
	Alignment             Alignment
	AutoFormatHeaders     bool
	AutoWrapText          bool
	Border                bool
	CenterSeparator       string
	ColumnSeparator       string
	ColWidth              int
	HeaderAlignment       Alignment
	HeaderLine            bool
	NoWhiteSpace          bool
	RowLine               bool
	RowSeparator          string
	TablePadding          string
	TopLeftSeparator      string
	TopCenterSeparator    string
	TopRightSeparator     string
	CenterLeftSeparator   string
	CenterRightSeparator  string
	BottomLeftSeparator   string
	BottomCenterSeparator string
	BottomRightSeparator  string
}

// renderContext defines the operations of rendering the table
type renderContext interface {
	Row() string
	Headers(titles ...string) string
	Footers(titles ...string) string
	Cell(value any) (string, error)
	EndTable() string
}

const (
	// UseTablesInHelpTemplate causes tables to be used in the help template
	UseTablesInHelpTemplate = Feature(1 << iota)

	// TemplateFuncs enables the template funcs feature, which provides template funcs
	// for colors and styles
	TemplateFuncs

	// AllFeatures enables all of the features.  This is the default
	AllFeatures = -1
)

const (
	// Unformatted is the table format associated with a table with no boxes or alignments
	Unformatted = "Unformatted"

	// Porcelain is a table that only uses tabs to separate columns
	Porcelain = "Porcelain"
)

type tableContext struct {
	table   *tablewriter.Table
	headers []string
	footers []string
	cells   [][]string
	buf     *bytes.Buffer

	cornerSeparators [9]string // left->right, top->bottom
	centerSeparator  string
}

// Align constants
const (
	AlignDefault Alignment = tablewriter.ALIGN_DEFAULT
	AlignCenter            = tablewriter.ALIGN_CENTER
	AlignRight             = tablewriter.ALIGN_RIGHT
	AlignLeft              = tablewriter.ALIGN_LEFT
)

const (
	flagTemplate = `
{{- Row -}}
{{- Execute "FlagSynopsis" .Synopsis | ExtraSpaceBeforeFlag | Cell -}}
{{- Cell .HelpText -}}
`

	flagListingTemplate = `
{{- Table "Unformatted" -}}
{{- range . -}}{{- template "Flag" . -}}{{- end -}}
{{- EndTable -}}
`

	subCommandListingTemplate = `
{{- Table "Unformatted" -}}
{{- range . -}}
{{- Row -}}
{{- .Names | BoldFirst | Join ", " | Cell -}}
{{- Cell .HelpText -}}
{{- end -}}
{{- EndTable -}}`
)

var (
	errCellCalledWrongTime = fmt.Errorf(`"cell" template function called before "row" template function`)

	featureMap = cli.FeatureMap[Feature]{
		UseTablesInHelpTemplate: RegisterHelpTemplates(),
		TemplateFuncs:           RegisterTemplateFuncs(),
	}

	defaultFormat = &Format{
		Alignment:             AlignDefault,
		AutoFormatHeaders:     false,
		AutoWrapText:          true,
		Border:                true,
		ColWidth:              -1,
		HeaderAlignment:       AlignDefault,
		HeaderLine:            true,
		RowLine:               true,
		NoWhiteSpace:          false,
		TablePadding:          "",
		CenterSeparator:       "┼",
		ColumnSeparator:       "│",
		RowSeparator:          "─",
		TopLeftSeparator:      "┌",
		TopCenterSeparator:    "┬",
		TopRightSeparator:     "┐",
		CenterLeftSeparator:   "├",
		CenterRightSeparator:  "┤",
		BottomLeftSeparator:   "└",
		BottomCenterSeparator: "┴",
		BottomRightSeparator:  "┘",
	}

	unformatted = &Format{
		Alignment:         AlignDefault,
		AutoFormatHeaders: false,
		AutoWrapText:      true,
		Border:            false,
		CenterSeparator:   "",
		ColumnSeparator:   "",
		ColWidth:          -1,
		HeaderAlignment:   AlignDefault,
		HeaderLine:        false,
		RowLine:           false,
		NoWhiteSpace:      false,
		RowSeparator:      "",
		TablePadding:      " ",
	}
)

func newTableContext(f ...*Format) *tableContext {
	c := &tableContext{}
	c.buf = new(bytes.Buffer)
	c.table = tablewriter.NewWriter(c.buf)

	format := defaultFormat
	if len(f) > 0 {
		format = getFormat(f[0])
	}
	if len(f) > 1 {
		panic("expected zero or one arg")
	}
	if format.ColWidth <= 0 {
		format.ColWidth = support.GuessWidth()
	}
	format.apply(c)

	c.headers = nil
	c.footers = nil
	c.cells = nil
	return c
}

type porcelainContext struct {
	cells []string
}

// wrapperRenderContext delegates to the appropriate internal
// render context depending upon which format is used.  This is the
// render context exposed to the template function context
type wrapperRenderContext struct {
	inner renderContext
}

// RegisterTemplateFuncs provides the template functions described in the package
// overview:
//   - Table
//   - EndTable
//   - Headers
//   - Footers
//   - Row, and
//   - Cell
func RegisterTemplateFuncs() cli.Action {
	return cli.ActionFunc(func(c *cli.Context) error {
		tc := new(wrapperRenderContext)
		templateFuncs := map[string]any{
			"Table":    tc.Table,
			"EndTable": tc.EndTable,
			"Headers":  tc.Headers,
			"Footers":  tc.Footers,
			"Row":      tc.Row,
			"Cell":     tc.Cell,
		}
		for k, v := range templateFuncs {
			c.RegisterTemplateFunc(k, v)
		}
		return nil
	})
}

// RegisterHelpTemplates causes various templates used in the default help template
// to be overridden with ones that use tables
func RegisterHelpTemplates() cli.Action {
	return cli.Before(cli.Pipeline(
		cli.RegisterTemplate("Flag", flagTemplate),
		cli.RegisterTemplate("FlagListing", flagListingTemplate),
		cli.RegisterTemplate("SubcommandListing", subCommandListingTemplate),
	))
}

func (c *tableContext) Row() string {
	c.cells = append(c.cells, []string{})
	return ""
}

func (c *tableContext) Headers(titles ...string) string {
	c.headers = append(c.headers, titles...)
	return ""
}

func (c *tableContext) Footers(titles ...string) string {
	c.footers = append(c.footers, titles...)
	return ""
}

func (c *tableContext) Cell(value any) (string, error) {
	if len(c.cells) == 0 {
		return "", errCellCalledWrongTime
	}
	c.cells[len(c.cells)-1] = append(c.cells[len(c.cells)-1], fmt.Sprintf("%v", value))
	return "", nil
}

func (c *tableContext) EndTable() string {
	if len(c.headers) > 0 || len(c.cells) > 0 || len(c.footers) > 0 {
		c.table.SetHeader(c.headers)
		c.table.SetFooter(c.footers)
		c.table.AppendBulk(c.cells)
		c.table.Render()
	}

	// Backtrack to replace corners.  This is somewhat a hack until
	// the underlying package supports it
	nl := "\n"
	sep := c.centerSeparator
	lines := strings.Split(c.buf.String(), nl)
	last := len(lines) - 2

	for i := range lines {
		var sin int
		switch i {
		case 0:
			sin = 0
		case last:
			sin = 6
		default:
			sin = 3
		}
		if after, ok := strings.CutPrefix(lines[i], sep); ok {
			lines[i] = c.cornerSeparators[sin] + after
		}
		if strings.HasSuffix(lines[i], sep) {
			lines[i] = strings.TrimSuffix(lines[i], sep) + c.cornerSeparators[sin+2]
		}
		lines[i] = strings.ReplaceAll(lines[i], sep, c.cornerSeparators[sin+1])
	}

	return strings.Join(lines, nl)
}

func (f *Format) apply(tc *tableContext) {
	t := tc.table
	t.SetAutoWrapText(f.AutoWrapText)
	t.SetAutoFormatHeaders(f.AutoFormatHeaders)
	t.SetHeaderAlignment(int(f.HeaderAlignment))
	t.SetAlignment(int(f.Alignment))
	t.SetCenterSeparator(f.CenterSeparator)
	t.SetColumnSeparator(f.ColumnSeparator)
	t.SetRowSeparator(f.RowSeparator)
	t.SetHeaderLine(f.HeaderLine)
	t.SetRowLine(f.RowLine)
	t.SetBorder(f.Border)
	t.SetTablePadding(f.TablePadding)
	t.SetNoWhiteSpace(f.NoWhiteSpace)
	t.SetColWidth(f.ColWidth)

	if f.Border {
		tc.cornerSeparators = [...]string{
			f.TopLeftSeparator,
			f.TopCenterSeparator,
			f.TopRightSeparator,
			f.CenterLeftSeparator,
			f.CenterSeparator,
			f.CenterRightSeparator,
			f.BottomLeftSeparator,
			f.BottomCenterSeparator,
			f.BottomRightSeparator,
		}
		tc.centerSeparator = f.CenterSeparator
	}
}

func (f Feature) Pipeline() cli.Action {
	if f == 0 {
		f = AllFeatures
	}
	return featureMap.Pipeline(f)
}

func (o Options) Execute(c context.Context) error {
	return cli.FromContext(c).Do(o.Features.Pipeline())
}

func (p *porcelainContext) Row() string {
	var res string
	if p.cells != nil {
		res = strings.Join(p.cells, "\t") + "\n"
	}
	p.cells = []string{}
	return res
}

func (porcelainContext) Headers(titles ...string) string {
	return strings.Join(titles, "\t") + "\n"
}

func (porcelainContext) Footers(titles ...string) string {
	return strings.Join(titles, "\t") + "\n"
}

func (p *porcelainContext) Cell(value any) (string, error) {
	p.cells = append(p.cells, fmt.Sprint(value))
	return "", nil
}

func (p *porcelainContext) EndTable() string {
	return p.Row()
}

func (c *wrapperRenderContext) Table(f ...any) (string, error) {
	switch len(f) {
	case 0:
		c.inner = newTableContext(defaultFormat)
	case 1:
		if f[0] == Porcelain {
			c.inner = &porcelainContext{}
		} else {
			c.inner = newTableContext(getFormat(f[0]))
		}
	default:
		return "", fmt.Errorf("func Table expects 0 or 1 arguments")
	}
	return "", nil
}

func (c *wrapperRenderContext) Row() string {
	return c.inner.Row()
}

func (c *wrapperRenderContext) Headers(titles ...string) string {
	return c.inner.Headers(titles...)
}

func (c *wrapperRenderContext) Footers(titles ...string) string {
	return c.inner.Footers(titles...)
}

func (c *wrapperRenderContext) Cell(value any) (string, error) {
	return c.inner.Cell(value)
}

func (c *wrapperRenderContext) EndTable() string {
	return c.inner.EndTable()
}

func getFormat(v any) *Format {
	switch f := v.(type) {
	case *Format:
		return f
	case string:
		if f == Unformatted {
			return unformatted
		}
		return defaultFormat
	default:
		panic(fmt.Sprintf("unexpected type: %T", v))
	}
}

var (
	_ cli.Action    = (*Options)(nil)
	_ renderContext = (*tableContext)(nil)
)
