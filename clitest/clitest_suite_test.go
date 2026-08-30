package clitest_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClitest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clitest Suite")
}
