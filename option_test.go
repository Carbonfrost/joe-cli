package cli_test

import (
	"context"
	"encoding/json"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Option", func() {

	Describe("MarshalJSON", func() {

		DescribeTable("examples", func(opt cli.Option, expected string) {
			actual, _ := json.Marshal(opt)
			Expect(string(actual)).To(Equal("\"" + expected + "\""))

			var o cli.Option
			_ = json.Unmarshal(actual, &o)
			Expect(o).To(Equal(opt))
		},
			Entry("DisallowFlagsAfterArgs", cli.DisallowFlagsAfterArgs, "DISALLOW_FLAGS_AFTER_ARGS"),
			Entry("Exits", cli.Exits, "EXITS"),
			Entry("Hidden", cli.Hidden, "HIDDEN"),
			Entry("MustExist", cli.MustExist, "MUST_EXIST"),
			Entry("No", cli.No, "NO"),
			Entry("Optional", cli.Optional, "OPTIONAL"),
			Entry("Required", cli.Required, "REQUIRED"),
			Entry("SkipFlagParsing", cli.SkipFlagParsing, "SKIP_FLAG_PARSING"),
			Entry("WorkingDirectory", cli.WorkingDirectory, "WORKING_DIRECTORY"),
			Entry("NonPersistent", cli.NonPersistent, "NON_PERSISTENT"),
			Entry("DisableSplitting", cli.DisableSplitting, "DISABLE_SPLITTING"),
			Entry("compound", cli.No|cli.Hidden, "HIDDEN, NO"),
		)
	})

	Describe("NewOption", func() {

		It("allocates two options", func() {
			opt1 := cli.NewOption("OPTION_1", nil)
			opt2 := cli.NewOption("OPTION_2", nil)

			Expect(opt1).NotTo(Equal(opt2))
		})

		It("re-uses previously named option", func() {
			opt1 := cli.NewOption("OPTION_A", nil)
			opt2 := cli.NewOption("OPTION_A", nil)

			Expect(opt1).To(Equal(opt2))
		})

		It("can invoke custom option", func() {
			act := new(joeclifakes.FakeAction)
			myCustomOption := cli.NewOption("MY_CUSTOM_OPTION", act)
			app := &cli.App{
				Name:    "app",
				Options: myCustomOption,
			}
			app.RunContext(context.TODO(), []string{"app"})
			Expect(act.ExecuteCallCount()).To(Equal(1))
		})

		It("custom option is marshal", func() {
			opt := cli.NewOption("MY_CUSTOM_OPTION", nil)

			actual, _ := json.Marshal(opt)
			Expect(string(actual)).To(Equal("\"MY_CUSTOM_OPTION\""))

			var o cli.Option
			_ = json.Unmarshal(actual, &o)
			Expect(o).To(Equal(opt))
		})
	})
})
