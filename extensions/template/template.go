// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template is used to create files from template file systems
package template

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
	"github.com/Carbonfrost/joe-cli/internal/shell"
)

//go:generate go tool counterfeiter -generate

//counterfeiter:generate . Generator

// Root is the root of a template, used to compose a sequence and
// configuration.
// You typically create an instance using the [New] function,
// which in addition to specifying the generator you want, sets
// up the default action for the template when it runs in the
// pipeline, which is to register the flags and run Generate
// at the action timing.
type Root struct {
	Generator Generator
	Overwrite bool
	DryRun    bool
	Path      string

	// MakeDirs controls whether to implicitly create directories.
	// It is set to true by default by [New]. When set to false, the
	// [Dir] generator becomes a container only and does not create directories.
	// To ensure a directory exists, add a FileMode, usually FileMode(0755),
	// as the first generator argument.
	MakeDirs bool
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

type dataSetter struct {
	name  string
	value any
}

// New creates a new template using the given sequence of generators.
// Directories will automatically be created by default because MakeDirs will
// also be set to true.
func New(items ...Generator) *Root {
	return &Root{
		Generator: Sequence(items),
		MakeDirs:  true,
	}
}

func Data(namevalue ...any) Generator {
	if len(namevalue)%2 != 0 {
		panic("expected name, value in pairs")
	}
	res := make(Sequence, 0, len(namevalue)/2)
	for namevalue := range slices.Chunk(namevalue, 2) {
		res = append(res, &dataSetter{namevalue[0].(string), namevalue[1]})
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
				{Uses: cli.Accessory0("", r.DryRunFlag())},
				{Uses: cli.Accessory0("", r.OverwriteFlag())},
			}...),
		},
		cli.At(cli.ActionTiming, cli.ActionFunc(func(c *cli.Context) error {
			workDir := r.Path
			if workDir == "" {
				workDir = "."
			}
			fsys, out := findOutput(c)
			return r.Generate(c, r.outputContext(fsys, out, workDir))
		})),
	)
}

func (r *Root) outputContext(fsys cli.FS, out cli.Writer, workDir string) *OutputContext {
	return &OutputContext{
		Vars:      map[string]any{},
		Overwrite: r.Overwrite,
		DryRun:    r.DryRun,
		MakeDirs:  r.MakeDirs,
		FS:        fsys,
		working:   []string{workDir},
		out:       out,
	}
}
func findOutput(ctx context.Context) (fsys cli.FS, out cli.Writer) {
	c, ok := cli.TryFromContext(ctx)
	if !ok {
		return cli.NewSysFS(cli.DirFS("."), os.Stdin, os.Stdout), cli.NewWriter(os.Stdout)
	}
	return c.FS.(cli.FS), c.Stdout
}

func (r *Root) Generate(ctx context.Context, c *OutputContext) error {
	// Provide a hint about the working directory if it is different from the one
	// that started the process
	wd, err := filepath.Abs(c.WorkDir())
	if err == nil && shell.StartDir() != wd {
		c.trace("working dir", wd)
	}

	return r.Generator.Generate(ctx, c)
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
	_ Option     = (Vars)(nil)
)
