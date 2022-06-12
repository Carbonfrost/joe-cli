package exec_test

import (
	"github.com/Carbonfrost/joe-cli"
	"github.com/Carbonfrost/joe-cli/extensions/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		func(args []string, expected string) {
			actual := newCounter()
			for _, a := range args {
				err := actual.Take(a, true)
				Expect(err).NotTo(HaveOccurred())
			}

			err := actual.Done()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(expected))
		},
		Entry(
			"missing terminating char",
			[]string{"name"},
			"must terminate expression with `;' or `+'",
		),
	)

})
