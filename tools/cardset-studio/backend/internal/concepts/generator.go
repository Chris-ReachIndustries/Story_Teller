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
// It batches requests to handle large counts reliably
func (g *Generator) GenerateConcepts(theme string, count int) ([]CardConcept, error) {
	const batchSize = 5   // Generate 5 concepts at a time for better reliability
	const maxRetries = 2  // Retries per batch
	const maxFillPasses = 3 // Maximum fill-in passes to reach target count
	var allConcepts []CardConcept

	// Phase 1: Initial batch generation
	fmt.Printf("Phase 1: Generating %d concepts in batches of %d...\n", count, batchSize)
	for i := 0; i < count; i += batchSize {
		remaining := count - i
		currentBatch := batchSize
		if remaining < batchSize {
			currentBatch = remaining
		}

		batchNum := (i / batchSize) + 1
		totalBatches := (count + batchSize - 1) / batchSize

		// Retry logic for failed batches
		var concepts []CardConcept
		var err error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if attempt > 1 {
				fmt.Printf("Retrying batch %d (attempt %d/%d)...\n", batchNum, attempt, maxRetries)
			} else {
				fmt.Printf("Generating concepts batch %d/%d (%d concepts)...\n", batchNum, totalBatches, currentBatch)
			}

			concepts, err = g.generateConceptBatch(theme, currentBatch, len(allConcepts))
			if err == nil {
				break
			}
			fmt.Printf("Warning: batch %d attempt %d failed: %v\n", batchNum, attempt, err)
		}

		if concepts != nil {
			allConcepts = append(allConcepts, concepts...)
		}
	}

	// Phase 2: Fill-in pass to reach target count
	for fillPass := 1; fillPass <= maxFillPasses && len(allConcepts) < count; fillPass++ {
		missing := count - len(allConcepts)
		fmt.Printf("Phase 2: Fill-in pass %d - generating %d missing concepts...\n", fillPass, missing)

		// Generate missing concepts in smaller batches
		fillBatchSize := 3 // Smaller batches for fill-in
		for i := 0; i < missing; i += fillBatchSize {
			remaining := missing - i
			currentBatch := fillBatchSize
			if remaining < fillBatchSize {
				currentBatch = remaining
			}

			// Try to generate fill-in batch
			var concepts []CardConcept
			var err error
			for attempt := 1; attempt <= maxRetries; attempt++ {
				concepts, err = g.generateConceptBatch(theme, currentBatch, len(allConcepts))
				if err == nil {
					break
				}
				if attempt < maxRetries {
					fmt.Printf("Fill-in attempt %d failed, retrying...\n", attempt)
				}
			}

			if concepts != nil {
				allConcepts = append(allConcepts, concepts...)
				fmt.Printf("Fill-in: now have %d/%d concepts\n", len(allConcepts), count)
			}

			// Stop if we've reached the target
			if len(allConcepts) >= count {
				break
			}
		}
	}

	if len(allConcepts) == 0 {
		return nil, fmt.Errorf("failed to generate any concepts")
	}

	// Trim to exact count if we somehow got more
	if len(allConcepts) > count {
		allConcepts = allConcepts[:count]
	}

	// Re-assign sequential card IDs
	for i := range allConcepts {
		allConcepts[i].CardID = fmt.Sprintf("card-%03d", i+1)
	}

	if len(allConcepts) < count {
		fmt.Printf("Warning: only generated %d/%d concepts after all attempts\n", len(allConcepts), count)
	} else {
		fmt.Printf("Successfully generated all %d concepts\n", count)
	}

	return allConcepts, nil
}

