package cli_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGocli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gocli Suite")
}
