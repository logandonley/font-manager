package fm

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FontInstaller handles the installation of fonts into the system
type FontInstaller struct {
	fontDir  string
	cacheCmd string
}

func NewFontInstaller(fontDir string) *FontInstaller {
	return &FontInstaller{
		fontDir:  fontDir,
		cacheCmd: "fc-cache", // default to fc-cache, can be overridden
	}
}

func (fi *FontInstaller) Install(font Font, data io.Reader) error {
	// Read all data into memory to avoid multiple reads
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, data); err != nil {
		return fmt.Errorf("reading font data: %w", err)
	}

	// Create font directory if it doesn't exist
	fontPath := filepath.Join(fi.fontDir, sanitizeFontName(font.Name))
	if err := os.MkdirAll(fontPath, 0755); err != nil {
		return fmt.Errorf("creating font directory: %w", err)
	}

	// Process the zip file
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return fmt.Errorf("reading zip data: %w", err)
	}

	installed := false
	for _, file := range zipReader.File {
		// Skip directories and hidden files
		if file.FileInfo().IsDir() || strings.HasPrefix(filepath.Base(file.Name), ".") {
			continue
		}

		// Check if it's a font file
		if isFontFile(file.Name) {
			if err := fi.extractFontFile(file, fontPath); err != nil {
				return fmt.Errorf("extracting font file %s: %w", file.Name, err)
			}
			installed = true
		}

		// Always extract LICENSE files
		if strings.EqualFold(filepath.Base(file.Name), "LICENSE") {
			if err := fi.extractFontFile(file, fontPath); err != nil {
				return fmt.Errorf("extracting license file: %w", err)
			}
		}
	}

	if !installed {
		return fmt.Errorf("no valid font files found in archive")
	}

	// Store metadata about the font source
	if err := fi.storeMetadata(fontPath, font); err != nil {
		return fmt.Errorf("storing font metadata: %w", err)
	}

	return nil
}

// storeMetadata saves information about the font's source and other metadata
func (fi *FontInstaller) storeMetadata(fontPath string, font Font) error {
	// Store the source information
	if font.Source != "" {
		sourcePath := filepath.Join(fontPath, ".source")
		if err := os.WriteFile(sourcePath, []byte(font.Source), 0644); err != nil {
			return fmt.Errorf("writing source metadata: %w", err)
		}
	}

	// Store additional metadata if present
	if len(font.Meta) > 0 {
		metadataPath := filepath.Join(fontPath, ".metadata")
		metadataJSON, err := json.Marshal(font.Meta)
		if err != nil {
			return fmt.Errorf("marshaling metadata: %w", err)
		}

		if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
			return fmt.Errorf("writing metadata file: %w", err)
		}
	}

	// Store installation timestamp
	timestampPath := filepath.Join(fontPath, ".installed")
	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(timestampPath, []byte(timestamp), 0644); err != nil {
		return fmt.Errorf("writing installation timestamp: %w", err)
	}

	return nil
}

// Uninstall removes a font from the system
func (fi *FontInstaller) Uninstall(fontName string) error {
	fontPath := filepath.Join(fi.fontDir, sanitizeFontName(fontName))

	// Check if font exists
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		return fmt.Errorf("font %s is not installed", fontName)
	}

	// Remove the font directory
	if err := os.RemoveAll(fontPath); err != nil {
		return fmt.Errorf("removing font directory: %w", err)
	}

	return nil
}

// UpdateCache runs the font cache update command
func (fi *FontInstaller) UpdateCache() error {
	cmd := exec.Command(fi.cacheCmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("updating font cache: %s: %w", output, err)
	}
	return nil
}

// IsInstalled checks if a font is installed
func (fi *FontInstaller) IsInstalled(fontName string) bool {
	fontPath := filepath.Join(fi.fontDir, sanitizeFontName(fontName))
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		return false
	}

	// Check if directory contains any font files
	hasFonts := false
	err := filepath.Walk(fontPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isFontFile(info.Name()) {
			hasFonts = true
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		fmt.Printf("error walking path: %v", err)
		return false
	}

	return hasFonts
}

// Helper functions

func isFontFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ttf" || ext == ".otf"
}

func sanitizeFontName(name string) string {
	// Remove any potentially problematic characters from font name
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	return strings.Trim(name, "-")
}

func (fi *FontInstaller) extractFontFile(file *zip.File, destPath string) error {
	// Open the file from the archive
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("opening file in archive: %w", err)
	}
	defer src.Close()

	// Create the destination file
	destFile := filepath.Join(destPath, filepath.Base(file.Name))
	dest, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer dest.Close()

	// Copy the contents
	if _, err := io.Copy(dest, src); err != nil {
		return fmt.Errorf("copying file contents: %w", err)
	}

	return nil
}
