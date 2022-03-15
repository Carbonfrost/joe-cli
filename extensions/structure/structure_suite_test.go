package structure_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStructure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Structure Suite")
}
