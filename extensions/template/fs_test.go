// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template_test

import (
	"context"
	"io"
	"io/fs"
	"testing/fstest"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var _ = Describe("FS", func() {

	Describe("Generate", func() {

		sourceFS := fstest.MapFS{
			"hello.txt":        &fstest.MapFile{Data: []byte("hello")},
			"subdir/world.txt": &fstest.MapFile{Data: []byte("world")},
			"main.go":          &fstest.MapFile{Data: []byte("package   m\ntype C    struct {  }")},
		}

		newDest := func() cli.FS {
			return wrapperFS{afero.NewMemMapFs()}
		}

		It("copies files as-is by default", func() {
			dest := newDest()
			app := &cli.App{
				Name:   "app",
				FS:     dest,
				Action: template.New(template.FS(sourceFS)),
				Stdout: io.Discard,
			}

			Expect(app.RunContext(context.Background(), []string{"app"})).To(Succeed())

			actual, err := fs.ReadFile(dest, "hello.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal("hello"))
		})

		It("copies nested files", func() {
			SkipOnWindows()

			dest := newDest()
			app := &cli.App{
				Name:   "app",
				FS:     dest,
				Action: template.New(template.FS(sourceFS)),
				Stdout: io.Discard,
			}

			Expect(app.RunContext(context.Background(), []string{"app"})).To(Succeed())

			actual, err := fs.ReadFile(dest, "subdir/world.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal("world"))
		})

		It("applies file generators for matching glob", func() {
			dest := newDest()
			app := &cli.App{
				Name: "app",
				FS:   dest,
				Action: template.New(
					template.FS(sourceFS,
						template.WithFileGenerator("*.go",
							template.OriginalContents(),
							template.Gofmt(),
						),
					),
				),
				Stdout: io.Discard,
			}

			Expect(app.RunContext(context.Background(), []string{"app"})).To(Succeed())

			actual, err := fs.ReadFile(dest, "main.go")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal("package m\n\ntype C struct{}\n"))
		})

		It("does not apply file generators for non-matching files", func() {
			dest := newDest()
			app := &cli.App{
				Name: "app",
				FS:   dest,
				Action: template.New(
					template.FS(sourceFS,
						template.WithFileGenerator("*.go",
							template.Contents("overridden"),
						),
					),
				),
				Stdout: io.Discard,
			}

			Expect(app.RunContext(context.Background(), []string{"app"})).To(Succeed())

			actual, err := fs.ReadFile(dest, "hello.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal("hello"))
		})
	})
})
