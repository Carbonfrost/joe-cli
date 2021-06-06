package main

import (
	"fmt"
	"os"

	"github.com/Carbonfrost/gocli"
)

func main() {
	app := &gocli.App{
		Name: "gocli",
		Flags: []gocli.Flag{
			&gocli.StringFlag{
				Name: "version",
			},
			&gocli.StringFlag{
				Name: "plus",
			},
			&gocli.StringFlag{
				Name: "time",
			},
		},
		Commands: []*gocli.Command{
			{
				Name: "generate",
				Args: []gocli.Arg{
					&gocli.StringArg{
						Name: "kind",
					},
				},
				Action: func(c *gocli.Context) error {
					fmt.Println("TODO: handle generating")
					return nil
				},
			},
		},
	}

	app.Run(os.Args)
}
