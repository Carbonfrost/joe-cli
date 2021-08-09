package main

import (
	"fmt"
	"os"

	"github.com/Carbonfrost/joe-cli"
)

func main() {
	app := &cli.App{
		Name: "cli",
		Flags: []*cli.Flag{
			{
				Name: "version",
			},
			{
				Name: "plus",
			},
			{
				Name: "time",
			},
		},
		Commands: []*cli.Command{
			{
				Name: "generate",
				Args: []*cli.Arg{
					{
						Name: "kind",
					},
				},
				Action: cli.Action(func() error {
					fmt.Println("TODO: handle generating")
					return nil
				}),
			},
		},
	}

	app.Run(os.Args)
}
