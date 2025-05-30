// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/Carbonfrost/joe-cli/internal/support"
)

// App provides the definition of an app, which is composed of commands, flags, and arguments.
type App struct {
	// Name provides the name of the app.  This value is inferred from the base name of the entry process if
	// it is explicitly not set.  It will be displayed on the help screen and other templates.
	Name string

	// Version provides the version of the app.  This value is typically used in templates, and otherwise
	// provides no special behavior.
	Version string

	// Description provides a description of the app.  This value is typically used in templates, and otherwise
	// provides no special behavior.  The type of Description should be string or
	// fmt.Stringer.  Refer to func Description for details.
	Description any

	// BuildDate provides the time when the app was built.  This value is typically used in templates, and otherwise
	// provides no special behavior.   The default value is inferred by checking the last modification time of
	// the entry process.
	BuildDate time.Time

	// Author provides the author of the app.  This value is typically used in templates, and otherwise
	// provides no special behavior.
	Author string

	// Copyright provides a copyright message for the app.  This value is typically used in templates, and otherwise
	// provides no special behavior.
	Copyright string

	// Comment provides a short comment.  This is
	// usually a few words to summarize the purpose of the app.
	Comment string

	// In corresponds to standard in
	Stdin io.Reader

	// In corresponds to standard out
	Stdout io.Writer

	// In corresponds to standard error
	Stderr io.Writer

	// Commands provides the list of commands in the app
	Commands []*Command

	// Flags supplies global flags for use with the app
	Flags []*Flag

	// Args will be bound for non-command arguments
	Args []*Arg

	// Action specifies the action to run for the app, assuming no other more specific command
	// has been selected.  Refer to cli.Action about the correct function signature to use.
	Action any

	// Before executes before the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	Before any

	// After executes after the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	After any

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	// Actions within the Uses and Before pipelines can modify the app Commands and Flags lists.  Any
	// commands or flags added to the list will be initialized
	Uses any

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]any

	// Options sets common options for use with the app
	Options Option

	// FS specifies the file system that is used by default for Files.
	// If the FS implements func OpenContext(context.Context, string)(fs.File, error), note that
	// this will be called instead of Open in places where the Context is available.
	// For os.File this means that if the context has a Deadline, SetReadDeadline
	// and/or SetWriteDeadline will be set.  Clients can implement similar functions in their
	// own fs.File implementations provided from an FS implementation.
	FS fs.FS

	// HelpText describes the help text displayed for the app
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the app.  If left blank, a succinct synopsis
	// is generated that lists each visible flag and arg
	UsageText string

	// License sets the text of the app license.
	License string

	rootCommandCreator func() *Command
	rootCommand        *Command
}

var (
	// ExitHandler defines how to handle exiting the process.  This function
	// takes the context, error message, and exit status.  By default, text is
	// written to stderr and the exit status is returned via os.Exit
	ExitHandler func(*Context, string, int)

	currentApp atomic.Value

	osExit = os.Exit

	defaultTemplates = map[string]func() string{
		"Help": func() string {
			return HelpTemplate
		},
		"Version": func() string {
			return VersionTemplate
		},
		"License": func() string {
			return LicenseTemplate
		},
		"Expressions": func() string {
			return expressionTemplate
		},
	}
)

// NewApp creates an app initialized from the specified command.  The App struct can be used directly
// to create and run an App.  It has the same set of exported fields; however, you can also initialize
// an app from a command directly using this function.  This benefits from having a
// fully consistent model with Command and fewer hidden semantics.
func NewApp(cmd *Command) *App {
	return &App{
		Name: cmd.Name,
		rootCommandCreator: func() *Command {
			return cmd
		},
	}
}

// CurrentApp contains the current app.  If the app was run with Run or RunContext, this will contain
// the app.
func CurrentApp() *App {
	a, _ := currentApp.Load().(*App)
	return a
}

// SetCurrentApp sets the current app directly so that tools, extensions, and plug-ins
// can discover which app to analyze.
func SetCurrentApp(a *App) {
	currentApp.Store(a)
}

// Run the application and exit using the exit handler.  This function exits using the
// ExitHandler if an error occurs.  If you want to process the error yourself, use RunContext.
func (a *App) Run(args []string) {
	c, err := a.runContextCore(context.Background(), args)
	exit(c, err)
}

// RunContext runs the application with the specified context and returns any error
// that occurred.
func (a *App) RunContext(c context.Context, args []string) error {
	_, err := a.runContextCore(c, args)
	return err
}

// Use appends actions to Uses pipeline
func (a *App) Use(action Action) *App {
	a.Uses = Pipeline(a.Uses).Append(action)
	return a
}

// Command gets the command by name.  If the name is the empty string,
// this refers to the command which backs the app once it has been initialized.
func (a *App) Command(name string) (*Command, bool) {
	if name == "" {
		return a.rootCommand, true
	}
	if a.rootCommand != nil {
		return a.rootCommand.Command(name)
	}
	c, _, ok := findCommandByName(a.Commands, name)
	return c, ok
}

// Flag gets the flag by name.
func (a *App) Flag(name string) (*Flag, bool) {
	if a.rootCommand != nil {
		return a.rootCommand.Flag(name)
	}
	f, _, ok := findFlagByName(a.Flags, name)
	return f, ok
}

// Arg gets the argument by name, which can be either string, int, or the actual
// Arg.
func (a *App) Arg(name any) (*Arg, bool) {
	if a.rootCommand != nil {
		return a.rootCommand.Arg(name)
	}
	res, _, ok := findArgByName(a.Args, name)
	return res, ok
}

