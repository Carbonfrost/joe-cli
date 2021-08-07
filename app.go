package cli

import (
	"context"
	"fmt"
	"os"
)

// App provides the definition of an app, which is composed of commands, flags, and arguments.
type App struct {
	// Name specifies the name of the app
	Name string

	// Comands provides the list of commands in the app
	Commands []*Command

	// Flags supplies global flags for use with the app
	Flags []Flag

	// Args will be bound for non-command arguments
	Args []Arg

	// Action specifies the action to run for the app, assuming no other more specific command
	// has been selected
	Action ActionFunc

	// Before executes before the app action or any sub-command action runs
	Before ActionFunc
}

var (
	ExitHandler func(string, int)
)

func (a *App) Run(args []string) {
	exit(a.RunContext(context.TODO(), args))
}

func (a *App) RunContext(ctx context.Context, args []string) error {
	root := a.createRoot(args[0])
	return root.parseAndExecute(rootContext(ctx), args)
}

func (a *App) createRoot(name string) *Command {
	return &Command{
		Name:        name,
		Flags:       a.Flags,
		Args:        a.Args,
		Subcommands: a.Commands,
		Action:      a.Action,
		Before:      a.Before,
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
