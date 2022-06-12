package main

import (
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"
)

func main() {
	app := &cli.App{
		Name:     "joeopen",
		HelpText: "Open a file or URL, optionally in the specified app",
		Args: []*cli.Arg{
			{
				Name:  "path",
				Value: new(cli.File),
			},
		},
		Flags: []*cli.Flag{
			{
				Name:    "app",
				Aliases: []string{"a"},
				Value:   new(string),
			},
		},
		Action: func(c *cli.Context) error {
			file := c.File("path")
			app := c.String("app")
			return exec.Open(file.Name, app)
		},
	}
	app.Run(os.Args)
}
