// Copyright 2025, 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"context"
	"encoding/json"
	"go/format"
	"io"
	"io/fs"
	"os"
	"os/exec"
	tt "text/template"
	"time"
)

type FileGenerator interface {
	GenerateFile(ctx context.Context, c *OutputContext, name string) error
}

type FileGeneratorFunc func(context.Context, *OutputContext, string) error

type FileMode int

type fileGenerator struct {
	name string
	ops  []FileGenerator
}

type generateFile func(context.Context, *OutputContext) ([]byte, error)

// File mode bits
const (
	Executable FileMode = 0755
	ReadOnly   FileMode = 0600
)

func newGenerateContents(f generateFile) FileGeneratorFunc {
	return func(ctx context.Context, c *OutputContext, name string) error {
		if f == nil {
			c.identical(name)
			return nil
		}

		data, err := f(ctx, c)
		if err != nil {
			return err
		}

		if c.DryRun {
			return nil
		}

		file, err := c.actualFS().Create(name)
		if err != nil {
			return err
		}

		_, err = file.(io.Writer).Write(data)
		return err
	}
}

// File generates a file with the given operations
func File(name string, ops ...FileGenerator) Generator {
	return &fileGenerator{
		name, ops,
	}
}

// MarshalWriter is used by ContentsMarshal to encode a value into file contents.
type MarshalWriter interface {
	MarshalWrite(w io.Writer, in any) error
}

// Contents generates a file with the given contents, which are copied to the
// output file of the given name.
func Contents(contents []byte) FileGenerator {
	return newGenerateContents(func(_ context.Context, _ *OutputContext) ([]byte, error) {
		return contents, nil
	})
}

// ContentsString generates a file whose contents are the given string.
func ContentsString(contents string) FileGenerator {
	return newGenerateContents(func(_ context.Context, _ *OutputContext) ([]byte, error) {
		return []byte(contents), nil
	})
}

// ContentsFrom generates a file whose contents are read from the given reader.
func ContentsFrom(r io.Reader) FileGenerator {
	return newGenerateContents(func(_ context.Context, _ *OutputContext) ([]byte, error) {
		return io.ReadAll(r)
	})
}

// ContentsJSON generates a file whose contents are the JSON encoding of the
// given value.
func ContentsJSON(v any) FileGenerator {
	return newGenerateContents(func(_ context.Context, _ *OutputContext) ([]byte, error) {
		return json.MarshalIndent(v, "", "    ")
	})
}

// ContentsMarshal generates a file whose contents are the encoding of the
// given value produced by the MarshalWriter.
func ContentsMarshal(v any, m MarshalWriter) FileGenerator {
	return newGenerateContents(func(_ context.Context, _ *OutputContext) ([]byte, error) {
		var buf bytes.Buffer
		if err := m.MarshalWrite(&buf, v); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

// Touch touches the file.
func Touch() FileGenerator {
	return FileGeneratorFunc(func(_ context.Context, c *OutputContext, name string) error {
		f := c.actualFS()
		fileName := name
		_, err := f.Stat(fileName)

		if os.IsNotExist(err) {
			file, err := f.Create(name)
			if err != nil {
				return err
			}
			defer file.Close()
			return nil
		}

		currentTime := time.Now().Local()
		return f.Chtimes(fileName, currentTime, currentTime)
	})
}

// Template generates a file by executing a template.
func Template(tt *tt.Template, namedata ...any) FileGenerator {
	return newGenerateContents(func(ctx context.Context, c *OutputContext) ([]byte, error) {
		err := Data(namedata...).Generate(ctx, c)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		err = tt.Execute(&buf, c.Vars)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func Gofmt() FileGenerator {
	return FileGeneratorFunc(func(ctx context.Context, c *OutputContext, name string) error {
		file, err := c.FS.OpenContext(ctx, name)
		if err != nil {
			return err
		}

		src, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		result, err := format.Source(src)
		if err != nil {
			return err
		}

		_, err = doFileGenerate(ctx, c, name, Contents(result))
		return err
	})
}

// Exec generates a file by piping the existing file contents through a subprocess.
// The subprocess receives the current file contents on stdin, and its stdout becomes
// the new file contents.
func Exec(name string, arg ...string) FileGenerator {
	return FileGeneratorFunc(func(ctx context.Context, c *OutputContext, filename string) error {
		var src []byte
		file, err := c.FS.OpenContext(ctx, filename)
		if err == nil {
			src, err = io.ReadAll(file)
			if err != nil {
				return err
			}
		}

		cmd := exec.CommandContext(ctx, name, arg...)
		cmd.Stdin = bytes.NewReader(src)

		result, err := cmd.Output()
		if err != nil {
			return err
		}

		_, err = doFileGenerate(ctx, c, filename, Contents(result))
		return err
	})
}

func Mode(mode fs.FileMode) FileGenerator {
	return FileGeneratorFunc(func(_ context.Context, c *OutputContext, name string) error {
		return c.Chmod(name, mode)
	})
}

func (f FileGeneratorFunc) GenerateFile(ctx context.Context, c *OutputContext, name string) error {
	if f == nil {
		return nil
	}
	return f(ctx, c, name)
}

func (m FileMode) GenerateFile(_ context.Context, c *OutputContext, name string) error {
	return c.Chmod(name, fs.FileMode(int(m)))
}

func (f *fileGenerator) Generate(ctx context.Context, c *OutputContext) error {
	if len(f.ops) == 0 {
		c.identical(f.name)
		return nil
	}

	file, err := c.OpenContext(ctx, f.name)
	created := os.IsNotExist(err)
	var original []byte
	if err == nil {
		original, _ = io.ReadAll(file)
	}

	if len(f.ops) == 0 {
		c.identical(f.name)
		return nil
	}

	fileName, err := doFileGenerate(ctx, c, f.name, f.ops...)
	if err != nil {
		return err
	}
	c.reportChange(original, fileName, created)
	return nil
}

func doFileGenerate(ctx context.Context, c *OutputContext, name string, ops ...FileGenerator) (fileName string, err error) {
	fileName, err = c.pathEnsure(name)
	if err != nil {
		return
	}
	for _, o := range ops {
		err = o.GenerateFile(ctx, c, fileName)
		if err != nil {
			break
		}
	}

	return
}

var (
	_ FileGenerator = FileMode(0)
)
