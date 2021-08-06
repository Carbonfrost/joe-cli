package cli

import (
	"context"
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

func (a *App) Run(args []string) {
	_ = a.RunContext(context.TODO(), args)
}

func (a *App) RunContext(ctx context.Context, args []string) error {
	return parseAndExecute(ctx, a.createRoot(args[0]), args[1:])
}

func (a *App) createRoot(name string) *Command {
	return &Command{
		Name:        name,
		Flags:       a.Flags,
		Subcommands: a.Commands,
		Action:      a.Action,
		Before:      a.Before,
	}
}

func parseAndExecute(cctx context.Context, current *Command, args []string) error {
	ctx := current.newContext(cctx, nil)
	for len(args) > 0 {
		err := ctx.set.Getopt(args, nil)
		if err != nil {
			// Failed to set the option to the corresponding flag
			return err
		}
		args = ctx.set.Args()

		// Args were modified by Getopt to apply any flags and stopped
		// at the first argument.  If the argument matches a sub-command, then
		// we push the command onto the stack
		if len(args) > 0 {
			if sub, ok := current.Command(args[0]); ok {
				current = sub
				ctx = current.newContext(cctx, ctx)
			} else {
				// Stop looking for commands; this is it
				break
			}
		}
	}

	current.applyArgs(ctx, args)
	err := execute(current.Before, ctx)
	if err != nil {
		return err
	}

	return execute(current.Action, ctx)
}
