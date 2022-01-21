package cli_test

import (
	"errors"

	"github.com/Carbonfrost/joe-cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Exit", func() {
	DescribeTable("arguments",
		func(message interface{}, expectedMessage types.GomegaMatcher, expectedCode int) {
			var err cli.ExitCoder
			switch msg := message.(type) {
			case []interface{}:
				err = cli.Exit(msg...)
			default:
				err = cli.Exit(message)
			}

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(expectedMessage)
			Expect(err.ExitCode()).To(Equal(expectedCode))
		},
		Entry("string", "message", Equal("message"), 1),
		Entry("string slice", []interface{}{"class", "ified"}, Equal("classified"), 1),
		Entry("end with exit code", []interface{}{"b", 255}, Equal("b"), 255),
		Entry("end with error code", []interface{}{"error", cli.UnexpectedArgument}, Equal("unexpected argument: error"), 2),
		Entry("error and error code", []interface{}{errors.New("error"), cli.UnexpectedArgument}, Equal("unexpected argument: error"), 2),
		Entry("already exit coder", cli.Exit("a", 255), Equal("a"), 255),
		Entry("error", errors.New("a"), Equal("a"), 1),
		Entry("error code", cli.UnexpectedArgument, Equal("unexpected argument"), 2),
	)
})
