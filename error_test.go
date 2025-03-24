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
		func(message any, expectedMessage types.GomegaMatcher, expectedCode int) {
			var err cli.ExitCoder
			switch msg := message.(type) {
			case []any:
				err = cli.Exit(msg...)
			default:
				err = cli.Exit(message)
			}

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expectedMessage))
			Expect(err.ExitCode()).To(Equal(expectedCode))
		},
		Entry("string", "message", Equal("message"), 1),
		Entry("string slice", []any{"class", "ified"}, Equal("classified"), 1),
		Entry("end with exit code", []any{"b", 255}, Equal("b"), 255),
		Entry("end with error code", []any{"error", cli.UnexpectedArgument}, Equal("unexpected argument: error"), 2),
		Entry("error and error code", []any{errors.New("error"), cli.UnexpectedArgument}, Equal("unexpected argument: error"), 2),
		Entry("already exit coder", cli.Exit("a", 255), Equal("a"), 255),
		Entry("error", errors.New("a"), Equal("a"), 1),
		Entry("error code", cli.UnexpectedArgument, Equal("unexpected argument"), 2),
		Entry("nil", nil, Equal("exited with status 1"), 1),
		Entry("empty", []any{}, Equal("exited with status 1"), 1),
	)
})
