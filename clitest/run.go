// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clitest provides API meant to support testing Joe CLI applications.
package clitest

import (
	"bytes"
	"context"
	"io"

	cli "github.com/Carbonfrost/joe-cli"
)

// Command produces a command on the background context
func Command(app *cli.App, arg ...string) *Cmd {
	return CommandContext(context.Background(), app, arg...)
}

// CommandContext produces a command on the given context.
func CommandContext(ctx context.Context, app *cli.App, arg ...string) *Cmd {
	return &Cmd{
		ctx:  ctx,
		app:  app,
		args: arg,
	}
}

// Cmd represents an app being prepared or run.
type Cmd struct {
	ctx  context.Context
	app  *cli.App
	args []string
}

// CombinedOutput runs the app and returns its combined standard output and standard error.
func (c *Cmd) CombinedOutput() ([]byte, error) {
	var buffer bytes.Buffer

	defer c.useStdout(&buffer)()
	defer c.useStderr(&buffer)()

	err := c.app.RunContext(c.ctx, c.args)
	return buffer.Bytes(), err
}

func (c *Cmd) useStdout(stdout io.Writer) func() {
	original := c.app.Stdout
	c.app.Stdout = stdout
	return func() {
		c.app.Stdout = original
	}
}

func (c *Cmd) useStderr(stderr io.Writer) func() {
	original := c.app.Stderr
	c.app.Stderr = stderr
	return func() {
		c.app.Stderr = original
	}
}
