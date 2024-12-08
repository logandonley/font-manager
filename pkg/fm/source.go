package fm

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Font represents a font that can be installed or removed
type Font struct {
	Name   string            // Display name of the font
	Source string            // Source identifier (e.g., "nerdfonts", "fontsource", "url")
	URL    string            // Direct URL if provided
	Meta   map[string]string // Additional metadata
}

// Source defines how to interact with a font source
type Source interface {
	// Name returns the identifier for this source
	Name() string

	// Search looks for fonts matching the given name
	Search(ctx context.Context, name string) ([]Font, error)

	// Download retrieves the font data
	Download(ctx context.Context, font Font) (io.ReadCloser, error)
}

// Common HTTP client with reasonable defaults
var defaultClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}
