package cli_test

import (
	"bytes"
	"io"
	"os"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"

	. "github.com/onsi/ginkgo"
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
			app.RunContext(nil, args)
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
					help, _ = c.App().Flag("help")
				},
			}

			app.RunContext(nil, []string{"app"})
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

			_ = app.RunContext(nil, []string{"app", "--help"})
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
					version, _ = c.App().Flag("version")
				},
			}

			app.RunContext(nil, []string{"app"})
			Expect(version).ToNot(BeNil())
		})

		It("prints default version output", func() {
			var (
				capture bytes.Buffer
			)
			app := &cli.App{
				Name:    "hunter",
				Version: "1.619",
				Stderr:  &capture,
			}

			_ = app.RunContext(nil, []string{"app", "--version"})
			Expect(capture.String()).To(HavePrefix("hunter, version 1.619"))
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

			app.RunContext(nil, []string{"app"})

			Expect(in).To(Equal(os.Stdin))
			Expect(out).To(Equal(os.Stdout))
			Expect(err).To(Equal(os.Stderr))
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

			app.RunContext(nil, []string{"app", "s"})

			Expect(in).To(Equal(os.Stdin))
			Expect(out).To(Equal(os.Stdout))
			Expect(err).To(Equal(os.Stderr))
		})
	})

})
