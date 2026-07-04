// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template is used to create files from template file systems
package template

import (
	"context"
	"io"
	"maps"
	tt "text/template"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

//go:generate go tool counterfeiter -generate

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

func Data(namevalue ...any) Generator {
	if len(namevalue)%2 != 0 {
		panic("expected name, value in pairs")
	}
	res := make(Sequence, 0, len(namevalue)/2)
	for i := 0; i < len(namevalue); i += 2 {
		res = append(res, &dataSetter{namevalue[i].(string), namevalue[i+1]})
	}
	return res
}

func (r *Root) setOverwrite(v bool) error {
	r.Overwrite = v
	return nil
}

func (r *Root) setDryRun(v bool) error {
	r.DryRun = v
	return nil
}

// OverwriteFlag obtains a conventions-based flag for overwriting
func (r *Root) OverwriteFlag() cli.Prototype {
	return cli.Prototype{
		Name:     "overwrite",
		HelpText: "Overwrite files",

		Uses: bind.Call(r.setOverwrite),
	}
}

// DryRunFlag obtains a conventions-based flag for overwriting
func (r *Root) DryRunFlag() cli.Prototype {
	return cli.Prototype{
		Name:     "dry-run",
		HelpText: "Display what commands will be run without actually executing them",
		Uses:     bind.Call(r.setDryRun),
	}
}

// Execute implements the action interface
func (r *Root) Execute(ctx context.Context) error {
	return r.Pipeline().Execute(ctx)
}

// Pipeline converts the root into a pipeline
func (r *Root) Pipeline() cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Uses: cli.AddFlags([]*cli.Flag{
				{Uses: r.DryRunFlag()},
				{Uses: r.OverwriteFlag()},
			}...),
		},
		cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) error {
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
		})),
	)
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

func (v Vars) applyFSOption(g *fsGenerator) {
	if g.vars == nil {
		g.vars = make(Vars)
	}
	maps.Copy(g.vars, v)
}

var (
	_ cli.Action = (*Root)(nil)
	_ Generator  = (*Root)(nil)
	_ Generator  = (Sequence)(nil)
	_ Interface  = (*tt.Template)(nil)
	_ Interface  = (*cli.Template)(nil)
	_ Option     = (Vars)(nil)
)
