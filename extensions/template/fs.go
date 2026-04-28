// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"context"
	"io"
	"io/fs"
	"path"
)

// Option configures the behavior of the FS generator.
type Option func(*fsGenerator)

type fsGenerator struct {
	source         fs.FS
	fileGenerators []*fileGenOption
}

type fileGenOption struct {
	pattern string
	gens    []FileGenerator
}

type originalContentsGen struct{}

// FS creates a generator that walks the given file system and reproduces its
// files and directories in the output.  By default each file is copied as-is.
// Options may override this behavior for files whose names match a glob pattern.
func FS(fsys fs.FS, opts ...Option) Generator {
	g := &fsGenerator{source: fsys}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// WithFileGenerator overrides the default copy behavior for files whose name
// matches the given glob pattern.  Any OriginalContents generator in gen is
// replaced with the actual file contents from the source file system.
func WithFileGenerator(filename string, gen ...FileGenerator) Option {
	return func(g *fsGenerator) {
		g.fileGenerators = append(g.fileGenerators, &fileGenOption{filename, gen})
	}
}

// OriginalContents is a FileGenerator sentinel.  When used inside
// WithFileGenerator, it is replaced with the file's contents from the source
// file system, allowing other generators in the chain to operate on them.
// For example:
//
//	FS(fsys, WithFileGenerator("*.go", OriginalContents(), Gofmt()))
func OriginalContents() FileGenerator {
	return originalContentsGen{}
}

func (originalContentsGen) GenerateFile(_ context.Context, _ *OutputContext, _ string) error {
	return nil
}

func (g *fsGenerator) Generate(ctx context.Context, c *OutputContext) error {
	return fs.WalkDir(g.source, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filePath == "." || d.IsDir() {
			return nil
		}
		return File(filePath, g.buildGenerators(filePath)...).Generate(ctx, c)
	})
}

func (g *fsGenerator) buildGenerators(filePath string) []FileGenerator {
	base := path.Base(filePath)
	for _, opt := range g.fileGenerators {
		matched, err := path.Match(opt.pattern, base)
		if err == nil && matched {
			result := make([]FileGenerator, len(opt.gens))
			for i, gen := range opt.gens {
				if _, ok := gen.(originalContentsGen); ok {
					result[i] = g.readFromSource(filePath)
				} else {
					result[i] = gen
				}
			}
			return result
		}
	}
	return []FileGenerator{g.readFromSource(filePath)}
}

func (g *fsGenerator) readFromSource(filePath string) FileGenerator {
	return newGenerateContents(func(_ context.Context, _ *OutputContext) ([]byte, error) {
		f, err := g.source.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	})
}

var (
	_ Generator     = (*fsGenerator)(nil)
	_ FileGenerator = originalContentsGen{}
)
