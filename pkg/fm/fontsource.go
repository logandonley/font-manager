package fm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// FontSourceAPI provides access to fontsource.org
type FontSourceAPI struct {
	client *http.Client
}

func NewFontSourceAPI() *FontSourceAPI {
	return &FontSourceAPI{
		client: defaultClient,
	}
}

func (s *FontSourceAPI) Name() string {
	return "fontsource"
}

type fontSourceFont struct {
	ID     string `json:"id"`
	Family string `json:"family"`
}

func (s *FontSourceAPI) Search(ctx context.Context, name string) ([]Font, error) {
	encodedName := url.QueryEscape(name)
	reqURL := fmt.Sprintf("https://api.fontsource.org/v1/fonts?family=%s", encodedName)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating search request: %w", err)
	}

	// Add required headers
	req.Header.Set("User-Agent", "FontManager/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searching fonts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var fonts []fontSourceFont
	if err := json.NewDecoder(resp.Body).Decode(&fonts); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var results []Font
	for _, f := range fonts {
		results = append(results, Font{
			Name:   f.Family,
			Source: s.Name(),
			Meta:   map[string]string{"id": f.ID},
		})
	}

	return results, nil
}

func (s *FontSourceAPI) Download(ctx context.Context, font Font) (io.ReadCloser, error) {
	fontID, ok := font.Meta["id"]
	if !ok {
		// If we don't have the ID, try to search for it
		fonts, err := s.Search(ctx, font.Name)
		if err != nil {
			return nil, fmt.Errorf("searching for font ID: %w", err)
		}
		if len(fonts) == 0 {
			return nil, fmt.Errorf("font not found: %s", font.Name)
		}
		fontID = fonts[0].Meta["id"]
	}

	downloadURL := fmt.Sprintf("https://r2.fontsource.org/fonts/%s@latest/download.zip", fontID)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}

	req.Header.Set("User-Agent", "FontManager/1.0")

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
