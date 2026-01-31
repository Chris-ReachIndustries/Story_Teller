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
	cardsPath string
	width     int
	cache     map[string]string // cardID → data:image/png;base64,...
	mu        sync.RWMutex
}

// NewService creates a new thumbnail service
func NewService(cardsPath string, width int) *Service {
	if width <= 0 {
		width = 320 // Default width
	}
	return &Service{
		cardsPath: cardsPath,
		width:     width,
		cache:     make(map[string]string),
	}
}

// GetCardThumbDataURL returns a base64 data URL for the card image
func (s *Service) GetCardThumbDataURL(cardID string) (string, error) {
	// Check cache first
	s.mu.RLock()
	if dataURL, ok := s.cache[cardID]; ok {
		s.mu.RUnlock()
		return dataURL, nil
	}
	s.mu.RUnlock()

	// Generate thumbnail
	dataURL, err := s.generateThumb(cardID)
	if err != nil {
		return "", err
	}

	// Cache the result
	s.mu.Lock()
	s.cache[cardID] = dataURL
	s.mu.Unlock()

	return dataURL, nil
}

// generateThumb renders an image to PNG thumbnail and returns base64 data URL
// Supports both SVG and PNG source images
func (s *Service) generateThumb(cardID string) (string, error) {
	// Try PNG first (generated cards), then SVG (manual cards)
	pngPath := filepath.Join(s.cardsPath, "images", cardID+".png")
	svgPath := filepath.Join(s.cardsPath, "images", cardID+".svg")

	if _, err := os.Stat(pngPath); err == nil {
		return s.generateThumbFromPNG(pngPath)
	}

	if _, err := os.Stat(svgPath); err == nil {
		return s.generateThumbFromSVG(svgPath)
	}

	return "", fmt.Errorf("card image not found: %s (looked in %s) - if using fantasy cards, ensure Git LFS is installed and run 'git lfs pull'", cardID, filepath.Dir(pngPath))
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

// PrewarmCache loads thumbnails for given card IDs into cache
func (s *Service) PrewarmCache(cardIDs []string) error {
	for _, cardID := range cardIDs {
		if _, err := s.GetCardThumbDataURL(cardID); err != nil {
			// Log but don't fail - some cards might not have images
			fmt.Printf("Warning: failed to prewarm cache for %s: %v\n", cardID, err)
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
