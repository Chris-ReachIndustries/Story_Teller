package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// OllamaConfig holds configuration for the Ollama client
type OllamaConfig struct {
	BaseURL         string
	Model           string        // Vision model (e.g., llava:7b) for image description
	TextModel       string        // Text model (e.g., llama3.2:3b) for decision making
	MaxTokens       int
	VisionMaxTokens int          // Max tokens for vision descriptions (shorter = faster)
	Temperature     float64
	Timeout         time.Duration
	DescribeConcurrency int       // Max concurrent vision requests (0 = default 4)
}

// OllamaClient implements the Client interface using Ollama's OpenAI-compatible API
// Uses a two-stage pipeline: vision model describes images, text model makes decisions
type OllamaClient struct {
	config     OllamaConfig
	httpClient *http.Client
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(config OllamaConfig) *OllamaClient {
	if config.Model == "" {
		config.Model = "llava:7b"
	}
	if config.TextModel == "" {
		config.TextModel = "llama3.2:3b" // Default text model for decisions
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 250
	}
	if config.VisionMaxTokens == 0 {
		config.VisionMaxTokens = 100 // Shorter descriptions = faster generation
	}
	if config.DescribeConcurrency == 0 {
		config.DescribeConcurrency = 4 // Parallel card descriptions, limit to avoid overloading Ollama
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.Timeout == 0 {
		config.Timeout = 180 * time.Second // Longer timeout for local vision inference
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

// Storytell implements Client.Storytell using two-stage pipeline:
// 1. Vision model (LLaVA) describes each card
// 2. Text model picks a card and creates a clue based on descriptions
func (c *OllamaClient) Storytell(hand []CardWithThumb) (*StorytellerResponse, error) {
	// Stage 1: Get descriptions from vision model
	descriptions, err := c.describeCards(hand)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cards: %w", err)
	}

	// Stage 2: Ask text model to pick a card and create a clue
	var promptBuilder strings.Builder
	promptBuilder.WriteString("You are the storyteller in Dixit. Here are your cards:\n\n")
	for i, desc := range descriptions {
		promptBuilder.WriteString(fmt.Sprintf("Card %d: %s\n", i+1, desc))
	}
	promptBuilder.WriteString(fmt.Sprintf(`
Your goal: give a clue so that SOME players guess your card, but not everyone (ideal: 1 to (players-2) correct guesses). You want an interesting split vote.

Clue rules:
- Do NOT literally describe the image or list objects ("a cat", "a tower", "a moon").
- Do NOT mention colors or composition. No emojis, no quotes.
- Use: metaphor, emotion, theme, atmosphere, contradiction, or a vague cultural echo (myth, proverb, story vibe).
- Length: 2–4 words (max 4 words).

Strategy:
- If your chosen card feels very distinctive, pick a clue that could also fit 1–2 other cards.
- If your hand is similar overall, pick a clue that subtly points to your card.

Pick exactly ONE card (1-%d) and give exactly ONE clue.

Respond in this format only:
Card: [number]
Clue: [your clue]`, len(hand)))

	respBody, err := c.sendTextRequest(promptBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("text model failed: %w", err)
	}

	// Parse the response
	var response StorytellerResponse
	cardNum := extractCardNumber(respBody, len(hand))
	clue := extractClue(respBody)

	if cardNum > 0 {
		response.SelectedCard = cardNum
		if clue != "" {
			response.Clue = clue
		} else {
			response.Clue = pickFallbackClue()
		}
	} else {
		return nil, fmt.Errorf("failed to parse response: %s", respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}
	if response.Clue == "" {
		response.Clue = pickFallbackClue()
	}

	return &response, nil
}

// Submit implements Client.Submit using two-stage pipeline
func (c *OllamaClient) Submit(hand []CardWithThumb, clue string) (*SubmitResponse, error) {
	// Stage 1: Get descriptions from vision model
	descriptions, err := c.describeCards(hand)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cards: %w", err)
	}

	// Stage 2: Ask text model to pick a card that matches the clue
	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("You are playing Dixit (you are NOT the storyteller). The storyteller's clue is: \"%s\"\n\nYour cards:\n\n", clue))
	for i, desc := range descriptions {
		promptBuilder.WriteString(fmt.Sprintf("Card %d: %s\n", i+1, desc))
	}
	promptBuilder.WriteString(fmt.Sprintf(`
Your goal: submit ONE card (1-%d) that could attract votes as a believable match to the clue. You want to be a plausible decoy, not obviously the storyteller's card.

Think: mood, symbolism, implied story, emotional tone. Interpret the clue as metaphor or theme, not literal description.
- If the clue is abstract: pick a card with strong atmosphere or symbolism.
- If the clue hints at a narrative: pick a card that suggests a similar story.
- Avoid being too perfect (obvious) or too random (no one will vote for it).

Which card do you submit?

Respond with only the card number, like:
Card: 2`, len(hand), clue))

	respBody, err := c.sendTextRequest(promptBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("text model failed: %w", err)
	}

	// Parse the response
	var response SubmitResponse
	cardNum := extractCardNumber(respBody, len(hand))
	if cardNum > 0 {
		response.SelectedCard = cardNum
	} else {
		return nil, fmt.Errorf("failed to parse response: %s", respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}

	return &response, nil
}

// Vote implements Client.Vote using two-stage pipeline
func (c *OllamaClient) Vote(submissions []CardWithThumb, clue string, ownIndex int) (*VoteResponse, error) {
	// Stage 1: Get descriptions from vision model
	descriptions, err := c.describeCards(submissions)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cards: %w", err)
	}

	// Stage 2: Ask text model to vote for the best matching card
	var promptBuilder strings.Builder
	promptBuilder.WriteString(fmt.Sprintf("You are playing Dixit. The storyteller's clue is: \"%s\"\n\nCards on the table:\n\n", clue))
	for i, desc := range descriptions {
		if i+1 == ownIndex {
			promptBuilder.WriteString(fmt.Sprintf("Card %d (YOUR CARD - you cannot vote for this): %s\n", i+1, desc))
		} else {
			promptBuilder.WriteString(fmt.Sprintf("Card %d: %s\n", i+1, desc))
		}
	}
	promptBuilder.WriteString(fmt.Sprintf(`
Vote for the card (1-%d) you think is the STORYTELLER'S card. You CANNOT vote for Card %d (your own card).

How to decide:
- Interpret the clue as the storyteller meant it: metaphor, mood, theme, indirect reference—not literal description.
- Prefer the card that feels like it *inspired* the clue (creative, central, evocative) over one that merely "fits".
- If several fit, choose the one with the most storyteller-like intent.

Respond with only the card number, like:
Card: 2`, len(submissions), ownIndex))

	respBody, err := c.sendTextRequest(promptBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("text model failed: %w", err)
	}

	// Parse the response
	var response VoteResponse
	cardNum := extractCardNumber(respBody, len(submissions))
	if cardNum > 0 && cardNum != ownIndex {
		response.SelectedCard = cardNum
	} else if cardNum == ownIndex {
		// Model picked its own card, pick first valid alternative
		for i := 1; i <= len(submissions); i++ {
			if i != ownIndex {
				response.SelectedCard = i
				break
			}
		}
	} else {
		return nil, fmt.Errorf("failed to parse response: %s", respBody)
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

// sendRequest sends a request to the Ollama API and returns the response text.
// Uses VisionMaxTokens for vision requests to keep descriptions short and fast.
func (c *OllamaClient) sendRequest(content []interface{}) (string, error) {
	maxTok := c.config.VisionMaxTokens
	if maxTok <= 0 {
		maxTok = c.config.MaxTokens
	}
	reqBody := ollamaRequest{
		Model: c.config.Model,
		Messages: []ollamaMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   maxTok,
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

// sendTextRequest sends a text-only request to the text model
func (c *OllamaClient) sendTextRequest(prompt string) (string, error) {
	content := []interface{}{
		ollamaTextContent{Type: "text", Text: prompt},
	}

	reqBody := ollamaRequest{
		Model: c.config.TextModel, // Use text model instead of vision model
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

	url := c.config.BaseURL + "/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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

// describeCard uses the vision model to describe a single card image
func (c *OllamaClient) describeCard(card CardWithThumb) (string, error) {
	content := []interface{}{
		ollamaImageContent{
			Type: "image_url",
			ImageURL: ollamaImgURL{
				URL:    card.DataURL,
				Detail: "low",
			},
		},
		ollamaTextContent{Type: "text", Text: "In one short sentence describe this fantasy card: main subject, mood, key symbols, and the atmosphere or feeling it gives (e.g. dreamlike, ominous, hopeful). No literal list of objects—focus on what it evokes."},
	}

	return c.sendRequest(content)
}

// describeCards describes multiple cards in parallel (up to DescribeConcurrency at a time).
func (c *OllamaClient) describeCards(cards []CardWithThumb) ([]string, error) {
	descriptions := make([]string, len(cards))
	limit := c.config.DescribeConcurrency
	if limit <= 0 {
		limit = 4
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i, card := range cards {
		wg.Add(1)
		go func(idx int, card CardWithThumb) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			desc, err := c.describeCard(card)
			if err != nil {
				descriptions[idx] = "A fantasy illustration"
			} else {
				descriptions[idx] = strings.TrimSpace(desc)
			}
		}(i, card)
	}
	wg.Wait()
	return descriptions, nil
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

// extractClue attempts to extract a clue from free-form AI response.
// Returns the first valid 2-4 word clue found; trims to one line and max 4 words when needed.
func extractClue(text string) string {
	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.TrimRight(s, ".,!?;")
		// Strip brackets if model used "Clue: [something]"
		if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
			s = strings.TrimSpace(s[1 : len(s)-1])
		}
		return s
	}
	wordCount := func(s string) int { return len(strings.Fields(s)) }
	trimToMaxWords := func(s string, max int) string {
		words := strings.Fields(s)
		if len(words) <= max {
			return strings.Join(words, " ")
		}
		return strings.Join(words[:max], " ")
	}

	// Pattern 1: "clue": "X" or clue: "X" (JSON-ish or quoted)
	cluePattern := regexp.MustCompile(`(?i)["']?clue["']?\s*[":]\s*["']([^"']+)["']`)
	if matches := cluePattern.FindStringSubmatch(text); len(matches) > 1 {
		clue := normalize(matches[1])
		clue = trimToMaxWords(clue, 4)
		if len(clue) >= 2 && wordCount(clue) >= 2 && wordCount(clue) <= 4 {
			return clue
		}
	}

	// Pattern 2: "Clue:" or "Clue -" followed by clue on same line or next line
	colonPattern := regexp.MustCompile(`(?i)clue(?:\s+is)?[\s:\-–—]+\s*["']?([^"'\n]+)["']?`)
	if matches := colonPattern.FindStringSubmatch(text); len(matches) > 1 {
		clue := normalize(matches[1])
		// Take first line only in case model added explanation
		if idx := strings.Index(clue, "\n"); idx >= 0 {
			clue = strings.TrimSpace(clue[:idx])
		}
		clue = trimToMaxWords(clue, 4)
		if len(clue) >= 2 && wordCount(clue) >= 2 && wordCount(clue) <= 4 {
			return clue
		}
	}

	// Pattern 2b: Clue on its own line after "Clue:" (e.g. "Clue:\n  lost in the woods")
	nextLinePattern := regexp.MustCompile(`(?i)clue[\s:\-–—]*\n\s*([^\n]+)`)
	if matches := nextLinePattern.FindStringSubmatch(text); len(matches) > 1 {
		clue := normalize(matches[1])
		clue = trimToMaxWords(clue, 4)
		if len(clue) >= 2 && wordCount(clue) >= 2 && wordCount(clue) <= 4 {
			return clue
		}
	}

	// Pattern 3: Quoted text that looks like a clue (2-4 words)
	quotePattern := regexp.MustCompile(`["']([^"']{5,80})["']`)
	if matches := quotePattern.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, m := range matches {
			clue := normalize(m[1])
			if wordCount(clue) >= 2 && wordCount(clue) <= 4 {
				return clue
			}
		}
	}

	// Pattern 4: Any line that is just 2-4 words (not "Card: N") — last resort
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = normalize(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "card") {
			continue
		}
		words := strings.Fields(line)
		if len(words) >= 2 && len(words) <= 4 && len(line) <= 50 {
			return line
		}
	}

	return ""
}

// fallbackClues are used when the model's clue cannot be parsed (avoids always "mysterious journey").
var fallbackClues = []string{
	"mysterious journey", "hidden meaning", "lost in dreams", "between two worlds",
	"what the heart sees", "echoes of memory", "the path unseen", "a moment suspended",
}

// pickFallbackClue returns a random 2–4 word fallback clue when extraction fails.
func pickFallbackClue() string {
	return fallbackClues[rand.Intn(len(fallbackClues))]
}
