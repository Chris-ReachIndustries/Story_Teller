package thumbs

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/draw"
)

// Service converts card images (PNG or SVG) to base64 PNG data URLs for OpenAI Vision API
type Service struct {
	basePath string // Base cards path (e.g., /app/cards)
	width    int
	cache    map[string]string // "setID/cardID" → data:image/png;base64,...
	mu       sync.RWMutex
}

// NewService creates a new thumbnail service
// basePath should be the base cards directory (e.g., /app/cards)
func NewService(basePath string, width int) *Service {
	if width <= 0 {
		width = 320 // Default width
	}
	return &Service{
		basePath: basePath,
		width:    width,
		cache:    make(map[string]string),
	}
}

// GetCardThumbDataURL returns a base64 data URL for a card image from a specific card set
func (s *Service) GetCardThumbDataURL(cardID string, cardSetID string) (string, error) {
	// Build cache key
	cacheKey := cardSetID + "/" + cardID

	// Check cache first
	s.mu.RLock()
	if dataURL, ok := s.cache[cacheKey]; ok {
		s.mu.RUnlock()
		return dataURL, nil
	}
	s.mu.RUnlock()

	// Generate thumbnail
	dataURL, err := s.generateThumb(cardID, cardSetID)
	if err != nil {
		return "", err
	}

	// Cache the result
	s.mu.Lock()
	s.cache[cacheKey] = dataURL
	s.mu.Unlock()

	return dataURL, nil
}

// generateThumb renders an image to PNG thumbnail and returns base64 data URL
// Supports both SVG and PNG source images
func (s *Service) generateThumb(cardID string, cardSetID string) (string, error) {
	// Build path: basePath/cardSetID/images/cardID.png
	setPath := filepath.Join(s.basePath, cardSetID)
	pngPath := filepath.Join(setPath, "images", cardID+".png")
	svgPath := filepath.Join(setPath, "images", cardID+".svg")

	if _, err := os.Stat(pngPath); err == nil {
		return s.generateThumbFromPNG(pngPath)
	}

	if _, err := os.Stat(svgPath); err == nil {
		return s.generateThumbFromSVG(svgPath)
	}

	return "", fmt.Errorf("card image not found: %s in set '%s' (looked in %s) - if using fantasy cards, ensure Git LFS is installed and run 'git lfs pull'", cardID, cardSetID, filepath.Dir(pngPath))
}

// generateThumbFromPNG creates a thumbnail from a PNG file
func (s *Service) generateThumbFromPNG(pngPath string) (string, error) {
	// Open and decode PNG
	file, err := os.Open(pngPath)
	if err != nil {
		return "", fmt.Errorf("failed to open PNG: %w", err)
	}
	defer file.Close()

	srcImg, err := png.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Calculate dimensions maintaining aspect ratio
	srcBounds := srcImg.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	w := s.width
	h := int(float64(w) * float64(srcHeight) / float64(srcWidth))

	// Create resized image
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// Use high-quality scaling
	draw.CatmullRom.Scale(dst, dst.Bounds(), srcImg, srcBounds, draw.Over, nil)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Create data URL
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURL := "data:image/png;base64," + b64

	return dataURL, nil
}

// generateThumbFromSVG renders SVG to PNG and returns base64 data URL
func (s *Service) generateThumbFromSVG(svgPath string) (string, error) {
	// Parse SVG
	icon, err := oksvg.ReadIcon(svgPath, oksvg.StrictErrorMode)
	if err != nil {
		return "", fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Calculate dimensions maintaining aspect ratio
	w := float64(s.width)
	h := w * icon.ViewBox.H / icon.ViewBox.W

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))

	// Set up rasterizer
	icon.SetTarget(0, 0, w, h)
	scanner := rasterx.NewScannerGV(int(w), int(h), img, img.Bounds())
	raster := rasterx.NewDasher(int(w), int(h), scanner)

	// Render
	icon.Draw(raster, 1.0)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	// Create data URL
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURL := "data:image/png;base64," + b64

	return dataURL, nil
}

// PrewarmCache loads thumbnails for given card IDs into cache for a specific card set
func (s *Service) PrewarmCache(cardIDs []string, cardSetID string) error {
	for _, cardID := range cardIDs {
		if _, err := s.GetCardThumbDataURL(cardID, cardSetID); err != nil {
			// Log but don't fail - some cards might not have images
			fmt.Printf("Warning: failed to prewarm cache for %s in set %s: %v\n", cardID, cardSetID, err)
		}
	}
	return nil
}

// ClearCache clears the thumbnail cache
func (s *Service) ClearCache() {
	s.mu.Lock()
	s.cache = make(map[string]string)
	s.mu.Unlock()
}
