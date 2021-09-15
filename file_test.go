package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) { return nil, nil }

var _ = Describe("File", func() {

	It("as an argument can be retrieved", func() {
		act := new(joeclifakes.FakeActionHandler)
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
		_ = app.RunContext(nil, []string{"app", tmpFileLocation.Name()})

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
		err := app.RunContext(nil, []string{"app", name})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(name + ": no such file or directory"))
	})

	It("writes to the app output when - is used", func() {
		var buf bytes.Buffer
		act := new(joeclifakes.FakeActionHandler)
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
		_ = app.RunContext(nil, []string{"app", "-"})
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

		_ = app.RunContext(nil, []string{"app"})
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
		_ = app.RunContext(nil, []string{"app", "-"})
		Expect(string(actual)).To(Equal("hello\n"))
	})
})
