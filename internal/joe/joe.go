package joe

import (
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/joe-cli/internal/build"
)

func Run() {
	NewApp().Run(os.Args)
}

func NewApp() *cli.App {
	return &cli.App{
		Name:     "joe",
		HelpText: "Easily generate a new Joe-cli app or utility",
		Version:  internal.Version,
		Flags: []*cli.Flag{
			{
				Name:     "force",
				HelpText: "Overwrite files and accept all prompts",
				Value:    cli.Bool(),
			},
			{
				Name:     "dry-run",
				HelpText: "Display what commands will be run without actually executing them",
				Value:    cli.Bool(),
			},
		},
		Commands: []*cli.Command{
			{
				Name:     "new",
				HelpText: "Generate apps, commands, etc. based on a template",
				Subcommands: []*cli.Command{
					newAppCommand(),
				},
			},
			{
				Name:     "init",
				HelpText: "Initialize the current Go module for use with Joe-cli",
				Action:   glueTemplateOptions(newInitTemplate),
			},
		},
	}
}

func glueTemplateOptions(root func() *template.Root) cli.ActionFunc {
	return func(c *cli.Context) error {
		t := root()
		_ = t.SetDryRun(c.Bool("dry-run"))
		_ = t.SetOverwrite(c.Bool("force"))
		return c.Do(t)
	}
}
