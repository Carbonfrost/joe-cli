package value_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Value Suite")
}
