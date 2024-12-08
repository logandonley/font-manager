package fm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Font Manager Suite")
}
