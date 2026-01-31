package cardgen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// CardWidth is the final card image width
	CardWidth = 1024
	// CardHeight is the final card image height (79:120 ratio like real Dixit cards)
	CardHeight = 1557
	// DALLEWidth is DALL-E's portrait mode width
	DALLEWidth = 1024
	// DALLEHeight is DALL-E's portrait mode height
	DALLEHeight = 1792
)

// DALLEClient handles DALL-E API calls
type DALLEClient struct {
	apiKey  string
	model   string
	size    string
	quality string
	client  *http.Client
}

// NewDALLEClient creates a new DALL-E client
func NewDALLEClient(apiKey, model, size, quality string) *DALLEClient {
	if model == "" {
		model = "dall-e-3"
	}
	if size == "" {
		size = "1024x1792" // Portrait mode
	}
	if quality == "" {
		quality = "standard"
	}

	return &DALLEClient{
		apiKey:  apiKey,
		model:   model,
		size:    size,
		quality: quality,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// GenerateImage generates a single image and saves to file
func (c *DALLEClient) GenerateImage(ctx context.Context, concept CardConcept, outputPath string) error {
	// Enhance prompt with content safety
	safePrompt := concept.Description
	if len(safePrompt) > 3900 { // DALL-E has ~4000 char limit
		safePrompt = safePrompt[:3900]
	}

	reqBody := map[string]interface{}{
		"model":           c.model,
		"prompt":          safePrompt,
		"n":               1,
		"size":            c.size,
		"quality":         c.quality,
		"response_format": "b64_json", // Get base64 directly
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/images/generations", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("DALL-E request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DALL-E API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse DALL-E response: %w", err)
	}

	if result.Error != nil {
		return fmt.Errorf("DALL-E error: %s", result.Error.Message)
	}

	if len(result.Data) == 0 {
		return fmt.Errorf("no image data in response")
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(result.Data[0].B64JSON)
	if err != nil {
		return fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Decode PNG image
	img, err := png.Decode(bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Crop to card aspect ratio (79:120)
	croppedImg := cropToCardRatio(img)

	// Save cropped image
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, croppedImg); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

// cropToCardRatio crops the image to match Dixit card ratio (79:120)
// Crops from center, removing equal amounts from top and bottom
func cropToCardRatio(img image.Image) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate target dimensions maintaining width
	targetWidth := srcWidth
	targetHeight := int(float64(srcWidth) * 120.0 / 79.0) // 79:120 ratio

	// If the source is taller than needed, crop from center
	if srcHeight > targetHeight {
		// Calculate crop offset (center crop)
		yOffset := (srcHeight - targetHeight) / 2

		// Create a subimage (doesn't copy data, just references)
		type subImager interface {
			SubImage(r image.Rectangle) image.Image
		}

		if si, ok := img.(subImager); ok {
			return si.SubImage(image.Rect(0, yOffset, targetWidth, yOffset+targetHeight))
		}

		// Fallback: create new image with cropped content
		cropped := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		for y := 0; y < targetHeight; y++ {
			for x := 0; x < targetWidth; x++ {
				cropped.Set(x, y, img.At(x, y+yOffset))
			}
		}
		return cropped
	}

	// If source is already correct or shorter, return as-is
	return img
}
