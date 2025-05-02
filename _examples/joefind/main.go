// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/Carbonfrost/joe-cli"
)

func main() {
	app := &cli.App{
		Name: "cli",
		Args: []*cli.Arg{
			{
				Name: "path",
				Value: &cli.File{
					Name: ".",
				},
			}, {
				Value: &cli.Expression{
					Exprs: []*cli.Expr{
						{Name: "cnewer", Args: cli.Args("file", new(cli.File))},
						{Name: "ctime", Args: cli.Args("n", cli.Int())},
						{Name: "empty"},
						{Name: "false"},
						{Name: "readable"},
						{Name: "writable"},
						{Name: "executable", Evaluate: cli.Predicate(isExecutable)},
					},
				},
			},
		},
		Action: func(c *cli.Context) {
			exp := c.Expression("expression")
			exp.Prepend(cli.NewExprBinding(cli.EvaluatorFunc(walker)))
			exp.Append(cli.NewExprBinding(cli.Predicate(printer)))

			// Pass true to the expression pipeline in order to ensure that the
			// first expression binding has an input
			exp.Evaluate(c, true)
		},
	}
	app.Run(os.Args)
}

func isExecutable(v interface{}) bool {
	// Whether the file is executable, but don't count directories
	in, _ := v.(*info).Info()
	return !in.IsDir() && in.Mode()&0100 != 0
}

func walker(c *cli.Context, _ interface{}, yield func(interface{}) error) error {
	c.File("path").Walk(func(path string, d fs.DirEntry, _ error) error {
		yield(&info{
			DirEntry: d,
			Path:     path,
		})
		return nil
	})
	return nil
}

func printer(v interface{}) bool {
	info := v.(*info)
	fmt.Println(info.Path)
	return true
}

type info struct {
	fs.DirEntry
	Path string
}
