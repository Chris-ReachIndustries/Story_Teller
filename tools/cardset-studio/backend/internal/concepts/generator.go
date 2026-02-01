package concepts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Generator creates card concepts using Ollama
type Generator struct {
	ollamaURL  string
	httpClient *http.Client
}

// NewGenerator creates a new concept generator
func NewGenerator(ollamaURL string) *Generator {
	return &Generator{
		ollamaURL: ollamaURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// CardConcept represents a generated card concept
type CardConcept struct {
	CardID string   `json:"cardId"`
	Title  string   `json:"title"`
	Prompt string   `json:"prompt"`
	Tags   []string `json:"tags"`
}

// GenerateConcepts creates card concepts for a given theme
func (g *Generator) GenerateConcepts(theme string, count int) ([]CardConcept, error) {
	prompt := fmt.Sprintf(`You are creating concepts for a fantasy card game like Dixit.
The theme is: "%s"

Generate exactly %d unique card concepts. Each card should be evocative, mysterious, and open to interpretation.

For each card, provide:
1. A short title (2-4 words)
2. A Stable Diffusion prompt to generate the image (focus on visual elements, mood, colors)
3. 3-5 tags for categorization

Format your response as a JSON array with objects containing "title", "prompt", and "tags" fields.

Make the prompts detailed and artistic, including:
- Art style keywords (fantasy art, digital painting, etc.)
- Mood and atmosphere
- Color palette hints
- Composition elements

Example format:
[
  {
    "title": "Moonlit Wanderer",
    "prompt": "fantasy art, mysterious figure walking through enchanted forest at night, silver moonlight filtering through ancient trees, ethereal mist, deep blues and silvers, magical atmosphere, detailed illustration",
    "tags": ["night", "forest", "mystery", "journey"]
  }
]

Generate all %d cards now:`, theme, count, count)

	response, err := g.callOllama(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate concepts: %w", err)
	}

	// Parse the JSON response
	concepts, err := parseConcepts(response, count)
	if err != nil {
		return nil, fmt.Errorf("failed to parse concepts: %w", err)
	}

	// Assign card IDs
	for i := range concepts {
		concepts[i].CardID = fmt.Sprintf("card-%03d", i+1)
	}

	return concepts, nil
}

// callOllama makes a request to the Ollama API
func (g *Generator) callOllama(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":  "llama3.2:3b",
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.8,
			"num_predict": 8000,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := g.httpClient.Post(
		g.ollamaURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API error: %s", string(body))
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Response, nil
}

// parseConcepts extracts CardConcept objects from the LLM response
func parseConcepts(response string, expectedCount int) ([]CardConcept, error) {
	// Try to find JSON array in the response
	jsonRegex := regexp.MustCompile(`\[[\s\S]*\]`)
	jsonMatch := jsonRegex.FindString(response)

	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	var concepts []CardConcept
	if err := json.Unmarshal([]byte(jsonMatch), &concepts); err != nil {
		// Try to clean up the JSON
		cleaned := cleanJSON(jsonMatch)
		if err := json.Unmarshal([]byte(cleaned), &concepts); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	// Validate and enhance prompts
	for i := range concepts {
		concepts[i].Prompt = enhancePrompt(concepts[i].Prompt)
		if len(concepts[i].Tags) == 0 {
			concepts[i].Tags = []string{"fantasy"}
		}
	}

	return concepts, nil
}

// cleanJSON attempts to fix common JSON issues from LLM output
func cleanJSON(s string) string {
	// Remove trailing commas before ] or }
	s = regexp.MustCompile(`,\s*\]`).ReplaceAllString(s, "]")
	s = regexp.MustCompile(`,\s*\}`).ReplaceAllString(s, "}")
	return s
}

// enhancePrompt adds quality keywords to a Stable Diffusion prompt
func enhancePrompt(prompt string) string {
	qualityKeywords := []string{
		"masterpiece",
		"best quality",
		"highly detailed",
	}

	prompt = strings.TrimSpace(prompt)

	// Check if quality keywords already present
	lowerPrompt := strings.ToLower(prompt)
	for _, kw := range qualityKeywords {
		if !strings.Contains(lowerPrompt, kw) {
			prompt = prompt + ", " + kw
		}
	}

	return prompt
}

// RegenerateConceptPrompt creates a new prompt for a specific concept
func (g *Generator) RegenerateConceptPrompt(title, theme string) (string, error) {
	prompt := fmt.Sprintf(`Create a Stable Diffusion prompt for a fantasy card game image.

Card title: "%s"
Theme: "%s"

Create a detailed prompt that describes:
- The main visual elements and subject
- Art style (fantasy art, digital painting, etc.)
- Mood and atmosphere
- Color palette
- Composition

Respond with ONLY the prompt text, no explanation.`, title, theme)

	response, err := g.callOllama(prompt)
	if err != nil {
		return "", err
	}

	return enhancePrompt(strings.TrimSpace(response)), nil
}
