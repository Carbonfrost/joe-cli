package cli_test

import (
	"encoding/json"
	"strconv"

	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/joe-clifakes"
	. "github.com/onsi/ginkgo/v2"
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
			Entry("Merge", cli.Merge, "MERGE"),
			Entry("RightToLeft", cli.RightToLeft, "RIGHT_TO_LEFT"),
			Entry("FileReference", cli.FileReference, "FILE_REFERENCE"),
			Entry("AllowFileReference", cli.AllowFileReference, "ALLOW_FILE_REFERENCE"),
			Entry("SortedCommands", cli.SortedCommands, "SORTED_COMMANDS"),
			Entry("SortedFlags", cli.SortedFlags, "SORTED_FLAGS"),
			Entry("ImpliedAction", cli.ImpliedAction, "IMPLIED_ACTION"),
			Entry("Visible", cli.Visible, "VISIBLE"),
			Entry("DisableAutoVisibility", cli.DisableAutoVisibility, "DISABLE_AUTO_VISIBILITY"),
			Entry("compound", cli.No|cli.Hidden, "HIDDEN, NO"),
		)

		Describe("supports all values for Option", func() {
			for o := 1; o < int(cli.MaxOption); o <<= 1 {
				opt := cli.Option(o)
				It("value "+strconv.Itoa(o), func() {
					actual, err := json.Marshal(opt)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(actual)).To(MatchRegexp(`"[A-Z_]+"`))
				})
			}
		})
	})
})

var _ = Describe("FeatureMap", func() {

	type Option int

	const (
		Lo  Option = 1
		Alp Option = 2
		Bet Option = 4
	)

	It("splits the options and invokes them", func() {
		loAction := new(joeclifakes.FakeAction)
		alpAction := new(joeclifakes.FakeAction)
		betAction := new(joeclifakes.FakeAction)
		fm := cli.FeatureMap[Option]{
			Lo:  loAction,
			Alp: alpAction,
			Bet: betAction,
		}
		cli.Initialized(&cli.Flag{}).Do(fm.Pipeline(Lo | Alp | Bet))

		Expect(loAction.ExecuteCallCount()).To(Equal(1))
		Expect(alpAction.ExecuteCallCount()).To(Equal(1))
		Expect(betAction.ExecuteCallCount()).To(Equal(1))

	})

	It("invokes composite flags in order of hamming weight", func() {
		alpLoAction := new(joeclifakes.FakeAction)
		alpAction := new(joeclifakes.FakeAction)
		betAction := new(joeclifakes.FakeAction)
		fm := cli.FeatureMap[Option]{
			Lo | Alp: alpLoAction,
			Alp:      alpAction,
			Bet:      betAction,
		}
		cli.Initialized(&cli.Flag{}).Do(fm.Pipeline(Lo | Alp | Bet))

		Expect(alpLoAction.ExecuteCallCount()).To(Equal(1))
		Expect(alpAction.ExecuteCallCount()).To(Equal(0)) // not called because Alp|Lo was available
		Expect(betAction.ExecuteCallCount()).To(Equal(1))
	})

})
