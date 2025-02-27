package synopsis_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSynopsis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Synopsis Suite")
}
