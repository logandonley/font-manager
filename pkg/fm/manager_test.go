package fm_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/logandonley/font-manager/internal/platform"
	"github.com/logandonley/font-manager/pkg/fm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Mock platform implementation for testing
type mockPlatform struct {
	fontDir string
}

func (m *mockPlatform) GetFontPaths() (platform.FontPaths, error) {
	return platform.FontPaths{
		SystemDir: filepath.Join(m.fontDir, "system"),
		UserDir:   filepath.Join(m.fontDir, "user"),
	}, nil
}

func (m *mockPlatform) UpdateFontCache() error {
	return nil
}

// Mock font source for testing
type mockSource struct {
	name     string
	fonts    map[string][]byte // name -> zip content
	failures map[string]error  // name -> error
}

type testFont struct {
	name    string
	format  string // "ttf" or "otf"
	content string
}

// Helper function to create a zip file with font content
func createTestZip(fonts ...testFont) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, font := range fonts {
		// Create font file in the zip with appropriate extension
		filename := fmt.Sprintf("%s.%s", font.name, font.format)
		f, err := zipWriter.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("creating %s: %w", filename, err)
		}
		_, err = f.Write([]byte(font.content))
		if err != nil {
			return nil, fmt.Errorf("writing %s: %w", filename, err)
		}

		// Add a dummy license file
		licenseFile, err := zipWriter.Create("LICENSE")
		if err != nil {
			return nil, fmt.Errorf("creating LICENSE: %w", err)
		}
		_, err = licenseFile.Write([]byte("Test License"))
		if err != nil {
			return nil, fmt.Errorf("writing LICENSE: %w", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func newMockSource() *mockSource {
	ms := &mockSource{
		name:     "testsource",
		fonts:    make(map[string][]byte),
		failures: make(map[string]error),
	}
	testFont1 := testFont{
		name:    "TestFont1",
		format:  "ttf",
		content: "fake ttf content",
	}

	testFont2 := testFont{
		name:    "TestFont2",
		format:  "ttf",
		content: "fake ttf content",
	}

	if content, err := createTestZip(testFont1); err == nil {
		ms.fonts["TestFont1"] = content
	}
	if content, err := createTestZip(testFont2); err == nil {
		ms.fonts["TestFont2"] = content
	}

	// Create test fonts with different formats
	ttfFont := testFont{
		name:    "TestTTF",
		format:  "ttf",
		content: "fake ttf content",
	}

	otfFont := testFont{
		name:    "TestOTF",
		format:  "otf",
		content: "fake otf content",
	}

	multiFormatFont := []testFont{
		{
			name:    "TestMulti",
			format:  "ttf",
			content: "ttf version",
		},
		{
			name:    "TestMulti",
			format:  "otf",
			content: "otf version",
		},
	}

	// Add single format fonts
	if content, err := createTestZip(ttfFont); err == nil {
		ms.fonts["TestTTF"] = content
	}
	if content, err := createTestZip(otfFont); err == nil {
		ms.fonts["TestOTF"] = content
	}

	// Add multi-format font
	if content, err := createTestZip(multiFormatFont...); err == nil {
		ms.fonts["TestMulti"] = content
	}

	ms.failures["FailingFont"] = fmt.Errorf("simulated failure")

	return ms
}

func (s *mockSource) Name() string {
	return s.name
}

func (s *mockSource) Search(_ context.Context, name string) ([]fm.Font, error) {
	if err, exists := s.failures[name]; exists {
		return nil, err
	}

	if _, exists := s.fonts[name]; exists {
		return []fm.Font{{
			Name:   name,
			Source: s.name,
		}}, nil
	}
	return nil, nil
}

func (s *mockSource) Download(_ context.Context, font fm.Font) (io.ReadCloser, error) {
	if err, exists := s.failures[font.Name]; exists {
		return nil, err
	}

	content, exists := s.fonts[font.Name]
	if !exists {
		return nil, fmt.Errorf("font not found")
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

var _ = Describe("Font Manager", func() {
	var (
		manager     *fm.DefaultManager
		tempDir     string
		mockSource1 *mockSource
		ctx         context.Context
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "font-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Create system and user font directories
		Expect(os.MkdirAll(filepath.Join(tempDir, "system"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(tempDir, "user"), 0755)).To(Succeed())

		// Create mock source with some test fonts
		mockSource1 = newMockSource()

		// Initialize manager with mocks
		manager = fm.NewManagerWithPlatform(&mockPlatform{fontDir: tempDir})
		Expect(manager.RegisterSource(mockSource1)).To(Succeed())

		ctx = context.Background()
	})

	AfterEach(func() {
		fonts, err := manager.List(ctx)
		if err == nil {
			for _, font := range fonts {
				_ = manager.Uninstall(ctx, font.Name)
			}
		}
		os.RemoveAll(tempDir)
	})

	Describe("Installing fonts", func() {
		Context("with different font formats", func() {
			It("should install TTF fonts", func() {
				Expect(manager.Install(ctx, "TestTTF")).To(Succeed())

				installed, err := manager.IsInstalled(ctx, "TestTTF")
				Expect(err).NotTo(HaveOccurred())
				Expect(installed).To(BeTrue())
			})

			It("should install OTF fonts", func() {
				Expect(manager.Install(ctx, "TestOTF")).To(Succeed())

				installed, err := manager.IsInstalled(ctx, "TestOTF")
				Expect(err).NotTo(HaveOccurred())
				Expect(installed).To(BeTrue())
			})

			It("should install fonts with multiple formats", func() {
				Expect(manager.Install(ctx, "TestMulti")).To(Succeed())

				installed, err := manager.IsInstalled(ctx, "TestMulti")
				Expect(err).NotTo(HaveOccurred())
				Expect(installed).To(BeTrue())

				// Verify both formats were installed
				fonts, err := manager.List(ctx)
				Expect(err).NotTo(HaveOccurred())

				var foundFont *fm.Font
				for _, f := range fonts {
					if f.Name == "TestMulti" {
						foundFont = &f
						break
					}
				}

				Expect(foundFont).NotTo(BeNil())

				// Check the font directory for both formats
				fontDir := foundFont.Meta["directory"]
				Expect(fontDir).NotTo(BeEmpty())

				files, err := os.ReadDir(fontDir)
				Expect(err).NotTo(HaveOccurred())

				var hasTTF, hasOTF bool
				for _, file := range files {
					if strings.HasSuffix(file.Name(), ".ttf") {
						hasTTF = true
					}
					if strings.HasSuffix(file.Name(), ".otf") {
						hasOTF = true
					}
				}

				Expect(hasTTF).To(BeTrue(), "Should have TTF file")
				Expect(hasOTF).To(BeTrue(), "Should have OTF file")
			})
		})
		It("should install a font successfully", func() {
			Expect(manager.Install(ctx, "TestFont1")).To(Succeed())

			// Verify the font was installed
			installed, err := manager.IsInstalled(ctx, "TestFont1")
			Expect(err).NotTo(HaveOccurred())
			Expect(installed).To(BeTrue())
		})

		It("should handle installation failures gracefully", func() {
			err := manager.Install(ctx, "FailingFont")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("simulated failure"))
		})

		It("should not reinstall already installed fonts", func() {
			Expect(manager.Install(ctx, "TestFont1")).To(Succeed())
			err := manager.Install(ctx, "TestFont1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already installed"))
		})
	})

	Describe("Listing fonts", func() {
		BeforeEach(func() {
			Expect(manager.Install(ctx, "TestFont1")).To(Succeed())
			Expect(manager.Install(ctx, "TestFont2")).To(Succeed())
		})

		It("should list all installed fonts", func() {
			fonts, err := manager.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(fonts).To(HaveLen(2))

			fontNames := []string{fonts[0].Name, fonts[1].Name}
			Expect(fontNames).To(ContainElements("TestFont1", "TestFont2"))
		})

		It("should include source information in listed fonts", func() {
			fonts, err := manager.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			for _, font := range fonts {
				Expect(font.Source).To(Equal("testsource"))
			}
		})
	})

	Describe("Uninstalling fonts", func() {
		BeforeEach(func() {
			Expect(manager.Install(ctx, "TestFont1")).To(Succeed())
		})

		It("should uninstall fonts successfully", func() {
			Expect(manager.Uninstall(ctx, "TestFont1")).To(Succeed())

			installed, err := manager.IsInstalled(ctx, "TestFont1")
			Expect(err).NotTo(HaveOccurred())
			Expect(installed).To(BeFalse())
		})

		It("should fail when trying to uninstall non-existent fonts", func() {
			err := manager.Uninstall(ctx, "NonExistentFont")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not installed"))
		})
	})
})
