package cli

import (
	"bytes"
	"context"
	"fmt"
	"go/doc"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// App provides the definition of an app, which is composed of commands, flags, and arguments.
type App struct {
	hooksSupport
	customizableSupport

	// Name provides the name of the app.  This value is inferred from the base name of the entry process if
	// it is explicitly not set.  It will be displayed on the help screen and other templates.
	Name string

	// Version provides the version of the app.  This value is typically used in templates, and otherwise
	// provides no special behavior.
	Version string

	// Description provides a description of the app.  This value is typically used in templates, and otherwise
	// provides no special behavior.
	Description string

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

	// In corresponds to standard in
	Stdin io.Reader

	// In corresponds to standard out
	Stdout io.Writer

	// In corresponds to standard error
	Stderr io.Writer

	// Comands provides the list of commands in the app
	Commands []*Command

	// Flags supplies global flags for use with the app
	Flags []*Flag

	// Args will be bound for non-command arguments
	Args []*Arg

	// Action specifies the action to run for the app, assuming no other more specific command
	// has been selected.  Refer to cli.Action about the correct function signature to use.
	Action interface{}

	// Before executes before the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	Before interface{}

	// After executes after the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	After interface{}

	// Uses provides an action handler that is always executed during the initialization phase
	// of the app.  Typically, hooks and other configuration actions are added to this handler.
	// Actions within the Uses and Before pipelines can modify the app Commands and Flags lists.  Any
	// commands or flags added to the list will be initialized
	Uses interface{}

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	// Options sets common options for use with the app
	Options Option

	// FS specifies the file system that is used by default for Files.
	FS fs.FS

	// HelpText describes the help text displayed for the app
	HelpText string

	// ManualText provides the text shown in the manual.  The default templates don't use this value
	ManualText string

	// UsageText provides the usage for the app.  If left blank, a succint synopsis
	// is generated that lists each visible flag and arg
	UsageText string

	rootCommandCreator func() *Command
	rootCommand        *Command
	flags              internalFlags
	templateFuncs      map[string]interface{}
	templates          map[string]string
}

type appContext struct {
	*commandContext
	app *App
}

var (
	// ExitHandler defines how to handle exiting the process.  This function
	// takes the context, error message, and exit status.  By default, text is
	// written to stderr and the exit status is returned via os.Exit
	ExitHandler func(*Context, string, int)

	defaultTemplates = map[string]func() string{
		"help": func() string {
			return HelpTemplate
		},
		"version": func() string {
			return VersionTemplate
		},
		"expressions": func() string {
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

// Command gets the command by name
func (a *App) Command(name string) (*Command, bool) {
	return findCommandByName(a.Commands, name)
}

// Flag gets the flag by name
func (a *App) Flag(name string) (*Flag, bool) {
	return findFlagByName(a.Flags, name)
}

// Arg gets the argument by name
func (a *App) Arg(name string) (*Arg, bool) {
	return findArgByName(a.Args, name)
}

func (a *App) createRoot() *Command {
	return a._createRootCore(false)
}

func (a *App) _createRootCore(force bool) *Command {
	if a.rootCommand == nil || force {
		var (
			flags   internalFlags
			hooks   hooksSupport
			customs customizableSupport
			data    map[string]interface{}
		)
		if a.rootCommand != nil {
			flags = a.rootCommand.flags
			hooks = a.rootCommand.hooksSupport
			customs = a.rootCommand.customizableSupport
			data = a.rootCommand.Data
		}
		flags |= a.flags
		for k, v := range a.Data {
			data[k] = v
		}

		a.rootCommand = &Command{
			Name:                a.Name,
			Flags:               a.Flags,
			Args:                a.Args,
			Subcommands:         a.Commands,
			Action:              a.Action,
			Description:         a.Description,
			Data:                data,
			After:               a.After,
			Uses:                a.Uses,
			Before:              a.Before,
			Options:             a.Options,
			flags:               flags,
			hooksSupport:        hooks.append(&a.hooksSupport),
			customizableSupport: customs.append(&a.customizableSupport),
		}
	}
	return a.rootCommand
}

// SetData sets the specified metadata on the app
func (a *App) SetData(name string, v interface{}) {
	a.ensureData()[name] = v
}

func (a *App) ensureData() map[string]interface{} {
	if a.Data == nil {
		a.Data = map[string]interface{}{}
	}
	return a.Data
}

func (a *App) runContextCore(c context.Context, args []string) (*Context, error) {
	ctx, err := a.Initialize(c)
	if err != nil {
		return ctx, err
	}

	root := a.createRoot()
	err = root.parseAndExecuteSelf(ctx, args)
	return ctx, err
}

// Initialize sets up the app with the given context and arguments
func (a *App) Initialize(c context.Context) (*Context, error) {
	ctx := rootContext(c, a)
	err := ctx.initialize()
	return ctx, err
}

func (a *App) ensureTemplates() map[string]string {
	if a.templates == nil {
		a.templates = map[string]string{}
	}
	return a.templates
}

func (a *App) ensureTemplateFuncs() map[string]interface{} {
	if a.templateFuncs == nil {
		a.templateFuncs = map[string]interface{}{}
	}
	return a.templateFuncs
}

func (a *App) defaultFS() fs.FS {
	if a.FS == nil {
		return newDefaultFS(a.Stdin, a.Stdout)
	}
	return a.FS
}

func (a *appContext) initialize(c *Context) error {
	var err error
	if a.app.rootCommandCreator == nil {
		err = a.defaultInitRootCommand(c)
	} else {
		err = a.factoryInitRootCommand(c)
	}

	if err != nil {
		return err
	}
	return a.commandContext.initializeCore(c)
}

func (a *appContext) defaultInitRootCommand(c *Context) error {
	rest := newPipelines(ActionOf(a.app.Uses), &a.app.Options)

	a.commandContext.cmd = a.app.createRoot()
	a.commandContext.cmd.setPipelines(rest)

	if err := executeAll(c, rest.Initializers, defaultApp.Initializers); err != nil {
		return err
	}

	// Re-create the root command because middleware may have changed things.
	a.commandContext.cmd = a.app._createRootCore(true)

	// We must also pierce the encapsulation of command context initialization here
	// (by calling initializeCore instead of initialize) because we
	// don't want to calculate separating out action pipelines again, and we
	// don't want to invoke app's initializers twice
	a.commandContext.cmd.setPipelines(rest.exceptInitializers())

	return nil
}

func (a *appContext) factoryInitRootCommand(c *Context) error {
	a.commandContext.cmd = a.app.rootCommandCreator()
	if err := executeAll(c, defaultApp.Initializers); err != nil {
		return err
	}

	a.commandContext.cmd.setPipelines(&actionPipelines{})
	return nil
}

func (a *appContext) target() target {
	return a.commandContext.cmd
}

func (a *appContext) Name() string { return a.app.Name }

func exit(c *Context, err error) {
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
	handler(c, err.Error(), code)
}

func defaultExitHandler(c *Context, message string, status int) {
	stderr := c.Stderr
	if stderr == nil {
		stderr = NewWriter(os.Stderr)
	}
	if message != "" && status != 0 {
		fmt.Fprintln(stderr, message)
	}
	os.Exit(status)
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

	c.contextData.Stdin = a.Stdin
	c.contextData.Stdout = adaptWriter(a.Stdout)
	c.contextData.Stderr = adaptWriter(a.Stderr)
	return nil
}

func adaptWriter(w io.Writer) Writer {
	if a, ok := w.(Writer); ok {
		return a
	}
	return NewWriter(w)
}

func setupDefaultTemplateFuncs(c *Context) error {
	width := guessWidth()

	funcMap := template.FuncMap{
		"Join": func(v string, args []string) string {
			return strings.Join(args, v)
		},
		"Trim": strings.TrimSpace,
		"Wrap": func(indent int, s string) string {
			buf := bytes.NewBuffer(nil)
			indentText := strings.Repeat(" ", indent)
			doc.ToText(buf, s, "", "  "+indentText, width-indent)
			return buf.String()
		},
		"BoldFirst": func(args []string) []string {
			args[0] = bold(args[0])
			return args
		},
		"ExtraSpaceBeforeFlag": func(s string) string {
			if strings.HasPrefix(controlCodes.ReplaceAllString(s, ""), "--") {
				return "    " + s
			}
			return s
		},
		"SynopsisHangingIndent": func(d *commandData) string {
			var buf bytes.Buffer
			hang := strings.Repeat(
				" ",
				len("usage:")+lenIgnoringCSI(d.Lineage)+len(d.Name)+1,
			)

			buf.WriteString(d.Lineage)

			limit := width - len("usage:") - lenIgnoringCSI(d.Lineage) - 1
			for _, t := range d.Synopsis {
				tLength := lenIgnoringCSI(t)
				if limit-tLength < 0 {
					buf.WriteString("\n")
					buf.WriteString(hang)
					limit = width - len(hang)
				}

				buf.WriteString(" ")
				buf.WriteString(t)
				limit = limit - 1 - tLength
			}
			return buf.String()
		},
	}

	a := c.App()
	funcs := a.ensureTemplateFuncs()
	for k, v := range funcMap {
		if _, ok := funcs[k]; !ok {
			funcs[k] = v
		}
	}
	return nil
}

func setupDefaultTemplates(c *Context) error {
	a := c.App()
	templates := a.ensureTemplates()
	for k, v := range defaultTemplates {
		if _, ok := templates[k]; !ok {
			templates[k] = v()
		}
	}
	return nil
}

func addAppCommand(name string, f *Flag, cmd *Command) ActionFunc {
	return func(c *Context) error {
		app := c.App()
		if len(app.Commands) > 0 {
			if _, ok := app.Command(name); !ok {
				app.Commands = append(app.Commands, cmd)
			}
		}

		if _, ok := app.Flag(name); !ok {
			app.Flags = append(app.Flags, f)
		}
		return nil
	}
}

var _ hookable = (*App)(nil)