func (a *App) createRoot() *Command {
	cmd := func() *Command {
		if a.rootCommandCreator == nil {
			return &Command{
				Name:        a.Name,
				Flags:       a.Flags,
				Args:        a.Args,
				Subcommands: a.Commands,
				Action:      a.Action,
				Description: a.Description,
				Comment:     a.Comment,
				Data:        a.Data,
				After:       a.After,
				Uses:        a.Uses,
				Before:      a.Before,
				Options:     a.Options,
			}
		}
		return a.rootCommandCreator()
	}()

	a.rootCommand = cmd
	cmd.fromApp = a
	return cmd
}

// SetData sets the specified metadata on the app
func (a *App) SetData(name string, v any) {
	a.Data = setData(a.Data, name, v)
}

func (a *App) runContextCore(c context.Context, args []string) (context.Context, error) {
	ctx, err := a.Initialize(c)
	if err != nil {
		return ctx, err
	}
	return ctx, FromContext(ctx).Execute(args)
}

// Initialize sets up the app with the given context and arguments
func (a *App) Initialize(c context.Context) (context.Context, error) {
	defer provideCurrentApp(a)()
	ctx := rootContext(c, a)
	err := ctx.initialize()
	return ctx, err
}

func provideCurrentApp(a *App) func() {
	SetCurrentApp(a)
	return func() {
		SetCurrentApp(nil)
	}
}

func newRootCommandData() *rootCommandData {
	root := template.New("_Root")
	tf := withExecute(template.FuncMap{}, root)
	return &rootCommandData{
		templates:     root,
		templateFuncs: tf,
	}
}

func (a *rootCommandData) ensureTemplates() *template.Template {
	// Internally, funcs are copied, so ensure the latest
	return a.templates.Funcs(a.templateFuncs)
}

func (a *rootCommandData) ensureTemplateFuncs() template.FuncMap {
	return a.templateFuncs
}

func exit(c context.Context, err error) {
	if err == nil {
		return
	}

	handler := ExitHandler
	code := 1
	if handler == nil {
		handler = defaultExitHandler
	}
	if coder, ok := err.(ExitCoder); ok {
		code = coder.ExitCode()
	}
	handler(FromContext(c), err.Error(), code)
}

func defaultExitHandler(c *Context, message string, status int) {
	stderr := c.Stderr
	if message != "" && status != 0 {
		fmt.Fprintln(stderr, message)
	}
	osExit(status)
}

func buildDate() time.Time {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

func setupDefaultData(c *Context) error {
	a := c.App()
	if a.Name == "" {
		a.Name = filepath.Base(os.Args[0])
	}
	if a.Version == "" {
		a.Version = "0.0.0"
	}
	if a.BuildDate.IsZero() {
		a.BuildDate = buildDate()
	}
	return nil
}

func fixupCommandInternals(c *Context) error {
	// For the command created at the root, we may need to sync again
	// if it was set on App
	a := c.App()
	cmd := c.Command()
	if cmd.Name == "" {
		cmd.Name = a.Name
	}
	return nil
}

func setupDefaultIO(c *Context) error {
	a := c.App()
	if a.Stdin == nil {
		a.Stdin = os.Stdin
	}
	if a.Stdout == nil {
		a.Stdout = os.Stdout
	}
	if a.Stderr == nil {
		a.Stderr = os.Stderr
	}

	c.Stdin = a.Stdin
	c.Stdout = adaptWriter(a.Stdout)
	c.Stderr = adaptWriter(a.Stderr)
	c.FS = a.FS

	// ensure some actual FS is set up if a.FS is nil
	c.FS = c.actualFS()

	return nil
}

func adaptWriter(w io.Writer) Writer {
	if a, ok := w.(Writer); ok {
		return a
	}
	return NewWriter(w)
}

func setupDefaultTemplateFuncs(c *Context) error {
	width := support.GuessWidth()
	wrap := func(indent int, s string) string {
		buf := bytes.NewBuffer(nil)
		indentText := strings.Repeat(" ", indent)
		Wrap(buf, s, indentText, width)
		return buf.String()
	}

	funcMap := template.FuncMap{
		"Wrap": wrap,
		"ExtraSpaceBeforeFlag": func(s string) string {
			if strings.HasPrefix(controlCodes.ReplaceAllString(s, ""), "--") {
				return "    " + s
			}
			return s
		},
		"HangingIndent": func(indent int, s string) string {
			return strings.TrimSpace(wrap(indent, s))
		},
	}

	a := c.root()
	funcs := a.ensureTemplateFuncs()
	for k, v := range builtinFuncs {
		if _, ok := funcs[k]; !ok {
			funcs[k] = v
		}
	}
	for k, v := range funcMap {
		if _, ok := funcs[k]; !ok {
			funcs[k] = v
		}
	}
	return nil
}

func setupDefaultTemplates(c *Context) error {
	for k, v := range defaultTemplates {
		err := c.RegisterTemplate(k, v())
		if err != nil {
			return err
		}
	}
	return nil
}

func optionalCommand(name string, cmd func() *Command) ActionFunc {
	return func(c *Context) error {
		app := c.Command()
		if len(app.Subcommands) > 0 {
			if _, ok := app.Command(name); !ok {
				app.Subcommands = append(app.Subcommands, cmd())
			}
		}
		return nil
	}
}

func optionalFlag(name string, f func() *Flag) ActionFunc {
	return func(c *Context) error {
		// Don't create optional flags if setup has been tainted
		// or if a flag with the same name exists
		if c.SkipImplicitSetup() {
			return nil
		}

		app := c.Command()
		if _, ok := app.Flag(name); !ok {
			return c.AddFlag(f())
		}
		return nil
	}
}
