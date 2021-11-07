package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) { return nil, nil }

var _ = Describe("File", func() {

	It("as an argument can be retrieved", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:  "f",
					Value: &cli.File{},
				},
			},
			Action: act,
		}
		tmpFileLocation, _ := os.CreateTemp("", "example.*.txt")
		_ = app.RunContext(context.TODO(), []string{"app", tmpFileLocation.Name()})

		context := act.ExecuteArgsForCall(0)
		Expect(context.File("f")).NotTo(BeNil())
		Expect(context.File("f").String()).To(Equal(tmpFileLocation.Name()))
		Expect(context.File("f").Exists()).To(BeTrue())
	})

	It("returns an error if the file does not exist", func() {
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "f",
					Value:   &cli.File{},
					Options: cli.MustExist,
				},
			},
		}
		name := filepath.Join(os.TempDir(), "nonexistent")
		err := app.RunContext(context.TODO(), []string{"app", name})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(name + ": no such file or directory"))
	})

	It("writes to the app output when - is used", func() {
		var buf bytes.Buffer
		act := new(joeclifakes.FakeAction)
		app := cli.App{
			Args: []*cli.Arg{
				{
					Name:  "f",
					Value: &cli.File{},
				},
			},
			Stdout: &buf,
			Action: act,
		}
		_ = app.RunContext(context.TODO(), []string{"app", "-"})
		context := act.ExecuteArgsForCall(0)

		f, err := context.File("f").Open()
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprint(f.(io.Writer), "hello")
		Expect(buf.String()).To(Equal("hello"))
	})

	It("sets FS from app", func() {
		var actual *cli.File
		globalFS := emptyFS{}
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "f",
					Value: &cli.File{},
				},
			},
			Action: func(c *cli.Context) {
				actual = c.File("f")
			},
			FS: globalFS,
		}

		_ = app.RunContext(context.TODO(), []string{"app"})
		Expect(actual.FS).To(BeIdenticalTo(globalFS))
	})

	It("reads from the app input when - is used", func() {
		var actual []byte
		buf := bytes.NewBufferString("hello\n")
		app := cli.App{
			Args: []*cli.Arg{
				{
					Name:  "f",
					Value: &cli.File{},
				},
			},
			Stdin: buf,
			Action: func(c *cli.Context) {
				f, _ := c.File("f").Open()
				actual, _ = ioutil.ReadAll(f)
			},
		}
		_ = app.RunContext(context.TODO(), []string{"app", "-"})
		Expect(string(actual)).To(Equal("hello\n"))
	})
})

var _ = Describe("FileSet", func() {

	It("as an argument can be retrieved", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:  "f",
					Value: &cli.FileSet{},
				},
			},
			Action: act,
		}
		_ = app.RunContext(context.TODO(), []string{"app", "fiche"})

		context := act.ExecuteArgsForCall(0)
		Expect(context.FileSet("f")).NotTo(BeNil())
		Expect(context.FileSet("f").String()).To(Equal("fiche"))
	})

	var testFileSystem = func() fs.FS {
		appFS := afero.NewMemMapFs()

		appFS.MkdirAll("src/a", 0755)
		afero.WriteFile(appFS, "src/a/b.txt", []byte("b"), 0644)
		afero.WriteFile(appFS, "src/c.txt", []byte("c"), 0644)

		return afero.NewIOFS(appFS)
	}()

	Describe("Do", func() {

		DescribeTable("examples",
			func(set *cli.FileSet, expected string) {
				var buf bytes.Buffer
				set.Do(func(d *cli.File, _ error) error {
					stat, _ := d.Stat()
					if stat.IsDir() {
						return nil
					}
					r, _ := d.Open()
					text, _ := ioutil.ReadAll(r)
					buf.Write(text)
					return nil
				})
				Expect(buf.String()).To(Equal(expected))
			},

			Entry("process file",
				&cli.FileSet{FS: testFileSystem, Files: []string{"src/a/b.txt"}}, "b"),
			Entry("process files recursive",
				&cli.FileSet{FS: testFileSystem, Recursive: true, Files: []string{"src"}}, "bc"),
		)
	})
})
