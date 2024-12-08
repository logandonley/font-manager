package fm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// NerdFontsSource provides access to NerdFonts repository
type NerdFontsSource struct {
	client *http.Client
}

func NewNerdFontsSource() *NerdFontsSource {
	return &NerdFontsSource{
		client: defaultClient,
	}
}

func (s *NerdFontsSource) Name() string {
	return "nerdfonts"
}

type nerdFontsRelease struct {
	TagName string `json:"tag_name"`
}

func (s *NerdFontsSource) getLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx,
		"GET",
		"https://api.github.com/repos/ryanoasis/nerd-fonts/releases/latest",
		nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release nerdFontsRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return release.TagName, nil
}

func (s *NerdFontsSource) Search(ctx context.Context, name string) ([]Font, error) {
	// NerdFonts doesn't have a search API, so we'll just create a Font object
	// if the name matches our expected format

	// Clean up the name to match NerdFonts naming convention
	cleanName := strings.ReplaceAll(strings.TrimSpace(name), " ", "")

	// You might want to maintain a list of known NerdFonts or fetch it dynamically
	// For now, we'll just assume if it looks like a NerdFont name, it might be one
	return []Font{{
		Name:   cleanName,
		Source: s.Name(),
		Meta:   map[string]string{"pending": "true"},
	}}, nil
}

func (s *NerdFontsSource) Download(ctx context.Context, font Font) (io.ReadCloser, error) {
	version, err := s.getLatestVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest version: %w", err)
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/ryanoasis/nerd-fonts/releases/download/%s/%s.zip",
		version,
		font.Name,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading font: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
