// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package template

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// GoGet assumes that the current template is a Go module and adds the given go module via
// go get.  An error results if the project is not a go module.
func GoGet(pkgs ...string) Generator {
	return &goGetter{pkgs}
}

type goGetter struct {
	pkgs []string
}

func (d *goGetter) Generate(ctx context.Context, c *OutputContext) error {
	if c.DryRun {
		return d.dryRun(ctx, c)
	}
	return d.realGenerate(ctx, c)
}

func (d *goGetter) realGenerate(ctx context.Context, c *OutputContext) error {
	originalMod, originalSum := d.files()

	err := execGoGet(d.pkgs)
	if err != nil {
		c.error("go.mod")
		return err
	}

	c.reportChange(originalMod, "go.mod", false)
	c.reportChange(originalSum, "go.sum", false)
	return nil
}

func (d *goGetter) dryRun(ctx context.Context, c *OutputContext) error {
	originalMod, _ := d.files()

	for _, pkg := range d.pkgs {
		if !strings.Contains(string(originalMod), pkg) {
			c.overwrite("go.mod")
			c.overwrite("go.sum")
			return nil
		}
	}
	return nil
}

func (d *goGetter) files() (originalMod, originalSum []byte) {
	originalMod, _ = os.ReadFile("go.mod")
	originalSum, _ = os.ReadFile("go.sum")
	return
}

func execGoGet(modules []string) error {
	args := append([]string{"get"}, modules...)
	cmd := exec.Command("go", args...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
