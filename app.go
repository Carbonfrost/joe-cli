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

	// Exprs will provide expression evaluation
	Exprs []*Expr

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
	Uses interface{}

	// Data provides an arbitrary mapping of additional data.  This data can be used by
	// middleware and it is made available to templates
	Data map[string]interface{}

	// Options sets common options for use with the app
	Options Option

	// FS specifies the file system that is used by default for Files.
	FS fs.FS

	HelpText  string
	UsageText string

	rootCommand   *Command
	appHooks      hooks
	flags         internalFlags
	templateFuncs map[string]interface{}
	templates     map[string]string
}

type appContext struct {
	*commandContext
	app_ *App
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
	}
)

// Run the application and exit using the exit handler.  This function exits using the
// ExitHandler if an error occurs.  If you want to process the error yourself, use RunContext.
func (a *App) Run(args []string) {
	a.runContextCore(context.TODO(), args, func(c *Context, err error) error {
		exit(c, err)
		return nil
	})
}

// RunContext runs the application with the specified context and returns any error
// that occurred.
func (a *App) RunContext(c context.Context, args []string) error {
	return a.runContextCore(c, args, func(_ *Context, err error) error {
		return err
	})
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

// Expr  gets the expression by name
func (a *App) Expr(name string) (*Expr, bool) {
	return findExprByName(a.Exprs, name)
}

func (a *App) createRoot() *Command {
	return a._createRootCore(false)
}

func (a *App) _createRootCore(force bool) *Command {
	if a.rootCommand == nil || force {
		var flags internalFlags
		if a.rootCommand != nil {
			flags = a.rootCommand.flags
		}
		flags |= a.flags

		a.rootCommand = &Command{
			Name:        a.Name,
			Flags:       a.Flags,
			Args:        a.Args,
			Exprs:       a.Exprs,
			Subcommands: a.Commands,
			Action:      a.Action,
			Description: a.Description,
			Data:        a.Data,
			After:       a.After,
			Uses:        a.Uses,
			Before:      a.Before,
			Options:     a.Options,
			flags:       flags,
			cmdHooks:    a.appHooks,
		}
	}
	return a.rootCommand
}

func (a *App) SetData(name string, v interface{}) {
	a.ensureData()[name] = v
}

func (a *App) ensureData() map[string]interface{} {
	if a.Data == nil {
		a.Data = map[string]interface{}{}
	}
	return a.Data
}

func (a *App) runContextCore(c context.Context, args []string, exit func(*Context, error) error) error {
	ctx := rootContext(c, a, args)
	ctx.initialize()
	root := a.createRoot()
	err := root.parseAndExecuteSelf(ctx)
	return exit(ctx, err)
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

func (a *appContext) initialize(c *Context) error {
	rest, err := takeInitializers(ActionOf(a.app_.Uses), a.app_.Options, c)

	a.commandContext.cmd = a.app_.createRoot()
	a.commandContext.cmd.uses = rest
	if err != nil {
		return err
	}

	if err := executeAll(c, rest.Initializers, defaultApp.Initializers); err != nil {
		return err
	}

	// Re-create the root command because middleware may have changed things.
	a.commandContext.cmd = a.app_._createRootCore(true)

	// We must also pierce the encapsulation of command context initialization here
	// (by calling initializeCore instead of initialize) because we
	// don't want to calculate separating out action pipelines again, and we
	// don't want to invoke app's initializers twice
	a.commandContext.cmd.uses = rest.exceptInitializers()
	a.commandContext.initializeCore(c)

	for _, sub := range a.app_.Commands {
		err := c.commandContext(sub, nil).initialize()
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *appContext) hooks() *hooks {
	return &a.app_.appHooks
}

func (a *appContext) app() (*App, bool) { return a.app_, true }
func (a *appContext) target() target {
	return a.commandContext.cmd
}

func (a *appContext) Name() string { return a.app_.Name }

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
		stderr = os.Stderr
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
	a := c.app()
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
	a := c.app()
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
	c.contextData.Stdout = a.Stdout
	c.contextData.Stderr = a.Stderr
	return nil
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
			doc.ToText(buf, s, indentText, "  "+indentText, width-indent)
			return buf.String()
		},
		"BoldFirst": func(args []string) []string {
			args[0] = bold.Open + args[0] + bold.Close
			return args
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

	a := c.app()
	funcs := a.ensureTemplateFuncs()
	for k, v := range funcMap {
		if _, ok := funcs[k]; !ok {
			funcs[k] = v
		}
	}
	return nil
}

func setupDefaultTemplates(c *Context) error {
	a := c.app()
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
		app := c.app()
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
