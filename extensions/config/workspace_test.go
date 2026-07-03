// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"runtime"
	"slices"
	"testing/fstest"

	cli "github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("DefaultWorkspaceFinder", func() {

	DescribeTable("examples", func(cwd string, expected types.GomegaMatcher) {
		var finder = config.DefaultWorkspaceFinder.FindWorkspacePath

		testFS := fstest.MapFS{
			"a":                       {Mode: fs.ModeDir},
			mustLocalize("a/b/c/d/e"): {Mode: fs.ModeDir},
			mustLocalize("a/.config"): {Mode: fs.ModeDir},
		}
		actual, _ := finder(context.Background(), testFS, cwd)

		Expect(actual).To(expected)
	},
		Entry("nominal", "a", Equal("a")),
		Entry("ancestor", mustLocalize("a/b/c/d/e"), Equal("a")),
		Entry("unknown", "unknown", Equal("")),
	)

	It("it defaults to app sentinel in app context", func() {
		var finder = config.DefaultWorkspaceFinder.FindWorkspacePath
		testFS := fstest.MapFS{
			mustLocalize("x/.myapp"): {Mode: fs.ModeDir},
		}

		app := &cli.App{
			Name: "myapp",
		}
		ctx, _ := app.Initialize(context.Background())
		actual, _ := finder(ctx, testFS, mustLocalize("x/y"))

		Expect(actual).To(Equal("x"))
	})
})

var _ = Describe("PrintEnv", func() {

	DescribeTable("examples", func(printEnv cli.Action, arguments string, expected string) {
		SkipOnWindows()
		var captured bytes.Buffer

		app := &cli.App{
			Uses: cli.Pipeline(
				config.NewWorkspace(
					config.WithEnvProvider(config.EnvMap{"HELLO": "R", "XXX": "S"}),
				),
				printEnv,
			),
			Stdout: &captured,
		}
		args, _ := cli.Split(arguments)
		app.RunContext(context.Background(), args)
		Expect(captured.String()).To(Equal(expected))
	},
		Entry("variable", config.PrintEnv(), "app HELLO", "R\n"),
		Entry("variables", config.PrintEnv(), "app HELLO XXX", "R\nS\n"),
		Entry("all variables", config.PrintEnv(), "app", "export HELLO=R\nexport XXX=S\n"),
		Entry("specified variable", config.PrintEnv("XXX"), "app", "S\n"),
	)
})

var _ = Describe("Workspace", func() {

	It("sets up the workspace flag in app", func() {
		app := &cli.App{
			Name: "myapp",
			Uses: config.NewWorkspace(),
		}
		_, err := app.Initialize(context.Background())
		Expect(err).NotTo(HaveOccurred())

		flag, _ := app.Flag("myapp-dir")
		Expect(flag).NotTo(BeNil())
	})

	Describe("ConfigDir", func() {

		It("sets up from generic context", func() {
			ws := config.NewWorkspace()
			config.CompleteSetup(context.Background(), ws)

			actual := ws.ConfigDir()
			Expect(actual).To(Equal(".config"))
		})

		It("sets up from app context", func() {
			app := &cli.App{
				Name: "myapp",
			}
			ctx, _ := app.Initialize(context.Background())

			ws := config.NewWorkspace()
			config.CompleteSetup(ctx, ws)

			actual := ws.ConfigDir()
			Expect(actual).To(Equal(".myapp"))
		})

		It("sets up config dir in run", func() {
			var ws *config.Workspace

			app := &cli.App{
				Name: "foo",
				Uses: config.NewWorkspace(),

				Action: func(c context.Context) {
					ws = config.WorkspaceFromContext(c)
				},
			}

			args, _ := cli.Split("foo --foo-dir=.alternate")
			err := app.RunContext(context.Background(), args)
			Expect(err).NotTo(HaveOccurred())

			actual := ws.ConfigDir()
			Expect(actual).To(Equal(".alternate"))
		})
	})

	Describe("Files", func() {

		It("loads files from workspace", func() {
			ws := config.NewWorkspace(
				config.WithFS(fstest.MapFS{
					"a":   {},
					"b/a": {},
					"b/b": {},
				}),
			)
			config.CompleteSetup(context.Background(), ws)

			files := ws.Files()

			var actual []string
			for f := range files {
				actual = append(actual, f)
			}
			Expect(actual).To(ConsistOf("a", "b/a", "b/b"))
		})
	})

	Describe("LoadFiles", func() {

		It("loads files from workspace", func() {
			ws := config.NewWorkspace(
				config.WithFS(fstest.MapFS{
					"a":   {},
					"b/a": {},
					"b/b": {},
				}),
				config.WithFileLoader(func(_ fs.FS, name string, _ fs.DirEntry) (any, error) {
					return "data:" + name, nil
				}),
			)
			config.CompleteSetup(context.Background(), ws)

			Expect(slices.Collect(ws.LoadFiles())).To(ConsistOf("data:a", "data:b/a", "data:b/b"))
		})
	})

})

func mustLocalize(s string) string {
	s, err := filepath.Localize(s)
	if err != nil {
		panic(err)
	}
	return s
}

func SkipOnWindows() {
	if runtime.GOOS == "windows" {
		Skip("not tested on Windows")
	}
}
