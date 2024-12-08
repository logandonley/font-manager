package fm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/logandonley/font-manager/internal/platform"
)

// Manager handles font operations
type Manager interface {
	// Install installs a font from any registered source
	Install(ctx context.Context, name string) error

	// InstallFromURL installs a font from a direct URL
	InstallFromURL(ctx context.Context, url string) error

	// Uninstall removes a font
	Uninstall(ctx context.Context, name string) error

	// IsInstalled checks if a font is installed
	IsInstalled(ctx context.Context, name string) (bool, error)

	// List returns all installed fonts
	List(ctx context.Context) ([]Font, error)

	// RegisterSource adds a new source to search for fonts
	RegisterSource(source Source) error

	// InstallFromConfig installs fonts from a config file
	InstallFromConfig(ctx context.Context, reader io.Reader) error
}

// DefaultManager provides the standard font management implementation
type DefaultManager struct {
	sources   []Source
	installer *FontInstaller
	platform  platform.Manager
}

// NewManager creates a new font manager using platform-specific settings
func NewManager() (*DefaultManager, error) {
	platformMgr := platform.New()

	paths, err := platformMgr.GetFontPaths()
	if err != nil {
		return nil, fmt.Errorf("getting font paths: %w", err)
	}

	installer := NewFontInstaller(paths.UserDir)

	return &DefaultManager{
		installer: installer,
		platform:  platformMgr,
	}, nil
}

func NewManagerWithPlatform(platform platform.Manager) *DefaultManager {
	paths, err := platform.GetFontPaths()
	if err != nil {
		panic(fmt.Sprintf("failed to get font paths: %v", err))
	}

	return &DefaultManager{
		installer: NewFontInstaller(paths.UserDir),
		platform:  platform,
		sources:   make([]Source, 0),
	}
}

// UpdateCache updates the system font cache
func (m *DefaultManager) UpdateCache() error {
	return m.platform.UpdateFontCache()
}

// ParseFontSpec parses a font specification line into a Font struct
func ParseFontSpec(line string) (*Font, error) {
	// Skip empty lines and comments
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	// Check if it's a URL
	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		_, err := url.Parse(line)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		return &Font{
			Source: "url",
			URL:    line,
			Name:   getFontNameFromURL(line),
		}, nil
	}

	// Check for source specification with @
	parts := strings.Split(line, "@")
	name := strings.TrimSpace(parts[0])
	source := ""
	if len(parts) > 1 {
		source = strings.TrimSpace(parts[1])
	}

	return &Font{
		Name:   name,
		Source: source,
	}, nil
}

// InstallFromConfig implements bulk font installation from a config file
func (m *DefaultManager) InstallFromConfig(ctx context.Context, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	var errors []error

	for scanner.Scan() {
		font, err := ParseFontSpec(scanner.Text())
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if font == nil {
			continue // Skip empty lines and comments
		}

		// For both URL and source-specific fonts, we can use the regular Install
		// The Font struct already contains the necessary source and URL information
		err = m.Install(ctx, font.Name)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to install %s: %w", font.Name, err))
		}
	}

	if err := scanner.Err(); err != nil {
		errors = append(errors, fmt.Errorf("error reading config: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered errors during installation: %v", errors)
	}

	return nil
}

func getFontNameFromURL(urlStr string) string {
	// Extract filename from URL and clean it up
	u, _ := url.Parse(urlStr)
	parts := strings.Split(u.Path, "/")
	filename := parts[len(parts)-1]

	// Remove extension and common suffixes
	name := strings.TrimSuffix(filename, ".zip")
	name = strings.TrimSuffix(name, ".ttf")
	name = strings.TrimSuffix(name, ".otf")

	return name
}

func (m *DefaultManager) Install(ctx context.Context, name string) error {
	// First check if it's already installed
	installed, err := m.IsInstalled(ctx, name)
	if err != nil {
		return fmt.Errorf("checking if font is installed: %w", err)
	}
	if installed {
		return fmt.Errorf("font %q is already installed", name)
	}

	// If it looks like a URL, treat it as a direct URL installation
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		font := Font{
			Name:   getFontNameFromURL(name),
			Source: "url",
			URL:    name,
		}

		// Create a simple HTTP client for direct URL downloads
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, "GET", name, nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("downloading font: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Install the font
		if err := m.installer.Install(font, resp.Body); err != nil {
			return fmt.Errorf("installing font: %w", err)
		}

		// Update font cache
		return m.UpdateCache()
	}

	// Check if there's a source specification with @
	sourceName := ""
	fontName := name
	if parts := strings.Split(name, "@"); len(parts) > 1 {
		fontName = strings.TrimSpace(parts[0])
		sourceName = strings.TrimSpace(parts[1])
	}

	// If a specific source is requested, use only that source
	if sourceName != "" {
		for _, source := range m.sources {
			if source.Name() == sourceName {
				return m.installFromSource(ctx, fontName, source)
			}
		}
		return fmt.Errorf("source %q not found", sourceName)
	}

	// Try all sources in order
	var lastErr error
	for _, source := range m.sources {
		err := m.installFromSource(ctx, fontName, source)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return fmt.Errorf("font %q not found in any source: %v", name, lastErr)
	}
	return nil
}

