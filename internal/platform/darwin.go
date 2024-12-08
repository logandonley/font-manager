package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type darwinManager struct{}

func newDarwinManager() Manager {
	return &darwinManager{}
}

func (m *darwinManager) GetFontPaths() (FontPaths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return FontPaths{}, fmt.Errorf("getting user home directory: %w", err)
	}

	paths := FontPaths{
		SystemDir: "/Library/Fonts",
		UserDir:   filepath.Join(homeDir, "Library/Fonts"),
	}

	// Ensure user fonts directory exists
	if err := os.MkdirAll(paths.UserDir, 0755); err != nil {
		return FontPaths{}, fmt.Errorf("creating user fonts directory: %w", err)
	}

	return paths, nil
}

func (m *darwinManager) UpdateFontCache() error {
	// macOS automatically detects new fonts, but we can force a refresh
	// by touching the fonts directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting user home directory: %w", err)
	}

	fontsDir := filepath.Join(homeDir, "Library/Fonts")
	now := time.Now()
	if err := os.Chtimes(fontsDir, now, now); err != nil {
		return fmt.Errorf("updating directory timestamp: %w", err)
	}

	// For older macOS versions, we might need to restart the font server
	if err := exec.Command("atsutil", "databases", "-remove").Run(); err == nil {
		if err := exec.Command("atsutil", "server", "-shutdown").Run(); err != nil {
			return fmt.Errorf("restarting font server: %w", err)
		}
	}

	return nil
}
