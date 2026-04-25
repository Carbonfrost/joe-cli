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
	c.state = &fakeContextState{
		tim: InitialTiming,
	}
}

func SetBeforeTiming(c *Context) {
	c.state = &fakeContextState{
		tim: BeforeTiming,
	}
}

func SetAfterTiming(c *Context) {
	c.state = &fakeContextState{
		tim: AfterTiming,
	}
}

func SetActionTiming(c *Context) {
	c.state = &fakeContextState{
		tim: ActionTiming,
	}
}

type fakeContextState struct {
	tim Timing
	ref context.Context
}

func (s *fakeContextState) getInternal() internalContext { return nil }
func (s *fakeContextState) getTarget() target            { return nil }
func (s *fakeContextState) getParent() *Context          { return nil }
func (s *fakeContextState) getTiming() Timing            { return s.tim }
func (s *fakeContextState) getOrigin() *Context          { return nil }
func (s *fakeContextState) close()                       {}
func (s *fakeContextState) getRef() context.Context      { return s.ref }
func (s *fakeContextState) updateRef(c context.Context)  { s.ref = c }

func WithStubState(c *Context) *Context {
	c.state = new(fakeContextState)
	return c
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
			f.Uses = Pipeline(f.Uses, useThunk)
			return &App{
				Flags: []*Flag{
					f,
				},
			}
		case *Arg:
			f.Uses = Pipeline(f.Uses, useThunk)
			return &App{
				Args: []*Arg{
					f,
				},
			}
		case *Command:
			f.Uses = Pipeline(f.Uses, useThunk)
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

func PipelineContents(v Action) []Action {
	return v.(pipeline).actions
}
