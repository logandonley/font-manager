package platform_test

import (
	"os"

	"github.com/logandonley/font-manager/internal/platform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Platform", func() {
	var (
		tempDir string
		manager platform.Manager
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "platform-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Set up environment for testing
		os.Setenv("HOME", tempDir)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("Linux Manager", func() {
		BeforeEach(func() {
			os.Setenv("GOOS", "linux")
			manager = platform.New()
		})

		It("should return correct font paths", func() {
			paths, err := manager.GetFontPaths()
			Expect(err).NotTo(HaveOccurred())

			Expect(paths.SystemDir).To(Equal("/usr/local/share/fonts"))
			Expect(paths.UserDir).To(ContainSubstring(".local/share/fonts"))
		})
	})

	Context("Darwin Manager", func() {
		BeforeEach(func() {
			os.Setenv("GOOS", "darwin")
			manager = platform.New()
		})

		It("should return correct font paths", func() {
			paths, err := manager.GetFontPaths()
			Expect(err).NotTo(HaveOccurred())

			Expect(paths.SystemDir).To(Equal("/Library/Fonts"))
			Expect(paths.UserDir).To(ContainSubstring("Library/Fonts"))
		})
	})
})
