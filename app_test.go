// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package cli_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("App", func() {

	Describe("actions", func() {

		var (
			act       *joeclifakes.FakeAction
			beforeAct *joeclifakes.FakeAction
			usesAct   *joeclifakes.FakeAction
			arguments string

			app *cli.App
		)

		JustBeforeEach(func() {
			act = new(joeclifakes.FakeAction)
			beforeAct = new(joeclifakes.FakeAction)
			usesAct = new(joeclifakes.FakeAction)

			app = &cli.App{
				Commands: []*cli.Command{
					{
						Name: "c",
					},
				},
				Flags: []*cli.Flag{
					{
						Name: "f",
					},
				},
				Action: act,
				Before: beforeAct,
				Uses:   usesAct,
			}

			args, _ := cli.Split(arguments)
			app.RunContext(context.Background(), args)
		})

		Context("when executing itself", func() {
			BeforeEach(func() {
				arguments = "cli"
			})

			It("executes action", func() {
				Expect(act.ExecuteCallCount()).To(Equal(1))
			})

			It("executes before action", func() {
				Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
			})

			It("executes uses action", func() {
				Expect(usesAct.ExecuteCallCount()).To(Equal(1))
			})
		})

		Context("when executing a sub-command", func() {
			BeforeEach(func() {
				arguments = "cli c"
			})

			It("does not execute action on executing sub-command", func() {
				Expect(act.ExecuteCallCount()).To(Equal(0))
			})

			It("executes before action on executing sub-command", func() {
				Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
			})
		})

		Context("when setting a flag", func() {
			BeforeEach(func() {
				arguments = "cli -f a"
			})

			It("executes before action on executing self", func() {
				Expect(beforeAct.ExecuteCallCount()).To(Equal(1))
			})
		})

	})

	Describe("help", func() {
		It("sets up default --help flag", func() {
			var (
				help *cli.Flag
			)
			app := &cli.App{
				Action: func(c *cli.Context) {
					help, _ = c.Command().Flag("help")
				},
			}

			app.RunContext(context.Background(), []string{"app"})
			Expect(help).ToNot(BeNil())
		})

		It("prints default help output", func() {
			var (
				capture bytes.Buffer
			)
			defer disableConsoleColor()()
			app := &cli.App{
				Name:   "hunter",
				Stderr: &capture,
			}

			_ = app.RunContext(context.Background(), []string{"app", "--help"})
			Expect(capture.String()).To(HavePrefix("usage: hunter "))
		})
	})

	Describe("version", func() {
		It("sets up default --version flag", func() {
			var (
				version *cli.Flag
			)
			app := &cli.App{
				Action: func(c *cli.Context) {
					version, _ = c.Command().Flag("version")
				},
			}

			app.RunContext(context.Background(), []string{"app"})
			Expect(version).ToNot(BeNil())
		})

		It("prints default version output", func() {
			var (
				capture bytes.Buffer
			)
			app := &cli.App{
				Name:    "hunter",
				Version: "1.619",
				Stdout:  &capture, // Python 2 -> 3 changed from stderr to stdout
			}

			_ = app.RunContext(context.Background(), []string{"app", "--version"})
			Expect(capture.String()).To(HavePrefix("hunter, version 1.619"))
		})
	})

	Describe("NewApp", func() {

		It("runs default app pipeline (such as setting up app version flag)", func() {
			var (
				version *cli.Flag
			)
			app := cli.NewApp(&cli.Command{
				Name: "app",
				Action: func(c *cli.Context) {
					version, _ = c.Command().Flag("version")
				},
			})

			err := app.RunContext(context.Background(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())
			Expect(version).ToNot(BeNil())
		})
	})

	Describe("i/o", func() {

		It("sets up default I/O from standard files", func() {
			var (
				in  io.Reader
				out io.Writer
				err io.Writer
			)
			app := &cli.App{
				Action: func(c *cli.Context) {
					in, out, err = c.Stdin, c.Stdout, c.Stderr
				},
			}

			app.RunContext(context.Background(), []string{"app"})

			Expect(in).To(Equal(os.Stdin))
			Expect(out).To(Equal(cli.NewWriter(os.Stdout)))
			Expect(err).To(Equal(cli.NewWriter(os.Stderr)))
		})

		It("sets up default file system", func() {
			var f fs.FS
			app := &cli.App{
				Action: func(c *cli.Context) {
					f = c.FS
				},
			}

			app.RunContext(context.Background(), []string{"app"})
			Expect(f).To(Equal(cli.DefaultFS()))
		})

		It("sets up I/O in nested commands", func() {
			var (
				in  io.Reader
				out io.Writer
				err io.Writer
			)
			app := &cli.App{
				Commands: []*cli.Command{
					{
						Name: "s",
						Action: func(c *cli.Context) {
							in, out, err = c.Stdin, c.Stdout, c.Stderr
						},
					},
				},
			}

			app.RunContext(context.Background(), []string{"app", "s"})

			Expect(in).To(Equal(os.Stdin))
			Expect(out).To(Equal(cli.NewWriter(os.Stdout)))
			Expect(err).To(Equal(cli.NewWriter(os.Stderr)))
		})

		It("sets up file system nested", func() {
			var f fs.FS
			app := &cli.App{
				Commands: []*cli.Command{
					{
						Name: "s",
						Action: func(c *cli.Context) {
							f = c.FS
						},
					},
				},
			}

			app.RunContext(context.Background(), []string{"app", "s"})
			Expect(f).To(Equal(cli.DefaultFS()))
		})
	})

	Describe("CurrentApp", func() {
		It("will be set to the app that executes", func() {
			var what *cli.App
			app := &cli.App{
				Action: func(c *cli.Context) {
					what = cli.CurrentApp()
				},
			}
			app.RunContext(context.Background(), []string{"app"})
			Expect(what).To(BeIdenticalTo(app))
		})

		It("will be clear after app executed", func() {
			app := &cli.App{
				Action: func() {},
			}
			app.RunContext(context.Background(), []string{"app"})
			Expect(cli.CurrentApp()).To(BeNil())
		})
	})
})

var _ = Describe("Run", func() {

	It("renders message to Stderr", func() {
		var buf bytes.Buffer
		cli.SetOSExit(func(v int) {
		})
		app := &cli.App{
			Name:   "app",
			Stderr: &buf,
			Action: func() error {
				return cli.Exit("my error message")
			},
		}

		app.Run([]string{"app"})
		Expect(buf.String()).To(Equal("my error message\n"))
	})

	It("exits with corresponding code", func() {
		var buf bytes.Buffer
		var exitCode int
		cli.SetOSExit(func(v int) {
			exitCode = v
		})
		app := &cli.App{
			Name:   "app",
			Stderr: &buf,
			Action: func() error {
				return cli.Exit(3)
			},
		}

		app.Run([]string{"app"})
		Expect(exitCode).To(Equal(3))
	})
})
