package platform

import (
	"runtime"
)

// FontPaths represents system and user font directories
type FontPaths struct {
	SystemDir string // System-wide font directory
	UserDir   string // User-specific font directory
}

// Manager handles platform-specific operations
type Manager interface {
	// GetFontPaths returns the system and user font directories
	GetFontPaths() (FontPaths, error)

	// UpdateFontCache updates the system's font cache
	UpdateFontCache() error
}

// New returns a platform-specific manager
func New() Manager {
	if runtime.GOOS == "darwin" {
		return newDarwinManager()
	}
	return newLinuxManager()
}
