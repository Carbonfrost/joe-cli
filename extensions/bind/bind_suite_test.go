package bind_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBind(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bind Suite")
}
