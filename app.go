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

	// Action specifies the action to run for the app, assuming no other more specific command
	// has been selected.  Refer to cli.Action about the correct function signature to use.
	Action interface{}

	// Before executes before the app action or any sub-command action runs.
	// Refer to cli.Action about the correct function signature to use.
	Before interface{}

	HelpText  string
	UsageText string
}

var (
	ExitHandler func(string, int)
)

func (a *App) Run(args []string) {
	exit(a.RunContext(context.TODO(), args))
}

func (a *App) RunContext(ctx context.Context, args []string) error {
	root := a.createRoot(args[0])
	return root.parseAndExecute(rootContext(ctx, a), args)
}

func (a *App) createRoot(name string) *Command {
	return &Command{
		Name:        name,
		Flags:       a.Flags,
		Args:        a.Args,
		Subcommands: a.Commands,
		Action:      a.Action,

		// Hooks are intentionally left nil because App handles its hooks
		// from the root context
		Before: nil,
	}
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
	if message != "" {
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
