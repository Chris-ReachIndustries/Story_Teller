package ai

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

// OllamaConfig holds configuration for the Ollama client
type OllamaConfig struct {
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// OllamaClient implements the Client interface using Ollama's OpenAI-compatible API
type OllamaClient struct {
	config     OllamaConfig
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(config OllamaConfig) *OllamaClient {
	if config.Model == "" {
		config.Model = "llava:7b"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 250
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.Timeout == 0 {
		config.Timeout = 180 * time.Second // Longer timeout for local vision inference (5 images)
	}
	if config.BaseURL == "" {
		config.BaseURL = "http://ollama:11434"
	}

	return &OllamaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Ping checks if Ollama is reachable
func (c *OllamaClient) Ping() error {
	resp, err := c.httpClient.Get(c.config.BaseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

// Ollama uses the same request structures as OpenAI (OpenAI-compatible API)
type ollamaRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type ollamaTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ollamaImageContent struct {
	Type     string       `json:"type"`
	ImageURL ollamaImgURL `json:"image_url"`
}

type ollamaImgURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ollamaResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Storytell implements Client.Storytell
func (c *OllamaClient) Storytell(hand []CardWithThumb) (*StorytellerResponse, error) {
	// Build content with images FIRST, then question at the END
	// LLaVA responds better to "look at images, then answer" format
	content := []interface{}{
		ollamaTextContent{Type: "text", Text: "Here are your cards for Dixit:"},
	}

	for i, card := range hand {
		content = append(content,
			ollamaTextContent{Type: "text", Text: fmt.Sprintf("Card %d:", i+1)},
			ollamaImageContent{
				Type: "image_url",
				ImageURL: ollamaImgURL{
					URL:    card.DataURL,
					Detail: "low",
				},
			},
		)
	}

	// Add the question AFTER all images - LLaVA follows this better
	question := fmt.Sprintf(`You must pick ONE card (1-%d) and give a short clue (2-5 words) that hints at it without being obvious.

Answer with ONLY: Card X, clue: "your clue"
Example: Card 2, clue: "dreams of tomorrow"

Which card do you pick and what is your clue?`, len(hand))
	content = append(content, ollamaTextContent{Type: "text", Text: question})

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanOllamaJSONResponse(respBody)

	var response StorytellerResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		// Try fallback parsing for non-JSON responses
		cardNum := extractCardNumber(respBody, len(hand))
		clue := extractClue(respBody)

		if cardNum > 0 {
			response.SelectedCard = cardNum
			if clue != "" {
				response.Clue = clue
			} else {
				// Generate a generic clue if model didn't provide one
				response.Clue = "mysterious journey"
			}
		} else {
			return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
		}
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}
	if response.Clue == "" {
		response.Clue = "hidden meaning"
	}

	return &response, nil
}

// Submit implements Client.Submit
func (c *OllamaClient) Submit(hand []CardWithThumb, clue string) (*SubmitResponse, error) {
	// Build content with images FIRST, then question at the END
	content := []interface{}{
		ollamaTextContent{Type: "text", Text: "Here are your cards:"},
	}

	for i, card := range hand {
		content = append(content,
			ollamaTextContent{Type: "text", Text: fmt.Sprintf("Card %d:", i+1)},
			ollamaImageContent{
				Type: "image_url",
				ImageURL: ollamaImgURL{
					URL:    card.DataURL,
					Detail: "low",
				},
			},
		)
	}

	// Add the question AFTER all images
	question := fmt.Sprintf(`The clue is: "%s"

Which card (1-%d) best matches this clue? Consider mood and theme, not literal meaning.

Answer with ONLY the card number, like: Card 3`, clue, len(hand))
	content = append(content, ollamaTextContent{Type: "text", Text: question})

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanOllamaJSONResponse(respBody)

	var response SubmitResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		// Try fallback parsing for non-JSON responses
		cardNum := extractCardNumber(respBody, len(hand))
		if cardNum > 0 {
			response.SelectedCard = cardNum
		} else {
			return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
		}
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}

	return &response, nil
}

// Vote implements Client.Vote
func (c *OllamaClient) Vote(submissions []CardWithThumb, clue string, ownIndex int) (*VoteResponse, error) {
	// Build content with images FIRST, then question at the END
	content := []interface{}{
		ollamaTextContent{Type: "text", Text: "Here are the cards on the table:"},
	}

	for i, card := range submissions {
		label := fmt.Sprintf("Card %d:", i+1)
		if i+1 == ownIndex {
			label = fmt.Sprintf("Card %d (this is YOUR card):", i+1)
		}
		content = append(content,
			ollamaTextContent{Type: "text", Text: label},
			ollamaImageContent{
				Type: "image_url",
				ImageURL: ollamaImgURL{
					URL:    card.DataURL,
					Detail: "low",
				},
			},
		)
	}

	// Add the question AFTER all images
	question := fmt.Sprintf(`The clue is: "%s"

Which card (1-%d) do you think the storyteller chose? You cannot pick Card %d (your own card).

Answer with ONLY the card number, like: Card 2`, clue, len(submissions), ownIndex)
	content = append(content, ollamaTextContent{Type: "text", Text: question})

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanOllamaJSONResponse(respBody)

	var response VoteResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		// Try fallback parsing for non-JSON responses
		cardNum := extractCardNumber(respBody, len(submissions))
		if cardNum > 0 && cardNum != ownIndex {
			response.SelectedCard = cardNum
		} else if cardNum == ownIndex {
			// Model picked its own card, try to find another number
			// Default to first valid card that isn't own
			for i := 1; i <= len(submissions); i++ {
				if i != ownIndex {
					response.SelectedCard = i
					break
				}
			}
		} else {
			return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
		}
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(submissions) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(submissions))
	}
	if response.SelectedCard == ownIndex {
		return nil, fmt.Errorf("AI voted for its own card (%d)", ownIndex)
	}

	return &response, nil
}

// sendRequest sends a request to the Ollama API and returns the response text
func (c *OllamaClient) sendRequest(content []interface{}) (string, error) {
	reqBody := ollamaRequest{
		Model: c.config.Model,
		Messages: []ollamaMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Ollama's OpenAI-compatible endpoint
	url := c.config.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// No Authorization header needed for Ollama

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w (status: %d, body: %s)", err, resp.StatusCode, string(body))
	}

	if ollamaResp.Error != nil {
		return "", fmt.Errorf("Ollama API error: %s", ollamaResp.Error.Message)
	}

	if len(ollamaResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in Ollama response")
	}

	return ollamaResp.Choices[0].Message.Content, nil
}

// cleanOllamaJSONResponse strips markdown code blocks and extra whitespace from AI responses
func cleanOllamaJSONResponse(s string) string {
	// Remove markdown code blocks (```json ... ``` or ``` ... ```)
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	if matches := re.FindStringSubmatch(s); len(matches) > 1 {
		s = matches[1]
	}

	// Trim whitespace
	s = strings.TrimSpace(s)

	return s
}

// extractCardNumber attempts to extract a card number from free-form AI response
// when the model doesn't return proper JSON. Returns 0 if no card found.
func extractCardNumber(text string, maxCard int) int {
	text = strings.ToLower(text)

	// Pattern 1: "card X" or "Card X" (most common)
	cardPattern := regexp.MustCompile(`card\s*[#:]?\s*(\d+)`)
	if matches := cardPattern.FindStringSubmatch(text); len(matches) > 1 {
		if num := parseCardNum(matches[1], maxCard); num > 0 {
			return num
		}
	}

	// Pattern 2: "selectedCard": X or selectedcard: X (JSON-ish)
	selectedPattern := regexp.MustCompile(`["']?selectedcard["']?\s*[":]\s*(\d+)`)
	if matches := selectedPattern.FindStringSubmatch(text); len(matches) > 1 {
		if num := parseCardNum(matches[1], maxCard); num > 0 {
			return num
		}
	}

	// Pattern 3: "choose X" or "select X" or "pick X"
	choosePattern := regexp.MustCompile(`(?:choose|select|pick|vote\s+for)\s+(?:card\s+)?#?(\d+)`)
	if matches := choosePattern.FindStringSubmatch(text); len(matches) > 1 {
		if num := parseCardNum(matches[1], maxCard); num > 0 {
			return num
		}
	}

	// Pattern 4: "option X" or "#X"
	optionPattern := regexp.MustCompile(`(?:option|#)\s*(\d+)`)
	if matches := optionPattern.FindStringSubmatch(text); len(matches) > 1 {
		if num := parseCardNum(matches[1], maxCard); num > 0 {
			return num
		}
	}

	// Pattern 5: Written numbers (first, second, etc.)
	ordinals := map[string]int{
		"first": 1, "second": 2, "third": 3, "fourth": 4,
		"fifth": 5, "sixth": 6, "seventh": 7, "eighth": 8,
	}
	for word, num := range ordinals {
		if strings.Contains(text, word) && num <= maxCard {
			return num
		}
	}

	// Pattern 6: Just find any single digit that could be a card number
	// Only if response is short (likely a simple answer)
	if len(text) < 200 {
		digitPattern := regexp.MustCompile(`\b(\d)\b`)
		if matches := digitPattern.FindStringSubmatch(text); len(matches) > 1 {
			if num := parseCardNum(matches[1], maxCard); num > 0 {
				return num
			}
		}
	}

	return 0
}

// parseCardNum safely parses a string to int and validates range
func parseCardNum(s string, maxCard int) int {
	var num int
	if _, err := fmt.Sscanf(s, "%d", &num); err == nil {
		if num >= 1 && num <= maxCard {
			return num
		}
	}
	return 0
}

// extractClue attempts to extract a clue from free-form AI response
func extractClue(text string) string {
	// Pattern 1: "clue": "X" (JSON-ish)
	cluePattern := regexp.MustCompile(`["']?clue["']?\s*[":]\s*["']([^"']+)["']`)
	if matches := cluePattern.FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Pattern 2: After "clue:" or "clue is"
	colonPattern := regexp.MustCompile(`(?i)clue(?:\s+is)?[:\s]+["']?([^"'\n]+)["']?`)
	if matches := colonPattern.FindStringSubmatch(text); len(matches) > 1 {
		clue := strings.TrimSpace(matches[1])
		// Clean up any trailing punctuation
		clue = strings.TrimRight(clue, ".,!?")
		if len(clue) > 2 && len(clue) < 100 {
			return clue
		}
	}

	// Pattern 3: Quoted text that looks like a clue (2-10 words)
	quotePattern := regexp.MustCompile(`["']([^"']{5,50})["']`)
	if matches := quotePattern.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, m := range matches {
			clue := strings.TrimSpace(m[1])
			words := strings.Fields(clue)
			if len(words) >= 2 && len(words) <= 10 {
				return clue
			}
		}
	}

	return ""
}
