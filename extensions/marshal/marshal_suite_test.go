package marshal_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMarshal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Marshal Suite")
}
