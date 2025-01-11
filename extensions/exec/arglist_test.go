package exec_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ArgList", func() {

	type argList struct {
		self           []string
		args           []string
		usePlaceholder bool
	}

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected argList) {
				actual := &exec.ArgList{}
				for _, a := range args {
					err := actual.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(argList{
					self:           []string(*actual),
					args:           actual.Args(),
					usePlaceholder: actual.UsePlaceholder(),
				}).To(Equal(expected))
			},
			Entry(
				"nominal",
				[]string{"1", "2", ";"},
				argList{
					self:           []string{"1", "2", ";"},
					args:           []string{"1", "2"},
					usePlaceholder: false,
				},
			),
			Entry(
				"plus",
				[]string{"1", "2", "+"},
				argList{
					self:           []string{"1", "2", "+"},
					args:           []string{"1", "2"},
					usePlaceholder: true,
				},
			),
		)
	})

	Describe("Command", func() {
		DescribeTable("examples",
			func(a *exec.ArgList, expected string, expected2 []string) {
				cmd, args := a.Command()
				Expect(cmd).To(Equal(expected))
				Expect(args).To(Equal(expected2))
			},
			Entry("nominal", &exec.ArgList{"1", "2", ";"}, "1", []string{"2"}),
			Entry("no args", &exec.ArgList{"1", ";"}, "1", []string{}),
			Entry("only delim", &exec.ArgList{";"}, "", nil),
			Entry("empty", &exec.ArgList{}, "", nil),
		)
	})

	Describe("String", func() {
		DescribeTable("examples",
			func(a *exec.ArgList, expected string) {
				Expect(a.String()).To(Equal(expected))
			},
			Entry("nominal", &exec.ArgList{"1", "2", ";"}, "1 2 ;"),
			Entry("implied delim", &exec.ArgList{"1"}, "1 ;"),
			Entry("empty", &exec.ArgList{}, ";"),
		)
	})
})

var _ = Describe("ArgListCounter", func() {

	var (
		newCounter = func() cli.ArgCounter {
			return new(exec.ArgList).NewCounter()
		}
	)

	DescribeTable("examples",
		func(args []string) {
			actual := newCounter()
			for _, a := range args {
				err := actual.Take(a, true)
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(actual.Done()).NotTo(HaveOccurred())
		},
		Entry(
			"nominal",
			[]string{"1", "2", ";"},
		),
		Entry(
			"plus",
			[]string{"1", "2", "+"},
		),
	)

	DescribeTable("errors",
		func(args []string, takeErr []types.GomegaMatcher, doneErr types.GomegaMatcher) {
			actual := newCounter()
			for i, a := range args {
				err := actual.Take(a, true)
				Expect(err).To(takeErr[i])
			}

			err := actual.Done()
			Expect(err).To(doneErr)
		},
		Entry(
			"missing terminating char",
			[]string{"name"},
			[]types.GomegaMatcher{Not(HaveOccurred())},
			MatchError("must terminate expression with `;' or `+'"),
		),
		Entry(
			"past done",
			[]string{"name", ";", "other"},
			[]types.GomegaMatcher{Not(HaveOccurred()), Not(HaveOccurred()), MatchError("no more arguments to take")},
			Not(HaveOccurred()),
		),
		Entry(
			"empty", // Uses the same error message as having no terminator
			[]string{},
			[]types.GomegaMatcher{},
			MatchError("must terminate expression with `;' or `+'"),
		),
	)

})
