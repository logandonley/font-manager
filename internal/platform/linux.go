package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func hasSudo() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}

func (m *linuxManager) UpdateFontCache() error {
	// First try fc-cache
	if err := runCommand("fc-cache", "-f"); err == nil {
		return nil
	}

	// If fc-cache fails, try with sudo (some distros require this)
	if os.Geteuid() != 0 {
		if !hasSudo() {
			return fmt.Errorf("font cache update failed. Please run 'fc-cache -f' manually with root privileges")
		}

		fmt.Printf("Unable to update font cache with current permissions.\n")
		fmt.Printf("This can happen if system-wide fonts were installed or if the cache is locked.\n")
		fmt.Printf("Attempting to update with elevated privileges. You may be prompted for your password.\n\n")

		if err := runCommand("sudo", "fc-cache", "-f"); err != nil {
			return fmt.Errorf("updating font cache with elevated privileges: %w", err)
		}
	}

	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed:\nCommand: %s %s\nOutput: %s\nError: %w",
			name, name, strings.Join(args, " "), output, err)
	}
	return nil
}
