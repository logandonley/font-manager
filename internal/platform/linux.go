package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type linuxManager struct{}

func newLinuxManager() Manager {
	return &linuxManager{}
}

func (m *linuxManager) GetFontPaths() (FontPaths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return FontPaths{}, fmt.Errorf("getting user home directory: %w", err)
	}

	paths := FontPaths{
		SystemDir: "/usr/local/share/fonts",
		UserDir:   filepath.Join(homeDir, ".local/share/fonts"),
	}

	// Ensure user fonts directory exists
	if err := os.MkdirAll(paths.UserDir, 0755); err != nil {
		return FontPaths{}, fmt.Errorf("creating user fonts directory: %w", err)
	}

	return paths, nil
}

func (m *linuxManager) UpdateFontCache() error {
	// First try fc-cache
	if err := runCommand("fc-cache", "-f"); err == nil {
		return nil
	}

	// If fc-cache fails, try with sudo (some distros require this)
	if os.Geteuid() != 0 {
		if err := runCommand("sudo", "fc-cache", "-f"); err != nil {
			return fmt.Errorf("updating font cache: %w", err)
		}
	}

	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running %s: %s: %w", name, output, err)
	}
	return nil
}
