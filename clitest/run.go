// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clitest provides API meant to support testing Joe CLI applications.
package clitest

import (
	"bytes"
	"cmp"
	"context"
	"io"

	cli "github.com/Carbonfrost/joe-cli"
)

// Command produces a command on the background context
func Command(app *cli.App, name string, arg ...string) *Cmd {
	return CommandContext(context.Background(), app, name, arg...)
}

// CommandContext produces a command on the given context.
func CommandContext(ctx context.Context, app *cli.App, name string, arg ...string) *Cmd {
	return &Cmd{
		ctx:  ctx,
		app:  app,
		args: append([]string{name}, arg...),
	}
}

// Cmd represents an app being prepared or run.
type Cmd struct {
	ctx  context.Context
	app  *cli.App
	args []string

	// Stdin specifies the app's standard input.
	//
	// If Stdin is nil, it reads from the null device.
	Stdin io.Reader

	// Stdout and Stderr specify the app's standard output and error.
	//
	// If either is nil, Run connects the corresponding file descriptor
	// to the null device.
	Stdout io.Writer
	Stderr io.Writer
}

// CombinedOutput runs the app and returns its combined standard output and standard error.
func (c *Cmd) CombinedOutput() ([]byte, error) {
	var buffer bytes.Buffer

	defer c.useStdout(&buffer)()
	defer c.useStderr(&buffer)()
	defer c.useStdin(cmp.Or(c.Stdin, emptyReader()))

	err := c.app.RunContext(c.ctx, c.args...)
	return buffer.Bytes(), err
}

// Args gets the arguments to the command, including the app name
func (c *Cmd) Args() []string {
	return c.args
}

// String gets the representation of the command arguments
func (c *Cmd) String() string {
	return cli.Join(c.args)
}

// Run invokes the app
func (c *Cmd) Run() error {
	defer c.useIO()
	return c.app.RunContext(c.ctx, c.args...)
}

func (c *Cmd) useIO() func() {
	cleanup := []func(){
		c.useStdout(cmp.Or(c.Stdout, io.Discard)),
		c.useStderr(cmp.Or(c.Stderr, io.Discard)),
		c.useStdin(cmp.Or(c.Stdin, emptyReader())),
	}
	return func() {
		for _, s := range cleanup {
			s()
		}
	}
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

func (c *Cmd) useStdin(stdin io.Reader) func() {
	original := c.app.Stdin
	c.app.Stdin = stdin
	return func() {
		c.app.Stdin = original
	}
}

func emptyReader() io.Reader {
	return bytes.NewBuffer(nil)
}
