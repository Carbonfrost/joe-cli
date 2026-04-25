// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/bind"
)

// Workspace provides the root directory of a project-specific workspace.
// The workspace provides two capabilities:
//  1. The workspace finder locates the root directory of a project. By default,
//     the workspace is found by searching up the directory hierarchy until it
//     finds a directory which contains a configuration sentinel, typically a
//     directory which the same name as the app with a leading period. For example,
//     this is what Git does when it finds the .git directory.
//  2. The workspace can load files. These files could be application files or
//     configuration files.
type Workspace struct {
	// Action provides the action to provide when the workspace is
	// added to a pipeline.
	cli.Action

	dir         string
	configDir   string
	f           fs.FS
	walkDirFunc fs.WalkDirFunc
	finder      WorkspaceFinder
	loader      func(fs.FS, string, fs.DirEntry) (any, error)
}

// WorkspaceOption provides an option for setting up the Workspace. This is also
// a cli.Action, which means it can be used within pipelines. Typically, when used
// within the Action pipeline, it sets the option on the workspace. When used within
// the Uses pipeline, it may also configure a flag or other behavior.
type WorkspaceOption interface {
	cli.Action
	apply(*Workspace)
}

// WorkspaceFinder locates the workspace
type WorkspaceFinder interface {

	// FindWorkspacePath locates the workspace. Param cwd represents the
	// the current working directory
	FindWorkspacePath(c context.Context, f fs.FS, cwd string) (string, error)
}

type defaultWorkspaceFinder int

type workspaceOption func(*Workspace) error

// SkipFile is returned by the walker function to indicate a file should not
// be returned during the workspace scan
var SkipFile = errors.New("skip this file")

// DefaultWorkspaceFinder provides the default logic for finding a workspace
var DefaultWorkspaceFinder WorkspaceFinder = new(defaultWorkspaceFinder)

var errWorkspaceNotFound = fmt.Errorf("workspace not found")

// NewWorkspace creates a workspace with the default action, which registers the
// flags and adds the workspace to the context.
func NewWorkspace(opts ...WorkspaceOption) *Workspace {
	w := withDefaultAction(new(Workspace))
	w.Apply(opts...)
	return w
}

// Apply applies options to the workspace
func (w *Workspace) Apply(opts ...WorkspaceOption) {
	for _, o := range opts {
		o.apply(w)
	}
}

// Pipeline obtains the pipeline for initializing the workspace
func (w *Workspace) Pipeline() cli.Action {
	return w.Action
}

func withDefaultAction(w *Workspace) *Workspace {
	w.Action = cli.Pipeline(
		ContextValue(w),
		SetupWorkspace(),
		cli.AddFlags([]*cli.Flag{
			{Uses: SetWorkingDir()},
			{Uses: SetConfigDir()},
		}...),
	)
	return w
}

// WorkspaceFromContext gets the Workspace from the context otherwise panics
func WorkspaceFromContext(ctx context.Context) *Workspace {
	return fromContext[*Workspace](ctx)
}

func tryWorkspaceFromContext(ctx context.Context) (*Workspace, error) {
	return tryFromContext[*Workspace](ctx)
}

// ContextValue provides an action that sets the given value into the context.
// The only supported type is *Workspace.
func ContextValue(v contextValue) cli.Action {
	return cli.WithContextValue(keyFor(v), v)
}

// SetupWorkspace provides the action that sets up the workspace, which locks in
// the workspace directories.
func SetupWorkspace() cli.Action {
	return cli.Before(cli.ActionOf(func(c context.Context) error {
		if ws, err := tryWorkspaceFromContext(c); err == nil {
			ws.completeSetup(c)
		}
		return nil
	}))
}

// SetWorkingDir sets the current directory. It is typical within apps that
// work with workspaces to allow the user to set the current directory.  \
func SetWorkingDir(diropt ...string) cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "chdir",
			HelpText: "Run the command as if started in a working DIRECTORY",
			Value:    new(string),
			Options:  cli.MustExist,
		},
		cli.At(cli.ValidatorTiming,
			cli.IfMatch(
				cli.Seen,
				bind.BeforeCall(os.Chdir, bind.Exact(diropt...)),
			),
		),
	)
}

// WithConfigDir sets the workspace config directory. This can be used as an option to
// NewWorkspace or Workspace.Apply, but when it is, the diropt must be specified.
// It can also be used as an Action.
func WithConfigDir(dir string) WorkspaceOption {
	return workspaceOption(func(w *Workspace) error {
		w.configDir = dir
		return nil
	})
}

// SetConfigDir sets the workspace config directory as an action.
// When it is used within the Uses pipeline,
// it initializes reasonable defaults for a flag and will set the directory within
// the workspace in the Before timing. The flag will either use the
// directory specified by diropt or its own value if diropt is not specified.
func SetConfigDir(diropt ...string) cli.Action {
	return cli.Pipeline(
		cli.Prototype{
			Name:     "config-dir",
			HelpText: "Set the path to the configuration DIRECTORY",
		},
		func(c *cli.Context) {
			c.SetName(appName(c) + "-dir")
		},
		cli.At(cli.ValidatorTiming,
			cli.IfMatch(
				cli.Seen,
				bind.Before(WithConfigDir, bind.Exact(diropt...)),
			),
		),
	)
}

func (w workspaceOption) apply(ws *Workspace) {
	w(ws)
}

func (w workspaceOption) Execute(c context.Context) error {
	return w(WorkspaceFromContext(c))
}

