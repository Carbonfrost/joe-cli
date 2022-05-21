package cli_test

import (
	"fmt"

	"github.com/Carbonfrost/joe-cli"
)

func ExampleAccessory() {
	(&cli.App{
		Name: "app",
		Args: []*cli.Arg{
			{
				Name:  "files",
				Value: new(cli.FileSet),
				Uses:  cli.Accessory("recursive", (*cli.FileSet).RecursiveFlag),
			},
		},
		Action: func(c *cli.Context) {
			fmt.Println(c.Command().Synopsis())
		},
	}).Run([]string{"app"})
	// Output:
	// app {--help | --version} [--recursive] [<files>]
}
