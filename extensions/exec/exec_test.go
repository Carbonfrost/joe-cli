// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"
	joeclifakes "github.com/Carbonfrost/joe-cli/internal/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaveLookPath", func() {

	It("finds the path", func() {
		fakeAction := new(joeclifakes.FakeAction)
		app := &cli.App{
			Name:   "app",
			Action: cli.IfMatch(exec.HaveLookPath("go"), fakeAction),
		}
		_ = app.RunContext(context.Background(), nil...)

		Expect(fakeAction.ExecuteCallCount()).To(Equal(1))
	})

})

var _ = Describe("ExternalCommand", func() {

	writeExecutable := func(name, body string) {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, name)
		Expect(os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755)).To(Succeed())
		GinkgoT().Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	newApp := func(uses cli.Action, stdout *bytes.Buffer) *cli.App {
		return &cli.App{
			Name: "app",
			Commands: []*cli.Command{
				{Name: "builtin"},
			},
			Uses:   uses,
			Stdout: stdout,
			Stderr: os.Stderr,
		}
	}

	BeforeEach(func() {
		SkipOnWindows()
	})

	It("runs a matching external executable", func() {
		writeExecutable("app-greet", "echo hello from plugin")

		var out bytes.Buffer
		app := newApp(exec.ExternalCommand(), &out)

		args, _ := cli.Split("app greet")
		err := app.RunContext(context.Background(), args...)

		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(Equal("hello from plugin\n"))
	})

	It("passes the remaining arguments to the executable", func() {
		writeExecutable("app-greet", `printf '%s\n' "$@"`)

		var out bytes.Buffer
		app := newApp(exec.ExternalCommand(), &out)

		args, _ := cli.Split("app greet alpha beta --flag")
		err := app.RunContext(context.Background(), args...)

		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(Equal("alpha\nbeta\n--flag\n"))
	})

	It("propagates the exit code without an extra message", func() {
		writeExecutable("app-fail", "exit 3")

		var out bytes.Buffer
		app := newApp(exec.ExternalCommand(), &out)

		args, _ := cli.Split("app fail")
		err := app.RunContext(context.Background(), args...)

		Expect(err).To(HaveOccurred())
		coder, ok := err.(cli.ExitCoder)
		Expect(ok).To(BeTrue())
		Expect(coder.ExitCode()).To(Equal(3))
		Expect(err.Error()).To(BeEmpty())
	})

	It("defers to the default behavior when no executable matches", func() {
		var out bytes.Buffer
		app := newApp(exec.ExternalCommand(), &out)

		args, _ := cli.Split("app missing")
		err := app.RunContext(context.Background(), args...)

		Expect(err).To(MatchError(`"missing" is not a command`))
	})

	It("uses a custom prefix when specified", func() {
		writeExecutable("tool-do", "echo custom prefix")

		var out bytes.Buffer
		app := newApp(exec.ExternalCommand("tool"), &out)

		args, _ := cli.Split("app do")
		err := app.RunContext(context.Background(), args...)

		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(Equal("custom prefix\n"))
	})

	It("panics when more than one prefix is specified", func() {
		Expect(func() {
			exec.ExternalCommand("a", "b")
		}).To(Panic())
	})
})
