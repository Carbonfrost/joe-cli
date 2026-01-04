// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing/fstest"
	"time"

	cli "github.com/Carbonfrost/joe-cli"
	joeclifakes "github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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
		_ = app.RunContext(context.Background(), []string{"app", tmpFileLocation.Name()})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.File("f")).NotTo(BeNil())
		Expect(context.File("f").String()).To(Equal(tmpFileLocation.Name()))
		Expect(context.File("f").Exists()).To(BeTrue())
	})

	It("as a flag it will consume flag-like arguments", func() {
		file := new(cli.File)
		app := &cli.App{
			Flags: []*cli.Flag{
				{
					Name:  "f",
					Value: file,
				},
			},
		}
		err := app.RunContext(context.Background(), []string{"app", "-f", "--other"})
		Expect(err).NotTo(HaveOccurred())
		Expect(file.Name).To(Equal("--other"))
	})

	Describe("MustExist", func() {

		var (
			arguments string
			err       error
			value     any
		)

		JustBeforeEach(func() {
			app := &cli.App{
				Args: []*cli.Arg{
					{
						Name:    "f",
						Value:   value,
						Options: cli.MustExist,
					},
				},
			}

			args, _ := cli.Split(arguments)
			err = app.RunContext(context.Background(), args)
		})

		Context("when the file does not exist", func() {

			var name string

			BeforeEach(func() {
				SkipOnWindows()
				name = filepath.Join(os.TempDir(), "nonexistent")
				arguments = "app " + name
				value = new(cli.File)
			})

			It("returns an error if the file does not exist", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring(name + ": no such file or directory")))
			})
		})

		Context("when the file is not set", func() {
			BeforeEach(func() {
				arguments = "app"
				value = new(cli.File)
			})

			It("does nothing", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the file is set to blank", func() {
			BeforeEach(func() {
				arguments = "app -f''"
				value = new(cli.File)
			})

			It("does nothing", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the file is a string", func() {
			var name string

			BeforeEach(func() {
				SkipOnWindows()
				name = filepath.Join(os.TempDir(), "nonexistent")
				arguments = "app " + name
				value = new(string)
			})

			It("returns an error if the file does not exist", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring(name + ": no such file or directory")))
			})
		})

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
		_ = app.RunContext(context.Background(), []string{"app", "-"})
		context := cli.FromContext(act.ExecuteArgsForCall(0))

		f, err := context.File("f").Open()
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprint(f.(io.Writer), "hello")
		Expect(buf.String()).To(Equal("hello"))
	})

	Describe("FS", func() {

		var (
			timeA = time.Now()
			timeB = time.Now()
		)

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

			_ = app.RunContext(context.Background(), []string{"app"})
			Expect(actual.FS).To(BeIdenticalTo(globalFS))
		})

		DescribeTable("delegates examples", func(
			f func(*cli.File),
			argsForCall any, // should be one of the ArgsForCall methods
			expected types.GomegaMatcher) {

			globalFS := new(joeclifakes.FakeFS)
			app := &cli.App{
				Flags: []*cli.Flag{
					{
						Name:  "f",
						Value: &cli.File{},
					},
				},
				Action: func(c *cli.Context) {
					f(c.File("f"))
				},
				FS: globalFS,
			}

			_ = app.RunContext(context.Background(), []string{"app", "-f", "filename"})
			actual := make([]any, 0)
			callArgs := []reflect.Value{reflect.ValueOf(globalFS), reflect.ValueOf(0)}
			for _, a := range reflect.ValueOf(argsForCall).Call(callArgs) {
				actual = append(actual, a.Interface())
			}
			Expect(actual).To(expected)
		},
			Entry("Create",
				func(f *cli.File) { f.Create() },
				(*joeclifakes.FakeFS).CreateArgsForCall,
				Equal([]any{"filename"}),
			),
			Entry("OpenFile",
				func(f *cli.File) { f.OpenFile(2, 4) },
				(*joeclifakes.FakeFS).OpenFileArgsForCall,
				Equal([]any{"filename", 2, fs.FileMode(4)}),
			),
			Entry("Chmod",
				func(f *cli.File) { f.Chmod(4) },
				(*joeclifakes.FakeFS).ChmodArgsForCall,
				Equal([]any{"filename", fs.FileMode(4)}),
			),
			Entry("Chown",
				func(f *cli.File) { f.Chown(1, 1) },
				(*joeclifakes.FakeFS).ChownArgsForCall,
				Equal([]any{"filename", 1, 1}),
			),
			Entry("Chtimes",
				func(f *cli.File) { f.Chtimes(timeA, timeB) },
				(*joeclifakes.FakeFS).ChtimesArgsForCall,
				Equal([]any{"filename", timeA, timeB}),
			),
			Entry("Remove",
				func(f *cli.File) { f.Remove() },
				(*joeclifakes.FakeFS).RemoveArgsForCall,
				Equal([]any{"filename"}),
			),
			Entry("RemoveAll",
				func(f *cli.File) { f.RemoveAll() },
				(*joeclifakes.FakeFS).RemoveAllArgsForCall,
				Equal([]any{"filename"}),
			),
			Entry("Mkdir",
				func(f *cli.File) { f.Mkdir(0123) },
				(*joeclifakes.FakeFS).MkdirArgsForCall,
				Equal([]any{"filename", fs.FileMode(0123)}),
			),
			Entry("MkdirAll",
				func(f *cli.File) { f.MkdirAll(0123) },
				(*joeclifakes.FakeFS).MkdirAllArgsForCall,
				Equal([]any{"filename", fs.FileMode(0123)}),
			),
			Entry("Rename",
				func(f *cli.File) { f.Rename("newone") },
				(*joeclifakes.FakeFS).RenameArgsForCall,
				Equal([]any{"filename", "newone"}),
			),
		)

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
				actual, _ = io.ReadAll(f)
			},
		}
		_ = app.RunContext(context.Background(), []string{"app", "-"})
		Expect(string(actual)).To(Equal("hello\n"))
	})

	It("reads multiple times from stdin", func() {
		var actual []byte
		buf := bytes.NewBufferString("hello\n")
		app := cli.App{
			Args: []*cli.Arg{
				{
					Name:  "f",
					Value: &cli.File{},
				},
				{
					Name:  "g",
					Value: &cli.File{},
				},
			},
			Stdin: buf,
			Action: func(c *cli.Context) {
				f, _ := c.File("f").Open()
				_, _ = io.ReadAll(f)

				g, _ := c.File("g").Open()
				actual, _ = io.ReadAll(g)
			},
		}
		_ = app.RunContext(context.Background(), []string{"app", "-", "-"})
		Expect(string(actual)).To(Equal("hello\n"))
	})

	Describe("standard file", func() {

		It("uses stdin when opening read mode", func() {
			f := &cli.File{Name: "-"}
			actual, _ := f.OpenFile(os.O_RDONLY, 0777)
			Expect(actual).To(BeIdenticalTo(os.Stdin))
		})

		It("uses stdout when opening write mode", func() {
			f := &cli.File{Name: "-"}
			actual, _ := f.OpenFile(os.O_WRONLY, 0777)
			Expect(actual).To(BeIdenticalTo(os.Stdout))
		})

		It("uses stdout when creating", func() {
			f := &cli.File{Name: "-"}
			actual, err := f.Create()
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(BeIdenticalTo(os.Stdout))
		})

		It("error when trying to do read-write without APPEND", func() {
			f := &cli.File{Name: "-"}
			_, err := f.OpenFile(os.O_RDWR, 0777)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Ext", func() {
		It("obtains the file ext", func() {
			f := &cli.File{Name: "ok.pgp"}
			Expect(f.Ext()).To(Equal(".pgp"))
		})
	})

	Describe("Dir", func() {
		It("obtains the file directory", func() {
			f := &cli.File{Name: "secrets/ok.pgp"}
			Expect(f.Dir()).To(Equal("secrets"))
		})
	})
})

