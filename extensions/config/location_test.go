// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing/fstest"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Layer", func() {

	Describe("String", func() {
		DescribeTable("examples", func(l config.Layer, expected string) {
			Expect(l.String()).To(Equal(expected))
		},
			Entry("LayerUnspecified", config.LayerUnspecified, "UNSPECIFIED"),
			Entry("LayerIntrinsic", config.LayerIntrinsic, "INTRINSIC"),
			Entry("LayerSystem", config.LayerSystem, "SYSTEM"),
			Entry("LayerUser", config.LayerUser, "USER"),
			Entry("LayerWorkspace", config.LayerWorkspace, "WORKSPACE"),
			Entry("LayerProfile", config.LayerProfile, "PROFILE"),
			Entry("LayerAdditional", config.LayerAdditional, "ADDITIONAL"),
			Entry("in between", config.LayerSystem+1, "SYSTEM+1"),
			Entry("in between 2", config.LayerProfile+1, "PROFILE+1"),
			Entry("over bounds", config.Layer(11), "ADDITIONAL"),
			Entry("under bounds", config.Layer(-2), "UNSPECIFIED"),
		)

	})

})

var _ = Describe("ParseLocation", func() {

	Describe("file location", func() {

		BeforeEach(func() {
			GinkgoT().Setenv("TEST_VAR", "/expanded")
		})

		DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
			loc := config.ParseLocation(text)
			paths, _ := loc.Paths(context.Background())
			Expect(paths).To(expected)
		},
			Entry("simple", "/tmp/test.txt", Equal([]string{"/tmp/test.txt"})),
			Entry("environment variables", "$TEST_VAR/file.txt", Equal([]string{"/expanded/file.txt"})),
			Entry("empty when variable cannot be resolved", "$NONEXISTENT_VAR/file.txt", BeEmpty()),
		)
	})

	Describe("directory location", func() {
		var testDir string

		BeforeEach(func() {
			testDir = GinkgoT().TempDir()

			os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("data"), 0644)
			os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("data"), 0644)
			os.Mkdir(filepath.Join(testDir, "subdir"), 0755)
			os.WriteFile(filepath.Join(testDir, "subdir", "nested.txt"), []byte("nested"), 0644)
		})

		It("returns files in directory only (non-recursive)", func() {
			loc := config.ParseLocation(testDir + "/")
			paths, err := loc.Paths(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(2))
			Expect(paths).To(ContainElement(filepath.Join(testDir, "file1.txt")))
			Expect(paths).To(ContainElement(filepath.Join(testDir, "file2.txt")))

			// Should not include nested file
			Expect(paths).NotTo(ContainElement(filepath.Join(testDir, "subdir", "nested.txt")))
		})
	})

	Describe("directory tree location", func() {
		var testDir string

		BeforeEach(func() {
			testDir = GinkgoT().TempDir()

			os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("data"), 0644)
			os.Mkdir(filepath.Join(testDir, "subdir"), 0755)
			os.WriteFile(filepath.Join(testDir, "subdir", "nested.txt"), []byte("nested"), 0644)
		})

		It("returns all files recursively", func() {
			loc := config.ParseLocation(testDir + "/...")
			paths, err := loc.Paths(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(2))
			Expect(paths).To(ContainElement(filepath.Join(testDir, "file1.txt")))
			Expect(paths).To(ContainElement(filepath.Join(testDir, "subdir", "nested.txt")))
		})
	})

	Describe("special variable expansion", func() {

		DescribeTable("runtime variables", func(varName, expectedValue string) {
			loc := config.ParseLocation("$" + varName + "/file.txt")
			paths, err := loc.Paths(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal(expectedValue + "/file.txt"))
		},
			Entry("GOOS", "GOOS", runtime.GOOS),
			Entry("GOARCH", "GOARCH", runtime.GOARCH),
		)

		It("expands cli:app in app context", func() {
			app := &cli.App{
				Name: "testapp",
			}
			ctx, err := app.Initialize(context.Background())
			Expect(err).NotTo(HaveOccurred())

			loc := config.ParseLocation("${cli:app}/file.txt")
			paths, err := loc.Paths(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal("testapp/file.txt"))
		})

		It("expands cli:workspace in workspace context", func() {
			testFS := fstest.MapFS{
				"workspace/.testapp": {Mode: fs.ModeDir},
			}
			app := &cli.App{
				Name: "testapp",
				Uses: config.NewWorkspace(
					config.WithFS(testFS),
					config.WithFinder(testWorkspaceFinder("workspace")),
				),
				Action: func(c context.Context) {
					loc := config.ParseLocation("${cli:workspace}/config.json")
					paths, err := loc.Paths(c)

					Expect(err).NotTo(HaveOccurred())
					Expect(paths).To(HaveLen(1))
					Expect(paths[0]).To(Equal("workspace/config.json"))
				},
			}

			err := app.RunContext(context.Background(), []string{"testapp"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("expands cli:workspace.config in workspace context", func() {
			testFS := fstest.MapFS{
				"workspace/.testapp": {Mode: fs.ModeDir},
			}
			app := &cli.App{
				Name: "testapp",
				Uses: config.NewWorkspace(
					config.WithFS(testFS),
					config.WithFinder(testWorkspaceFinder("workspace")),
					config.WithConfigDir(".testapp"),
				),
				Action: func(c context.Context) {
					loc := config.ParseLocation("${cli:workspace.config}/settings.json")
					paths, _ := loc.Paths(c)

					Expect(paths).To(Equal([]string{".testapp/settings.json"}))
				},
			}

			err := app.RunContext(context.Background(), []string{"testapp"})
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("workspace variables without context", func(varName string) {
			loc := config.ParseLocation("${" + varName + "}/file.txt")
			paths, err := loc.Paths(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(BeEmpty())
		},
			Entry("cli:workspace", "cli:workspace"),
			Entry("cli:workspace.config", "cli:workspace.config"),
		)
	})

	Describe("complex path expansion", func() {
		It("expands multiple variables in single path", func() {
			os.Setenv("BASE_DIR", "/base")
			defer os.Unsetenv("BASE_DIR")

			loc := config.ParseLocation("$BASE_DIR/$GOOS/$GOARCH/file.txt")
			paths, err := loc.Paths(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(1))
			expected := filepath.Join("/base", runtime.GOOS, runtime.GOARCH, "file.txt")
			Expect(paths[0]).To(Equal(expected))
		})

		It("returns empty if any variable is unresolved", func() {
			os.Setenv("GOOD_VAR", "/good")
			defer os.Unsetenv("GOOD_VAR")

			loc := config.ParseLocation("$GOOD_VAR/$BAD_VAR/file.txt")
			paths, err := loc.Paths(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(BeEmpty())
		})
	})
})

type testWorkspaceFinder string

func (t testWorkspaceFinder) FindWorkspacePath(_ context.Context, _ fs.FS, _ string) (string, error) {
	return string(t), nil
}
