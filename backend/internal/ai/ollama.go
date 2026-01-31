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
		config.Timeout = 60 * time.Second // Longer timeout for local inference
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
	prompt := `You are playing Dixit. You are the storyteller this round.

Objective:
Create a clue that makes SOME (not all, not none) players guess your card.
You are trying to engineer an interesting split vote (ideally 1–(players-2) correct guesses).

Clue rules:
- Do NOT literally describe the image or list visible objects ("a cat", "a tower", "a moon", etc.).
- Do NOT mention colors, composition, or camera framing ("top-left", "close up").
- Avoid unique proper nouns that directly identify the card. Indirect references are okay (myth, proverb, classic story vibe).
- Prefer: metaphor, emotion, theme, relationship, contradiction, atmosphere, or a vague cultural echo.
- Clue length: 2–10 words. No emojis. No quotes.

Strategy:
- If your card feels very distinctive, choose a clue that could plausibly match 1–2 other cards.
- If your hand is uniformly similar, choose a clue that distinguishes your chosen card subtly.

Task:
1) Choose exactly ONE card from your hand.
2) Provide exactly ONE clue.

Return ONLY valid JSON (no markdown, no extra keys):
{"selectedCard": <1-N>, "clue": "<text>"}

Self-check before responding:
- selectedCard is a valid number 1..N
- clue obeys the rules above`

	content := []interface{}{
		ollamaTextContent{Type: "text", Text: prompt},
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

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanOllamaJSONResponse(respBody)

	var response StorytellerResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}
	if response.Clue == "" {
		return nil, fmt.Errorf("empty clue in response")
	}

	return &response, nil
}

// Submit implements Client.Submit
func (c *OllamaClient) Submit(hand []CardWithThumb, clue string) (*SubmitResponse, error) {
	prompt := fmt.Sprintf(`You are playing Dixit. You are NOT the storyteller.

The storyteller's clue is: "%s"

Objective:
Submit ONE card from your hand that will attract votes by seeming like it could be the storyteller's card.

Important:
- You are trying to be a believable decoy, not the "best match" in a literal sense.
- Think: mood, symbolism, implied story, emotional tone, genre.

Strategy:
- If the clue is abstract: pick a card with strong atmosphere or symbolism.
- If the clue hints at a narrative: pick a card that suggests a similar story arc.
- Avoid being too perfect (obvious storyteller) or too random (no votes).

Return ONLY valid JSON (no markdown, no extra keys):
{"selectedCard": <1-N>}

Self-check:
- selectedCard is 1..N`, clue)

	content := []interface{}{
		ollamaTextContent{Type: "text", Text: prompt},
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

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanOllamaJSONResponse(respBody)

	var response SubmitResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}

	return &response, nil
}

// Vote implements Client.Vote
func (c *OllamaClient) Vote(submissions []CardWithThumb, clue string, ownIndex int) (*VoteResponse, error) {
	prompt := fmt.Sprintf(`You are playing Dixit. You are voting for which card is the STORYTELLER'S.

The storyteller's clue is: "%s"

Look at all the submitted cards and vote for the one you think is the STORYTELLER'S card (not your own).

IMPORTANT: Card %d is YOUR card - you cannot vote for it! Choose a different card.

How to decide:
- Interpret the clue as the storyteller intended: metaphor, mood, theme, indirect reference.
- Prefer the card that feels like it inspired the clue rather than one that merely "fits".
- If multiple cards fit, choose the one with the most storyteller-like intent (creative, central, evocative).

Return ONLY valid JSON (no markdown, no extra keys):
{"selectedCard": <1-N>}

Self-check:
- selectedCard is 1..N and not %d (your own card)
Remember: You CANNOT vote for card %d (your own card).`, clue, ownIndex, ownIndex, ownIndex)

	content := []interface{}{
		ollamaTextContent{Type: "text", Text: prompt},
	}

	for i, card := range submissions {
		label := fmt.Sprintf("Card %d:", i+1)
		if i+1 == ownIndex {
			label = fmt.Sprintf("Card %d (YOUR CARD - cannot vote):", i+1)
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

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanOllamaJSONResponse(respBody)

	var response VoteResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
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
