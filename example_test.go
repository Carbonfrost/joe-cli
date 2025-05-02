// Copyright 2022 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
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
