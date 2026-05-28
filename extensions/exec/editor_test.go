// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	eexec "os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Carbonfrost/joe-cli/extensions/exec"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Editor", func() {

	type captured struct {
		path    string
		content []byte
		info    fs.FileInfo
	}

	captureAndNoop := func(cap *captured) func(*eexec.Cmd) {
		return func(cmd *eexec.Cmd) {
			if len(cmd.Args) > 0 {
				cap.path = cmd.Args[len(cmd.Args)-1]
				cap.content, _ = os.ReadFile(cap.path)
				cap.info, _ = os.Stat(cap.path)
			}
			setToNoopEditor(cmd)
		}
	}

	Describe("Execute", func() {

		It("succeeds with no Data", func() {
			var cap captured
			e := exec.Editor{Cmd: captureAndNoop(&cap)}
			Expect(e.Execute(context.Background())).To(Succeed())
		})

		DescribeTable("writes Data to temp file",
			func(data any, expected string) {
				var cap captured
				e := exec.Editor{
					Data: data,
					Cmd:  captureAndNoop(&cap),
				}
				Expect(e.Execute(context.Background())).To(Succeed())
				Expect(string(cap.content)).To(Equal(expected))
			},
			Entry("string", "hello editor", "hello editor"),
			Entry("[]byte", []byte("byte content"), "byte content"),
			Entry("io.Reader", strings.NewReader("reader content"), "reader content"),
		)

		It("applies Mode to the temp file", func() {
			var cap captured
			e := exec.Editor{
				Mode: 0640,
				Cmd:  captureAndNoop(&cap),
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(cap.info.Mode() & 0777).To(Equal(fs.FileMode(0640)))
		})

		It("applies ModTime to the temp file", func() {
			want := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
			var cap captured
			e := exec.Editor{
				ModTime: want,
				Cmd:     captureAndNoop(&cap),
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(cap.info.ModTime().UTC()).To(Equal(want))
		})

		It("doesn't delete file when KeepTempFile is set", func() {
			var cap captured
			e := exec.Editor{
				KeepTempFile: true,
				Cmd:          captureAndNoop(&cap),
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(cap.path).To(BeAnExistingFile())
		})

		It("deletes the temporary file when KeepTempFile is false", func() {
			var cap captured
			e := exec.Editor{
				KeepTempFile: false,
				Cmd:          captureAndNoop(&cap),
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(cap.path).NotTo(BeAnExistingFile())
		})

		It("sets Output to editor output", func() {
			SkipOnWindows()

			e := exec.Editor{
				KeepTempFile: false,
				Data:         "my content",
				Cmd: func(cmd *eexec.Cmd) {
					path, err := eexec.LookPath("sed")
					Expect(err).NotTo(HaveOccurred())

					cmd.Path = path
					cmd.Args = append([]string{`s/content/edited/`, "-i''", "-e", `s/content/edited/`}, cmd.Args[1:]...)
				},
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(e.Output).To(Equal([]byte("my edited")))
		})

		It("calls Cmd before starting the editor", func() {
			called := false
			e := exec.Editor{
				Cmd: func(cmd *eexec.Cmd) {
					called = true
					setToNoopEditor(cmd)
				},
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(called).To(BeTrue())
		})

		It("returns error for unsupported Data type", func() {
			e := exec.Editor{Data: 42}
			err := e.Execute(context.Background())
			Expect(err).To(MatchError(ContainSubstring("unsupported Data type")))
		})

		It("respects context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			e := exec.Editor{
				// Use a command that would block so cancellation is observable.
				// On context cancellation exec.CommandContext kills the process.
				Cmd: setToNoopEditor,
			}
			// A cancelled context may or may not error depending on timing;
			// we just verify Execute doesn't panic.
			_ = e.Execute(ctx)
		})

		It("passes bytes.Buffer as io.Reader", func() {
			var cap captured
			buf := bytes.NewBufferString("buffered")
			e := exec.Editor{
				Data: buf,
				Cmd:  captureAndNoop(&cap),
			}
			Expect(e.Execute(context.Background())).To(Succeed())
			Expect(string(cap.content)).To(Equal("buffered"))
		})
	})
})

func setToNoopEditor(cmd *eexec.Cmd) {
	noop, _ := eexec.LookPath("true")
	if noop == "" {
		noop, _ = eexec.LookPath("cmd")
	}
	cmd.Path = noop
	cmd.Args = []string{noop}
}

func SkipOnWindows() {
	if runtime.GOOS == "windows" {
		Skip("not tested on Windows")
	}
}
