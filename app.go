package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	HelpText  string
	UsageText string

	rootCommand *Command
}

var (
	ExitHandler func(string, int)
)

// Run the application and exit using the exit handler.  This function exits using the
// ExitHandler if an error occurs.  If you want to process the error yourself, use RunContext.
func (a *App) Run(args []string) {
	exit(a.RunContext(context.TODO(), args))
}

func (a *App) RunContext(c context.Context, args []string) error {
	ctx := rootContext(c, a)
	err := ctx.executeBefore()
	if err != nil {
		return err
	}

	root := a.createRoot()
	return root.parseAndExecute(ctx, args)
}

func (a *App) Command(name string) (*Command, bool) {
	return findCommandByName(a.Commands, name)
}

func (a *App) Flag(name string) (*Flag, bool) {
	return findFlagByName(a.Flags, name)
}

func (a *App) Arg(name string) (*Arg, bool) {
	return findArgByName(a.Args, name)
}

func (a *App) createRoot() *Command {
	if a.rootCommand == nil {
		a.rootCommand = &Command{
			Name:        a.Name,
			Flags:       a.Flags,
			Args:        a.Args,
			Exprs:       a.Exprs,
			Subcommands: a.Commands,
			Action:      a.Action,
			Description: a.Description,

			// Hooks are intentionally left nil because App handles its hooks
			// from the root context
			Before: nil,
		}
	}
	return a.rootCommand
}

func exit(err error) {
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
	handler(err.Error(), code)
}

func defaultExitHandler(message string, status int) {
	if message != "" && status != 0 {
		fmt.Fprintln(os.Stderr, message)
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
	a := c.target.(*App)
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
	a := c.target.(*App)
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

func addAppCommand(name string, f *Flag, cmd *Command) ActionFunc {
	return func(c *Context) error {
		app := c.target.(*App)
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

var _ command = &App{}
