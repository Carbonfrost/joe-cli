// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package exec allows invoking other commands, including triggering the manual, opening a document,
// or opening a Web page in the default Web browser.  It also provides a representation of the
// flag value syntax used by the -exec expression in Unix-like find designed to pass the name of
// a command and its arguments.
package exec

import (
	"errors"
	eexec "os/exec"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

// Open a file or URL, optionally in the given app.
// Either one or two arguments is specified, the name/URL of the file to open
// and the name of the app to use. If the argument looks like a URL, then it
// triggers the use of the protocol handler associated with the protocol
// if no app is specified.
func Open(file string, appopt ...string) error {
	cmd, err := func() (*eexec.Cmd, error) {
		switch len(appopt) {
		case 0:
			return openUsingApp(file, "")
		case 1:
			return openUsingApp(file, appopt[0])
		default:
			panic("invalid arguments: expected file and optional app")
		}
	}()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	return appearsSuccessful(cmd, 3*time.Second)
}

// appearsSuccessful reports whether the command appears to have run successfully.
// If the command runs longer than the timeout, it's deemed successful.
// If the command runs within the timeout, it's deemed successful if it exited cleanly.
func appearsSuccessful(cmd *eexec.Cmd, timeout time.Duration) error {
	errc := make(chan error, 1)
	go func() {
		errc <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		return nil
	case err := <-errc:
		return err
	}
}

// HaveLookPath returns a ContextFilter that checks if the given pathname exists on the PATH.
func HaveLookPath(pathname string) cli.ContextFilter {
	return cli.ContextFilterFunc(func(*cli.Context) bool {
		_, err := eexec.LookPath(pathname)
		return err == nil
	})
}

// ExternalCommand provides a CommandNotFoundHandler that dispatches to external
// executables found on the PATH, with the name "<prefix>-<name>", where name is the missing
// sub-command.
//
// By default, the prefix is the name of the app.  An alternate prefix can be supplied
// as the optional argument.
//
// The external process inherits the context's Stdin, Stdout, and Stderr and is run
// with the context so that cancellation propagates to it.  All arguments that follow
// the sub-command name are passed through verbatim.
func ExternalCommand(prefix ...string) cli.CommandNotFoundHandler {
	var override string
	switch len(prefix) {
	case 0:
	case 1:
		override = prefix[0]
	default:
		panic("expected zero or one arg")
	}

	return cli.HandleCommandNotFound(func(c *cli.Context, err error) (*cli.Command, error) {
		args := c.Args()
		if len(args) == 0 {
			return nil, err
		}
		sub := args[0]

		name := override
		if name == "" {
			name = c.App().Name
		}

		path, lookErr := eexec.LookPath(name + "-" + sub)
		if lookErr != nil {
			return nil, err
		}

		return &cli.Command{
			Name:    sub,
			Options: cli.SkipFlagParsing,
			Args: []*cli.Arg{
				{
					Name:  "args",
					NArg:  cli.TakeRemaining,
					Value: cli.List(),
				},
			},
			Action: cli.ActionFunc(func(c *cli.Context) error {
				return runExternal(c, path, c.List("args"))
			}),

			// TODO May have to propagate signals/child process group so ^C works as expected
		}, nil
	})
}

func runExternal(c *cli.Context, path string, args []string) error {
	cmd := eexec.CommandContext(c, path, args...)
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	if err := cmd.Run(); err != nil {

		// Wrap the exit error so that a redundant message is not printed.
		if exitErr, ok := errors.AsType[*eexec.ExitError](err); ok {
			return exitStatus(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

type exitStatus int

// Error prints no message so that the built-in exit handler does not print anything
func (exitStatus) Error() string   { return "" }
func (e exitStatus) ExitCode() int { return int(e) }

var _ cli.ExitCoder = exitStatus(0)