// Dir gets the workspace directory, which is the root of all content in
// the workspace.  The directory is set to the current working directory at the time
// the app starts, sometime in the Before pipeline.
func (w *Workspace) Dir() string {
	return w.dir
}

func (w *Workspace) completeSetup(c context.Context) {
	if w.dir == "" {
		cwd, _ := os.Getwd()

		finder := cmp.Or(w.finder, DefaultWorkspaceFinder)

		cwd, _ = finder.FindWorkspacePath(c, actualFS(c), cwd)
		w.dir = cwd

		// TODO Seems like this should be a sub-FS on the actualFS rather than
		// always in the file system
		w.f = cmp.Or(w.f, os.DirFS(cwd))
		w.configDir = cmp.Or(w.configDir, filepath.Join(cwd, "."+appName(c)))
	}

	if w.walkDirFunc == nil {
		w.walkDirFunc = func(string, fs.DirEntry, error) error {
			return nil
		}
	}
}

// ConfigDir gets the directory where configuration is stored. Typically,
// this has the same name as the app with a leading dot, e.g. $WORKSPACE_DIR/.app.
func (w *Workspace) ConfigDir() string {
	return w.configDir
}

// FS gets the file system that represents the workspace.
func (w *Workspace) FS() fs.FS {
	return w.f
}

// WorkspaceFileLoaderFunc defines the function for loading files in a workspace.
type WorkspaceFileLoaderFunc func(root fs.FS, name string, d fs.DirEntry) (any, error)

// WithFileLoader is an option that determines how to read files in a workspace.
func WithFileLoader(loader WorkspaceFileLoaderFunc) WorkspaceOption {
	return workspaceOption(func(w *Workspace) error {
		w.loader = loader
		return nil
	})

}

// WithFinder is an option that determines how to finder the workspace
func WithFinder(finder WorkspaceFinder) WorkspaceOption {
	return workspaceOption(func(w *Workspace) error {
		w.finder = finder
		return nil
	})

}

// WithFS specifies the file system to use for the workspace. By default, the workspace
// with use [os.DirFS] corresponding to the workspace directory.
func WithFS(f fs.FS) WorkspaceOption {
	return workspaceOption(func(w *Workspace) error {
		w.f = f
		return nil
	})
}

// WithWalkDirFunc specifies the function that walks the directory for files.
// The function is used in a call to [io/fs.WalkDir], where it can return
// [io/fs.SkipDir] and [io/fs.SkipAll], with the same behavior. It can also
// return SkipFile introduced by this package to indicate that a file is skipped
// in the result of the [Workspace.Files] and [Workspace.LoadFiles] methods.
func WithWalkDirFunc(fn fs.WalkDirFunc) WorkspaceOption {
	return workspaceOption(func(w *Workspace) error {
		w.walkDirFunc = fn
		return nil
	})
}

// Files enumerates all files in the workspace which match the filters.
// The result contains the name of the file and a [io/fs.DirEntry].
func (w *Workspace) Files(diropt ...string) iter.Seq2[string, fs.DirEntry] {
	return func(yield func(string, fs.DirEntry) bool) {
		_ = w.walkDir(func(path string, d fs.DirEntry, _ error) error {
			if !yield(path, d) {
				return fs.SkipAll
			}

			return nil
		}, diropt...)
	}
}

// LoadFiles loads all files in the workspace which match the filters.
// This method is good for a single pass, not dynamic workspaces where files
// are expected to change while the app is running. The type of the items
// returned from this depend upon what was set as the loader. When no loader
// is set, the result will be fs.File.
func (w *Workspace) LoadFiles(diropt ...string) iter.Seq[any] {
	return func(yield func(any) bool) {
		_ = w.walkDir(func(path string, d fs.DirEntry, _ error) error {
			loaded, err := w.loader(w.FS(), path, d)
			if err != nil {
				return err
			}

			if !yield(loaded) {
				return fs.SkipAll
			}

			return nil
		}, diropt...)
	}
}

func (w *Workspace) walkDir(fn fs.WalkDirFunc, diropt ...string) error {
	rootFS := w.FS()
	var err error

	if len(diropt) > 0 {
		rootFS, err = fs.Sub(w.FS(), filepath.Join(diropt...))
		if err != nil {
			return err
		}
	}

	return fs.WalkDir(rootFS, ".", func(name string, d fs.DirEntry, err error) error {
		err = w.walkDirFunc(name, d, err)
		if SkipFile == err {
			return nil
		}

		if err != nil {
			return err
		}

		if d == nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		return fn(name, d, err)
	})
}

func (*Workspace) contextValueSigil() {}

func (defaultWorkspaceFinder) FindWorkspacePath(c context.Context, f fs.FS, cwd string) (string, error) {
	var sentinel = "." + appName(c)

	for _, ws := range ancestorPaths(cwd) {
		f, err := fs.Stat(f, filepath.Join(ws, sentinel))
		if err == nil && f.IsDir() {
			return ws, nil
		}
	}
	return "", errWorkspaceNotFound
}

func actualFS(ctx context.Context) fs.FS {
	c, ok := cli.TryFromContext(ctx)
	if !ok || c == nil || c.FS == nil {
		return os.DirFS("")
	}
	return c.FS
}

func appName(ctx context.Context) string {
	c, ok := cli.TryFromContext(ctx)
	if !ok || c == nil {
		return "config"
	}
	return strings.ToLower(c.App().Name)
}

func ancestorPaths(c string) []string {
	res := []string{}
	for ; ; c = filepath.Dir(c) {
		// Stop on no parent
		if len(res) > 0 && res[len(res)-1] == c {
			break
		}
		res = append(res, c)
	}

	return res
}
