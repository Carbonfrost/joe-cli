// Copyright 2023, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/expr"
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
				Value: &expr.Expression{
					Exprs: []*expr.Expr{
						{Name: "cnewer", Args: cli.Args("file", new(cli.File))},
						{Name: "ctime", Args: cli.Args("n", cli.Int())},
						{Name: "empty"},
						{Name: "false"},
						{Name: "readable"},
						{Name: "writable"},
						{Name: "executable", Evaluate: expr.Predicate(isExecutable)},
					},
					Compiler: func(be []expr.BindingEvaluator) (expr.Evaluator, error) {
						var items []expr.BindingEvaluator
						items = append(items, expr.NewBindingEvaluator(expr.EvaluatorFunc(walker)))
						items = append(items, be...)
						items = append(items, expr.NewBindingEvaluator(expr.Predicate(printer)))
						return expr.Compile(items)
					},
				},
			},
		},
		// Pass true to the expression pipeline in order to ensure that the
		// first expression binding has an input
		Action: expr.Evaluate(true),
	}
	app.Run(os.Args...)
}

func isExecutable(v any) bool {
	// Whether the file is executable, but don't count directories
	in, _ := v.(*info).Info()
	return !in.IsDir() && in.Mode()&0100 != 0
}

func walker(c *cli.Context, _ any, yield func(any) error) error {
	c.File("path").Walk(func(path string, d fs.DirEntry, _ error) error {
		yield(&info{
			DirEntry: d,
			Path:     path,
		})
		return nil
	})
	return nil
}

func printer(v any) bool {
	info := v.(*info)
	fmt.Println(info.Path)
	return true
}

type info struct {
	fs.DirEntry
	Path string
}
