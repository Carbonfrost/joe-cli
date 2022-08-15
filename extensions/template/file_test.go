package template_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"runtime"
	tt "text/template"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/template"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/spf13/afero"
)

var _ = Describe("File", func() {

	Describe("Generate", func() {

		var testFileSystem = func() cli.FS {
			appFS := afero.NewMemMapFs()
			return wrapperFS{appFS}
		}

		DescribeTable("examples", func(initial string, gen template.FileGenerator, expected types.GomegaMatcher) {
			testFileSystem := testFileSystem()
			app := &cli.App{
				Name: "app",
				FS:   testFileSystem,
				Action: template.New(
					template.Vars{"file": "file_generate_test.txt"},
					template.Data("Var", "variable"),
					template.File("{{ .file }}", gen),
				),
				Stdout: io.Discard,
			}

			f, _ := testFileSystem.Create("file_generate_test.txt")
			f.(io.Writer).Write([]byte(initial))
			f.Close()

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			actual, _ := fs.ReadFile(testFileSystem, "file_generate_test.txt")
			Expect(string(actual)).To(expected)
		},
			Entry("write string",
				"",
				template.Contents("string"),
				Equal("string"),
			),
			Entry("write bytes",
				"",
				template.Contents([]byte("bytes")),
				Equal("bytes"),
			),
			Entry("write reader",
				"",
				template.Contents(bytes.NewBuffer([]byte("reader contents"))),
				Equal("reader contents"),
			),
			Entry("JSON",
				"",
				template.Contents(struct {
					F string
					L string
				}{F: "O", L: "D"}),
				MatchJSON(`{"F": "O", "L": "D"}`),
			),
			Entry("Gofmt",
				`package   m
type C    struct {  }`,
				template.Gofmt(),
				Equal("package m\n\ntype C struct{}\n"),
			),
			Entry("template vars",
				"",
				template.Template(tt.Must(tt.New("").Parse("my {{ .Var }}"))),
				Equal("my variable"),
			),
			Entry("template inherits vars",
				"",
				template.Template(tt.Must(tt.New("").Parse("my {{ .file }}"))),
				Equal("my file_generate_test.txt"),
			),
		)

		DescribeTable("mode examples", func(gen template.FileMode, expected types.GomegaMatcher) {
			testFileSystem := testFileSystem()
			app := &cli.App{
				Name:   "app",
				FS:     testFileSystem,
				Action: template.New(template.File("mode.txt", gen)),
				Stdout: io.Discard,
			}

			_, _ = testFileSystem.Create("mode.txt")

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			f, _ := testFileSystem.Stat("mode.txt")
			actual := f.Mode().Perm()
			Expect(actual).To(expected)
		},
			Entry("readonly", template.ReadOnly, Equal(fs.FileMode(0600))),
			Entry("executable", template.Executable, Equal(fs.FileMode(0755))),
		)

		It("expands variables", func() {
			testFileSystem := testFileSystem()
			app := &cli.App{
				Name: "app",
				FS:   testFileSystem,
				Action: template.New(
					template.Vars{"App": map[string]any{"Name": "o"}},
					template.File("{{ .App.Name }}.txt", template.Contents("OK")),
				),
				Stdout: io.Discard,
			}

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			actual, _ := fs.ReadFile(testFileSystem, "o.txt")
			Expect(string(actual)).To(Equal("OK"))
		})

		It("ensures directory hierarchy", func() {
			SkipOnWindows()

			test := testFileSystem()
			ff := new(joeclifakes.FakeFS)
			ff.CreateStub = test.Create
			ff.OpenStub = test.Open
			ff.StatStub = test.Stat
			app := &cli.App{
				Name:   "app",
				FS:     ff,
				Action: template.New(template.File("a/b/c.txt", template.Contents("OK"))),
				Stdout: io.Discard,
			}

			err := app.RunContext(context.TODO(), []string{"app"})
			Expect(err).NotTo(HaveOccurred())

			path, _ := ff.MkdirAllArgsForCall(0)
			Expect(path).To(Equal("a/b"))
		})

		Context("when checking status messages", func() {

			var testFileSystem = func() cli.FS {
				appFS := afero.NewMemMapFs()
				f, _ := appFS.Create("existing.txt")
				f.(io.StringWriter).WriteString("existing contents")
				return wrapperFS{appFS}
			}

			It("reports identical on no generators", func() {
				app := &cli.App{
					Name:   "app",
					FS:     testFileSystem(),
					Action: template.New(template.File("file_generate_test.txt")),
				}

				res := renderScreen(app, "app")
				Expect(res).To(Equal("   identical  file_generate_test.txt\n"))
			})

			It("reports identical on generate same", func() {
				app := &cli.App{
					Name:   "app",
					FS:     testFileSystem(),
					Action: template.New(template.File("existing.txt", template.Contents("existing contents"))),
				}

				res := renderScreen(app, "app")
				Expect(res).To(Equal("   identical  existing.txt\n"))
			})

			It("reports overwrite on generate different text", func() {
				app := &cli.App{
					Name:   "app",
					FS:     testFileSystem(),
					Action: template.New(template.File("existing.txt", template.Contents("difference"))),
				}

				res := renderScreen(app, "app")
				Expect(res).To(Equal("   overwrite  existing.txt\n"))
			})

			It("reports create on new file", func() {
				app := &cli.App{
					Name:   "app",
					FS:     testFileSystem(),
					Action: template.New(template.File("new.txt", template.Contents("new"))),
				}

				res := renderScreen(app, "app")
				Expect(res).To(Equal("      create  new.txt\n"))
			})

			It("reports nested file", func() {
				SkipOnWindows()

				app := &cli.App{
					Name:   "app",
					FS:     testFileSystem(),
					Action: template.New(template.Dir("bin", template.File("new.txt", template.Contents("new")))),
				}

				res := renderScreen(app, "app")
				Expect(res).To(Equal("      create  bin/new.txt\n"))
			})

		})
	})
})

type wrapperFS struct {
	afero.Fs
}

func (w wrapperFS) Create(name string) (fs.File, error) {
	return w.Fs.Create(name)
}

func (w wrapperFS) Open(name string) (fs.File, error) {
	return w.Fs.Open(name)
}

func (w wrapperFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return w.Fs.OpenFile(name, flag, perm)
}

func disableConsoleColor() func() {
	os.Setenv("TERM", "dumb")
	return func() {
		os.Setenv("TERM", "0")
	}
}

func renderScreen(app *cli.App, args string) string {
	defer disableConsoleColor()()

	arguments, _ := cli.Split(args)
	var buffer bytes.Buffer
	app.Stderr = &buffer
	app.Stdout = &buffer
	_ = app.RunContext(context.TODO(), arguments)
	return buffer.String()
}

func SkipOnWindows() {
	if runtime.GOOS == "windows" {
		Skip("not tested on Windows")
	}
}
