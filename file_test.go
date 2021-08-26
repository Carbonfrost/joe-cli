package cli_test

import (
	"os"
	"path/filepath"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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
})
