// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	eexec "os/exec"
	"time"

	"github.com/Carbonfrost/joe-cli"
)

// Editor is a pipeline-enabled value that opens a text editor using
// OS-specific conventions. The VISUAL environment variable is consulted first,
// then EDITOR, then an OS-specific fallback.
//
// The fields correspond to those of testing/fstest.MapFile (excluding Sys) and
// are used to initialize the temporary file the editor opens. Data accepts
// io.Reader, string, or []byte; the zero value produces an empty file.
type Editor struct {
	// Data provides the initial content of the temporary file.
	// Accepted types: io.Reader, string, or []byte.
	Data any

	// Mode sets the file permissions on the temporary file.
	Mode fs.FileMode

	// ModTime, when non-zero, sets the modification time of the temporary file.
	ModTime time.Time

	// Cmd, when non-nil, is called with the *exec.Cmd created via
	// exec.CommandContext before the editor is started, allowing further
	// customization (environment variables, extra arguments, etc.).
	Cmd func(*eexec.Cmd)

	// KeepTempFile when set to true, causes the temporary file not to be deleted
	// after the
	KeepTempFile bool

	// Output receives the output of the editor
	Output []byte
}

// Execute implements cli.Action. It creates a temporary file, writes Data into
// it (if set), applies Mode and ModTime, then invokes the system editor and
// waits for it to exit.
func (e *Editor) Execute(ctx context.Context) error {
	tmp, err := os.CreateTemp("", "joe-*")
	if err != nil {
		return err
	}

	defer func() {
		if !e.KeepTempFile {
			os.Remove(tmp.Name())
		}
	}()

	if e.Data != nil {
		if err := writeEditorData(tmp, e.Data); err != nil {
			tmp.Close()
			return err
		}
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if e.Mode != 0 {
		if err := os.Chmod(tmp.Name(), e.Mode); err != nil {
			return err
		}
	}

	if !e.ModTime.IsZero() {
		if err := os.Chtimes(tmp.Name(), e.ModTime, e.ModTime); err != nil {
			return err
		}
	}

	editor := findEditor()
	cmd := eexec.CommandContext(ctx, editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if e.Cmd != nil {
		e.Cmd(cmd)
	}

	err = cmd.Run()
	if err != nil {
		return err
	}
	e.Output, err = os.ReadFile(tmp.Name())
	return err
}

// FindEditor gets the command used for the editor.
func FindEditor() string {
	return findEditor()
}

func writeEditorData(w io.Writer, data any) error {
	switch v := data.(type) {
	case io.Reader:
		_, err := io.Copy(w, v)
		return err
	case string:
		_, err := io.WriteString(w, v)
		return err
	case []byte:
		_, err := bytes.NewReader(v).WriteTo(w)
		return err
	default:
		return fmt.Errorf("editor: unsupported Data type: %T", data)
	}
}

var _ cli.Action = (*Editor)(nil)
