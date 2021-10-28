package cli_test

import (
	"encoding/json"

	"github.com/Carbonfrost/joe-cli"
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
			Entry("compound", cli.No|cli.Hidden, "HIDDEN, NO"),
		)
	})
})
