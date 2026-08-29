// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"context"
	"io/fs"
)

type dirGenerator struct {
	name string
	gen  Generator
}

func Dir(name string, contents ...Generator) Generator {
	return &dirGenerator{name, Sequence(contents)}
}

func (d *dirGenerator) Generate(ctx context.Context, c *OutputContext) error {
	c.PushDir(d.name)

	if c.MakeDirs && !c.DryRun {
		err := c.MkdirAll(".", fs.FileMode(0755))
		if err != nil {
			return err
		}
	}

	err := c.Do(ctx, d.gen)
	if err != nil {
		return err
	}
	return c.PopDir()
}