// Helper method to install from a specific source
func (m *DefaultManager) installFromSource(ctx context.Context, name string, source Source) error {
	fonts, err := source.Search(ctx, name)
	if err != nil {
		return fmt.Errorf("searching in %s: %w", source.Name(), err)
	}

	if len(fonts) == 0 {
		return fmt.Errorf("font not found in %s", source.Name())
	}

	data, err := source.Download(ctx, fonts[0])
	if err != nil {
		return fmt.Errorf("downloading from %s: %w", source.Name(), err)
	}
	defer data.Close()

	if err := m.installer.Install(fonts[0], data); err != nil {
		return fmt.Errorf("installing font: %w", err)
	}

	return m.UpdateCache()
}

// RegisterSource adds a new source to search for fonts
func (m *DefaultManager) RegisterSource(source Source) error {
	// Check if source is nil
	if source == nil {
		return fmt.Errorf("cannot register nil source")
	}

	// Check for duplicate sources
	for _, existing := range m.sources {
		if existing.Name() == source.Name() {
			return fmt.Errorf("source %q is already registered", source.Name())
		}
	}

	// Add the source to our list
	m.sources = append(m.sources, source)
	return nil
}

// List returns all installed fonts
func (m *DefaultManager) List(ctx context.Context) ([]Font, error) {
	paths, err := m.platform.GetFontPaths()
	if err != nil {
		return nil, fmt.Errorf("getting font paths: %w", err)
	}

	var fonts []Font

	// Read fonts from user directory
	userFonts, err := m.listFontsInDir(paths.UserDir)
	if err != nil {
		return nil, fmt.Errorf("listing user fonts: %w", err)
	}
	fonts = append(fonts, userFonts...)

	// Optionally read from system directory if we have permission
	systemFonts, err := m.listFontsInDir(paths.SystemDir)
	if err == nil {
		fonts = append(fonts, systemFonts...)
	}
	// We intentionally ignore system directory errors since we might not have permission

	return fonts, nil
}

// FontMetadata contains additional font information
type FontMetadata struct {
	InstalledAt time.Time         `json:"installed_at"`
	Additional  map[string]string `json:"additional,omitempty"`
}

func (m *DefaultManager) listFontsInDir(dir string) ([]Font, error) {
	var fonts []Font

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's not a font file
		if info.IsDir() || !isFontFile(info.Name()) {
			return nil
		}

		// Get relative path from font directory
		relPath, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		// The first directory component after the base dir is the font name
		parts := strings.Split(relPath, string(filepath.Separator))
		fontName := parts[0]
		if fontName == "." {
			fontName = strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		}

		// Check if we already have this font in our list
		for _, existing := range fonts {
			if existing.Name == fontName {
				return nil
			}
		}

		// Build the font object with metadata
		font := Font{
			Name: fontName,
			Meta: make(map[string]string),
		}

		fontDir := filepath.Dir(path)

		// Read source information
		if sourceBytes, err := os.ReadFile(filepath.Join(fontDir, ".source")); err == nil {
			font.Source = strings.TrimSpace(string(sourceBytes))
		}

		// Read installation timestamp
		if timestampBytes, err := os.ReadFile(filepath.Join(fontDir, ".installed")); err == nil {
			font.Meta["installed_at"] = strings.TrimSpace(string(timestampBytes))
		}

		// Read additional metadata
		metadataPath := filepath.Join(fontDir, ".metadata")
		if metadataBytes, err := os.ReadFile(metadataPath); err == nil {
			var additionalMeta map[string]string
			if err := json.Unmarshal(metadataBytes, &additionalMeta); err == nil {
				// Merge additional metadata into the Meta map
				for k, v := range additionalMeta {
					font.Meta[k] = v
				}
			}
		}

		// Add file path information
		font.Meta["path"] = path
		font.Meta["directory"] = fontDir

		fonts = append(fonts, font)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dir, err)
	}

	return fonts, nil
}

func (m *DefaultManager) IsInstalled(ctx context.Context, name string) (bool, error) {
	fonts, err := m.List(ctx)
	if err != nil {
		return false, fmt.Errorf("checking installation status: %w", err)
	}

	// Normalize the name for comparison
	normalizedName := sanitizeFontName(name)

	for _, font := range fonts {
		if sanitizeFontName(font.Name) == normalizedName {
			return true, nil
		}
	}

	return false, nil
}

func (m *DefaultManager) Uninstall(ctx context.Context, name string) error {
	// First check if the font is installed and get its metadata
	fonts, err := m.List(ctx)
	if err != nil {
		return fmt.Errorf("checking font installation: %w", err)
	}

	// Normalize the name for comparison
	normalizedName := sanitizeFontName(name)

	var targetFont *Font
	for _, font := range fonts {
		if sanitizeFontName(font.Name) == normalizedName {
			targetFont = &font
			break
		}
	}

	if targetFont == nil {
		return fmt.Errorf("font %q is not installed", name)
	}

	// Get the font directory from metadata
	fontDir, ok := targetFont.Meta["directory"]
	if !ok {
		return fmt.Errorf("font directory information missing")
	}

	// Check if this is in the user directory (we shouldn't remove system fonts)
	paths, err := m.platform.GetFontPaths()
	if err != nil {
		return fmt.Errorf("getting font paths: %w", err)
	}

	if !strings.HasPrefix(fontDir, paths.UserDir) {
		return fmt.Errorf("cannot uninstall system font %q", name)
	}

	// Remove the entire font directory
	if err := os.RemoveAll(fontDir); err != nil {
		return fmt.Errorf("removing font directory: %w", err)
	}

	// Update the system's font cache
	if err := m.UpdateCache(); err != nil {
		// Log the error but don't fail - the font is already removed
		fmt.Fprintf(os.Stderr, "Warning: failed to update font cache: %v\n", err)
	}

	return nil
}
