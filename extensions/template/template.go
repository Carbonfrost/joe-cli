// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template is used to create files from template file systems
package template

import (
	"context"
	"io"
	tt "text/template"

	"github.com/Carbonfrost/joe-cli"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Generator

// Root is the root of a template, used to compose a sequence and
// configuration
type Root struct {
	Sequence         Sequence
	Overwrite        bool
	DryRun           bool
	WorkingDirectory string
}

// Sequence is a sequence of template generators
type Sequence []Generator

// Generator is the interface for generating files.
type Generator interface {
	Generate(ctx context.Context, c *OutputContext) error
}

// Vars contains template variables.  Variables are copied into the template
// context
type Vars map[string]any

// Interface provides the interface of a template.  The primary implementation
// is usually from the text/template package.
type Interface interface {
	Execute(out io.Writer, data any) error
}

type dataSetter struct {
	name  string
	value any
}

func New(items ...Generator) *Root {
	return &Root{
		Sequence: items,
	}
}

func Data(name string, value any) Generator {
	return &dataSetter{name, value}
}

// SetOverwrite sets whether to overwrite.  This function is for
// bindings
func (r *Root) SetOverwrite(v bool) error {
	r.Overwrite = v
	return nil
}

// SetDryRun sets whether to do a dry run.  This function is for
// bindings
func (r *Root) SetDryRun(v bool) error {
	r.DryRun = v
	return nil
}

// OverwriteFlag obtains a conventions-based flag for overwriting
func (r *Root) OverwriteFlag() cli.Prototype {
	return cli.Prototype{
		Name:     "overwrite",
		HelpText: "Overwrite files",
		Setup: cli.Setup{
			Uses: cli.Bind(r.SetOverwrite),
		},
	}
}

// DryRunFlag obtains a conventions-based flag for overwriting
func (r *Root) DryRunFlag() cli.Prototype {
	return cli.Prototype{
		Name:     "dry-run",
		HelpText: "Display what commands will be run without actually executing them",
		Setup: cli.Setup{
			Uses: cli.Bind(r.SetDryRun),
		},
	}
}

func (r *Root) Execute(ctx context.Context) error {
	return cli.Do(ctx, r.pipeline())
}

func (r *Root) pipeline() cli.Action {
	return cli.Setup{
		Optional: true,
		Uses: cli.AddFlags([]*cli.Flag{
			{Uses: r.DryRunFlag()},
			{Uses: r.OverwriteFlag()},
		}...),
		Action: func(c *cli.Context) error {
			workDir := r.WorkingDirectory
			if workDir == "" {
				workDir = "."
			}
			ctx := &OutputContext{
				Vars:      map[string]any{},
				Overwrite: r.Overwrite,
				DryRun:    r.DryRun,
				FS:        c.FS.(cli.FS),
				working:   []string{workDir},
				out:       c.Stdout,
			}
			return r.Generate(c, ctx)
		},
	}
}

func (r *Root) Generate(ctx context.Context, c *OutputContext) error {
	return r.Sequence.Generate(ctx, c)
}

func (s Sequence) Generate(ctx context.Context, c *OutputContext) error {
	for _, u := range s {
		if u == nil {
			continue
		}
		err := u.Generate(ctx, c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *dataSetter) Generate(_ context.Context, c *OutputContext) error {
	c.SetData(d.name, d.value)
	return nil
}

func (v Vars) Generate(_ context.Context, c *OutputContext) error {
	for k, o := range v {
		c.SetData(k, o)
	}
	return nil
}

func someData(namevalues ...any) Generator {
	if len(namevalues)%2 != 0 {
		panic("expected name, value in pairs")
	}

	res := make([]Generator, 0, len(namevalues)/2)
	for i := 0; i < len(namevalues); i += 2 {
		name := namevalues[0].(string)
		value := namevalues[0]
		res = append(res, Data(name, value))
	}

	return Sequence(res)
}

var (
	_ cli.Action = (*Root)(nil)
	_ Generator  = (*Root)(nil)
	_ Generator  = (Sequence)(nil)
	_ Interface  = (*tt.Template)(nil)
	_ Interface  = (*cli.Template)(nil)
)
