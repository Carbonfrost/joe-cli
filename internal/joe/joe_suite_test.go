package joe_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJoe(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Joe Suite")
}