var _ = Describe("FileReference", func() {

	testFileSystem := fstest.MapFS{
		"d/b.bin":  {Data: []byte("facade")},
		"d/b.list": {Data: []byte("a\nb\nc\n")},
	}

	It("gets the contents of a file", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "b",
					Value:   cli.Bytes(),
					Options: cli.FileReference,
				},
			},
			FS:     testFileSystem,
			Action: act,
		}
		_ = app.RunContext(context.Background(), []string{"app", "d/b.bin"})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.Bytes("b")).NotTo(BeNil())
		Expect(context.Bytes("b")).To(Equal([]byte("facade")))
		Expect(context.Raw("b")).To(Equal([]string{"<b>", "d/b.bin"}))
	})

	It("gets the contents of a @file when allowed", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "b",
					Value:   cli.String(),
					Options: cli.AllowFileReference,
				},
				{
					Name:    "c",
					Value:   cli.String(),
					Options: cli.AllowFileReference,
				},
			},
			FS:     testFileSystem,
			Action: act,
		}
		_ = app.RunContext(context.Background(), []string{"app", "@d/b.bin", "d/b.bin"})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.String("b")).To(Equal("facade"))
		Expect(context.String("c")).To(Equal("d/b.bin"))
	})

	It("gets the contents of a multiple @files", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "b",
					Value:   cli.List(),
					Options: cli.AllowFileReference,
				},
			},
			FS:     testFileSystem,
			Action: act,
		}
		_ = app.RunContext(context.Background(), []string{"app", "@d/b.bin", "@d/b.bin"})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.List("b")).To(Equal([]string{"facade", "facade"}))
	})

	It("gets the contents of a multiple @files", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "b",
					Value:   cli.String(),
					NArg:    cli.TakeUntilNextFlag,
					Options: cli.AllowFileReference,
				},
			},
			FS:     testFileSystem,
			Action: act,
		}
		_ = app.RunContext(context.Background(), []string{"app", "@d/b.bin", "@d/b.bin"})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.String("b")).To(Equal("facade facade"))
	})

	It("gets lines into fileset", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "b",
					Value:   new(cli.FileSet),
					Options: cli.FileReference | cli.Merge,
				},
			},
			FS:     testFileSystem,
			Action: act,
		}
		err := app.RunContext(context.TODO(), []string{"app", "d/b.list", "d/b.list"})
		Expect(err).NotTo(HaveOccurred())

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.FileSet("b").Files).To(Equal([]string{"a", "b", "c", "a", "b", "c"}))
	})

	It("gets lines into fileset from stdin", func() {
		act := new(joeclifakes.FakeAction)
		app := &cli.App{
			Args: []*cli.Arg{
				{
					Name:    "b",
					Value:   new(cli.FileSet),
					Options: cli.FileReference,
				},
			},
			Action: act,
			Stdin:  strings.NewReader("ok.txt\n\n"),
		}
		_ = app.RunContext(context.TODO(), []string{"app", "-"})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.FileSet("b").Files).To(Equal([]string{"ok.txt"}))
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
		_ = app.RunContext(context.Background(), []string{"app", "fiche"})

		context := cli.FromContext(act.ExecuteArgsForCall(0))
		Expect(context.FileSet("f")).NotTo(BeNil())
		Expect(context.FileSet("f").String()).To(Equal("fiche"))
	})

	testFileSystem := fstest.MapFS{
		"src/a/b.txt": {Data: []byte("b")},
		"src/c.txt":   {Data: []byte("c")},
	}

	It("all files must exist", func() {
		set := &cli.FileSet{FS: testFileSystem, Files: []string{"src/a/b.txt"}}
		Expect(set.Exists()).To(BeTrue())

		set = &cli.FileSet{
			FS:    testFileSystem,
			Files: []string{"src/a/b.txt", "somethingelse"},
		}
		Expect(set.Exists()).To(BeFalse())
	})

	It("uses - for stdin/out", func() {
		var actual []byte
		buf := bytes.NewBufferString("hello\n")
		app := cli.App{
			Args: []*cli.Arg{
				{
					Name:  "f",
					Value: &cli.FileSet{},
				},
			},
			Stdin: buf,
			Action: func(c *cli.Context) {
				_ = c.FileSet("f").Do(func(f *cli.File, _ error) error {
					fs, _ := f.Open()
					actual, _ = io.ReadAll(fs)
					return nil
				})
			},
		}
		_ = app.RunContext(context.Background(), []string{"app", "--", "-"})
		Expect(string(actual)).To(Equal("hello\n"))
	})

	Describe("SetData", func() {

		DescribeTable("examples", func(data string, expected types.GomegaMatcher) {
			value := new(cli.FileSet)
			err := cli.SetData(value, bytes.NewReader([]byte(data)))
			Expect(err).NotTo(HaveOccurred())
			Expect(value.Files).To(expected)
		},
			Entry(
				"nominal",
				"a.txt\nb.txt",
				Equal([]string{"a.txt", "b.txt"}),
			),
			Entry(
				"comments and blank lines",
				"# comment\n; comment\n\n\n\ns.txt",
				Equal([]string{"s.txt"}),
			),
			Entry(
				"trim leading whitespace",
				"    a.txt",
				Equal([]string{"a.txt"}),
			),
			Entry(
				"trim trailing whitespace",
				"a.txt  ",
				Equal([]string{"a.txt"}),
			),
		)
	})

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
					text, _ := io.ReadAll(r)
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

	Describe("All", func() {

		DescribeTable("examples",
			func(set *cli.FileSet, expected []string) {
				var all []string
				for f := range set.All() {
					all = append(all, f.Name)
				}

				Expect(all).To(Equal(expected))
			},

			Entry("process file",
				&cli.FileSet{
					FS:    testFileSystem,
					Files: []string{"src/a/b.txt"},
				},
				[]string{"src/a/b.txt"}),
			Entry("process files recursive",
				&cli.FileSet{
					FS:        testFileSystem,
					Recursive: true,
					Files:     []string{"src"},
				}, []string{"src", "src/a", "src/a/b.txt", "src/c.txt"}),
		)
	})
})
