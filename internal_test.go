// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cli // Intentional

import (
	"context"
	"io/fs"
	"os"
)

// Expose some members for testing

const MaxOption = maxOption

// Provides the logic of os.Exit for tests
func SetOSExit(fn func(int)) {
	osExit = fn
}

func IsVisible(t any) bool {
	return !t.(target).internalFlags().hidden()
}

func IsInitialized(t any) bool {
	return t.(target).internalFlags().initialized()
}

func IsDestinationImplicitlyCreated(t any) bool {
	return t.(target).internalFlags().destinationImplicitlyCreated()
}

func SetInitialTiming(c *Context) {
	c.timing = InitialTiming
}

func SetBeforeTiming(c *Context) {
	c.timing = BeforeTiming
}

func SetAfterTiming(c *Context) {
	c.timing = AfterTiming
}

func SetActionTiming(c *Context) {
	c.timing = ActionTiming
}

func (a *Arg) ActualArgCounter() ArgCounter {
	return ArgCount(a)
}

func DefaultFlagCounter() ArgCounter {
	return &defaultCounter{requireSeen: true}
}

func Initialized(t target) *Context {
	var captured *Context
	useThunk := ActionFunc(func(c *Context) error {
		captured = c
		return nil
	})

	app := func() *App {
		switch f := t.(type) {
		case *Flag:
			f.Use(useThunk)
			return &App{
				Flags: []*Flag{
					f,
				},
			}
		case *Arg:
			f.Use(useThunk)
			return &App{
				Args: []*Arg{
					f,
				},
			}
		case *Command:
			f.Use(useThunk)
			return &App{
				Commands: []*Command{
					f,
				},
			}
		}
		panic("unreachable!")
	}()

	app.Initialize(context.Background())
	return captured
}

// DefaultFS is the FS that is expected to be created when no
// other is set up
func DefaultFS() fs.FS {
	return newDefaultFS(os.Stdin, NewWriter(os.Stdout))
}
