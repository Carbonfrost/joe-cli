package exec_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ArgList", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected *exec.ArgList, expectedArgs []string, usePlaceholder bool) {
				actual := &exec.ArgList{}
				for _, a := range args {
					err := actual.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(actual).To(Equal(expected))
				Expect(actual.Args()).To(Equal(expectedArgs))
				Expect(actual.UsePlaceholder()).To(Equal(usePlaceholder))
			},
			Entry(
				"nominal",
				[]string{"1", "2", ";"},
				&exec.ArgList{"1", "2", ";"},
				[]string{"1", "2"},
				false,
			),
			Entry(
				"plus",
				[]string{"1", "2", "+"},
				&exec.ArgList{"1", "2", "+"},
				[]string{"1", "2"},
				true,
			),
		)
	})

	Describe("String", func() {
		DescribeTable("examples",
			func(a *exec.ArgList, expected string) {
				Expect(a.String()).To(Equal(expected))
			},
			Entry("nominal", &exec.ArgList{"1", "2", ";"}, "1 2 ;"),
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
	)

})
