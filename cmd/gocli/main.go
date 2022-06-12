package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
)

func main() {
	app := &cli.App{
		Name: filepath.Base(os.Args[0]),
		Flags: []*cli.Flag{
			{
				Name:     "will",
				HelpText: "a useful measure of {POWER}",
			},
			{
				Name:     "plus",
				HelpText: "the only operator that works",
				Value:    cli.Bool(),
			},
			{
				Name:     "time",
				HelpText: "an absolute property",
				Value:    cli.Duration(),
			},
		},
		Commands: []*cli.Command{
			{
				Name:     "generate",
				HelpText: "Generate something useful",
				Args: []*cli.Arg{
					{
						Name: "kind",
					},
				},
				Action: cli.ActionOf(func() error {
					fmt.Println("TODO: handle generating")
					return nil
				}),
				Subcommands: []*cli.Command{
					{
						Name:     "loki",
						HelpText: "Generate something useful",
						Args: []*cli.Arg{
							{
								Name: "kind",
							},
						},
						Action: cli.ActionOf(func() error {
							fmt.Println("TODO: handle generating")
							return nil
						}),
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