// generateConceptBatch generates a small batch of concepts
func (g *Generator) generateConceptBatch(theme string, count int, startIndex int) ([]CardConcept, error) {
	prompt := fmt.Sprintf(`Generate %d card concepts for a fantasy card game. Theme: "%s"

IMPORTANT: Output ONLY a valid JSON array. No explanation, no preamble, just JSON.

Each object needs: "title" (2-4 words), "prompt" (Stable Diffusion prompt with art style, mood, colors), "tags" (3-5 tags).

Example:
[{"title":"Moonlit Wanderer","prompt":"fantasy art, mysterious figure walking through enchanted forest at night, silver moonlight, ethereal mist, deep blues","tags":["night","forest","mystery"]}]

Generate %d concepts now (JSON only):`, count, theme, count)

	response, err := g.callOllama(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}

	// Parse the JSON response
	concepts, err := parseConcepts(response, count)
	if err != nil {
		return nil, fmt.Errorf("failed to parse concepts: %w (response preview: %.200s...)", err, response)
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
		// Try to find start of array and salvage what we can
		startIdx := strings.Index(response, "[")
		if startIdx == -1 {
			return nil, fmt.Errorf("no JSON array found in response")
		}
		jsonMatch = response[startIdx:]
	}

	var concepts []CardConcept
	if err := json.Unmarshal([]byte(jsonMatch), &concepts); err != nil {
		// Try to clean up the JSON
		cleaned := cleanJSON(jsonMatch)
		if err := json.Unmarshal([]byte(cleaned), &concepts); err != nil {
			// Try to salvage partial response by fixing truncation
			salvaged := salvageTruncatedJSON(cleaned)
			if err := json.Unmarshal([]byte(salvaged), &concepts); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}
		}
	}

	if len(concepts) == 0 {
		return nil, fmt.Errorf("no valid concepts parsed")
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

// salvageTruncatedJSON tries to fix truncated JSON arrays by finding the last complete object
func salvageTruncatedJSON(s string) string {
	// Count brace levels to find last complete object
	braceLevel := 0
	bracketLevel := 0
	lastCompleteEnd := -1

	for i, c := range s {
		switch c {
		case '[':
			bracketLevel++
		case ']':
			bracketLevel--
		case '{':
			braceLevel++
		case '}':
			braceLevel--
			if braceLevel == 0 && bracketLevel == 1 {
				// Found end of a complete object in array
				lastCompleteEnd = i
			}
		}
	}

	if lastCompleteEnd > 0 {
		// Truncate at last complete object and close the array
		return s[:lastCompleteEnd+1] + "]"
	}
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

// DeriveThemeStyle generates a consistent art style description for a theme
// This style is used to ensure visual coherence across all cards in a set
func (g *Generator) DeriveThemeStyle(theme string) (string, error) {
	prompt := fmt.Sprintf(`Given this card set theme: "%s"

Generate a short art style description (20-30 words) for all cards in this set.
Include: color palette, art style, mood, lighting, visual atmosphere.

Example output: "ethereal fantasy art, soft moonlit blues and silvers, mystical atmosphere, painterly digital illustration, dreamlike quality, soft diffused lighting"

Output ONLY the style description, nothing else.`, theme)

	response, err := g.callOllama(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to derive theme style: %w", err)
	}

	// Clean up the response - remove quotes and extra whitespace
	style := strings.TrimSpace(response)
	style = strings.Trim(style, "\"'")

	fmt.Printf("Derived theme style: %s\n", style)
	return style, nil
}

// RegenerateTitle creates a new title based on a prompt and theme
func (g *Generator) RegenerateTitle(theme, prompt string) (string, error) {
	llmPrompt := fmt.Sprintf(`Create a short, evocative title (2-4 words) for a fantasy card game card.

Theme: "%s"
Image description: "%s"

The title should be:
- Mysterious and open to interpretation
- 2-4 words maximum
- Evocative and atmospheric

Respond with ONLY the title, nothing else.`, theme, prompt)

	response, err := g.callOllama(llmPrompt)
	if err != nil {
		return "", err
	}

	// Clean up the response - remove quotes and extra whitespace
	title := strings.TrimSpace(response)
	title = strings.Trim(title, "\"'")
	return title, nil
}

// RegenerateSingleConcept creates a completely new concept for the given theme
func (g *Generator) RegenerateSingleConcept(theme, currentTitle string) (*CardConcept, error) {
	prompt := fmt.Sprintf(`You are creating a concept for a fantasy card game like Dixit.
The theme is: "%s"

Generate ONE unique card concept. It should be evocative, mysterious, and open to interpretation.
Make it different from: "%s"

Provide:
1. A short title (2-4 words)
2. A Stable Diffusion prompt to generate the image (focus on visual elements, mood, colors)
3. 3-5 tags for categorization

Format your response as a JSON object with "title", "prompt", and "tags" fields.

Example format:
{
  "title": "Moonlit Wanderer",
  "prompt": "fantasy art, mysterious figure walking through enchanted forest at night, silver moonlight filtering through ancient trees, ethereal mist, deep blues and silvers, magical atmosphere, detailed illustration",
  "tags": ["night", "forest", "mystery", "journey"]
}

Generate the card concept now:`, theme, currentTitle)

	response, err := g.callOllama(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}

	// Try to find JSON object in the response
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonMatch := jsonRegex.FindString(response)

	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	var concept CardConcept
	if err := json.Unmarshal([]byte(jsonMatch), &concept); err != nil {
		cleaned := cleanJSON(jsonMatch)
		if err := json.Unmarshal([]byte(cleaned), &concept); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	concept.Prompt = enhancePrompt(concept.Prompt)
	if len(concept.Tags) == 0 {
		concept.Tags = []string{"fantasy"}
	}

	return &concept, nil
}
