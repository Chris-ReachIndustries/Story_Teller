package images

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Generator creates images using Stable Diffusion (A1111 API)
type Generator struct {
	sdURL      string
	httpClient *http.Client
}

// NewGenerator creates a new image generator
func NewGenerator(sdURL string) *Generator {
	return &Generator{
		sdURL: sdURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // SD can be slow
			// Don't follow redirects - SD API shouldn't redirect
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// GenerationSettings contains parameters for image generation
type GenerationSettings struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	Steps          int     `json:"steps"`
	CFGScale       float64 `json:"cfg_scale"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Sampler        string  `json:"sampler_name"`
	Seed           int64   `json:"seed"`
	Model          string  `json:"model,omitempty"` // SD model checkpoint to use
}

// DefaultSettings returns default generation settings
func DefaultSettings() GenerationSettings {
	return GenerationSettings{
		NegativePrompt: "text, watermark, signature, blurry, low quality, deformed, ugly, bad anatomy, cropped, jpeg artifacts, nsfw",
		Steps:          30,
		CFGScale:       7.5,
		Width:          768,
		Height:         1152,
		Sampler:        "DPM++ 2M Karras",
		Seed:           -1, // Random seed
	}
}

// txt2imgRequest is the A1111 API request format
type txt2imgRequest struct {
	Prompt           string                 `json:"prompt"`
	NegativePrompt   string                 `json:"negative_prompt"`
	Steps            int                    `json:"steps"`
	CFGScale         float64                `json:"cfg_scale"`
	Width            int                    `json:"width"`
	Height           int                    `json:"height"`
	SamplerName      string                 `json:"sampler_name"`
	Seed             int64                  `json:"seed"`
	BatchSize        int                    `json:"batch_size"`
	NIter            int                    `json:"n_iter"`
	SaveImages       bool                   `json:"save_images"`
	SendImages       bool                   `json:"send_images"`
	OverrideSettings map[string]interface{} `json:"override_settings,omitempty"`
}

// txt2imgResponse is the A1111 API response format
type txt2imgResponse struct {
	Images []string `json:"images"` // Base64 encoded PNGs
	Info   string   `json:"info"`
}

// Generate creates an image from a prompt and saves it to the specified path
func (g *Generator) Generate(prompt string, settings GenerationSettings, outputPath string) error {
	// Build request
	req := txt2imgRequest{
		Prompt:         prompt,
		NegativePrompt: settings.NegativePrompt,
		Steps:          settings.Steps,
		CFGScale:       settings.CFGScale,
		Width:          settings.Width,
		Height:         settings.Height,
		SamplerName:    settings.Sampler,
		Seed:           settings.Seed,
		BatchSize:      1,
		NIter:          1,
		SaveImages:     false,
		SendImages:     true,
	}

	// Add model override if specified
	if settings.Model != "" {
		req.OverrideSettings = map[string]interface{}{
			"sd_model_checkpoint": settings.Model,
		}
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make API call
	resp, err := g.httpClient.Post(
		g.sdURL+"/sdapi/v1/txt2img",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("failed to call SD API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		return fmt.Errorf("SD API redirected (status %d) to %s - check if auth is disabled", resp.StatusCode, location)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SD API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result txt2imgResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Images) == 0 {
		return fmt.Errorf("no images in response")
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(result.Images[0])
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Save to file
	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

// IsAvailable checks if the Stable Diffusion API is accessible
func (g *Generator) IsAvailable() bool {
	resp, err := g.httpClient.Get(g.sdURL + "/sdapi/v1/sd-models")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// GetModels returns available SD models
func (g *Generator) GetModels() ([]string, error) {
	resp, err := g.httpClient.Get(g.sdURL + "/sdapi/v1/sd-models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var models []struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}

	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.Title
	}
	return names, nil
}

// GetSamplers returns available samplers
func (g *Generator) GetSamplers() ([]string, error) {
	resp, err := g.httpClient.Get(g.sdURL + "/sdapi/v1/samplers")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var samplers []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&samplers); err != nil {
		return nil, err
	}

	names := make([]string, len(samplers))
	for i, s := range samplers {
		names[i] = s.Name
	}
	return names, nil
}
