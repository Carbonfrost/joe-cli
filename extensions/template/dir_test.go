// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template_test

import (
	"context"
	"io"
	"io/fs"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var _ = Describe("Dir", func() {

	newDest := func() cli.FS {
		return wrapperFS{afero.NewMemMapFs()}
	}

	newApp := func(tpl *template.Root, dest cli.FS) *cli.App {
		return &cli.App{
			FS:     dest,
			Action: tpl,
			Stdout: io.Discard,
		}
	}

	It("doesn't create directories when MakeDirs is false", func() {
		dest := newDest()
		tpl := template.New(template.Dir("classified"))
		tpl.MakeDirs = false

		Expect(newApp(tpl, dest).RunContext(context.Background(), []string{"app"})).To(Succeed())

		_, err := fs.ReadDir(dest, "classified")

		Expect(err).To(MatchError(ContainSubstring("classified: file does not exist")))
	})

	It("creates directories manually with FileMode", func() {
		dest := newDest()
		tpl := template.New(template.Dir("classified", template.FileMode(0755)))
		tpl.MakeDirs = false
		app := &cli.App{
			FS:     dest,
			Action: tpl,
			Stdout: io.Discard,
		}

		Expect(app.RunContext(context.Background(), []string{"app"})).To(Succeed())

		actual, err := fs.ReadDir(dest, "classified")
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(BeEmpty())
	})

	It("creates directories by default", func() {
		dest := newDest()
		app := &cli.App{
			FS:     dest,
			Action: template.New(template.Dir("classified")),
			Stdout: io.Discard,
		}

		Expect(app.RunContext(context.Background(), []string{"app"})).To(Succeed())

		_, err := fs.ReadDir(dest, "classified")
		Expect(err).NotTo(HaveOccurred())
	})

})
